package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/centrifuge/go-centrifuge/config"
	cented25519 "github.com/centrifuge/go-centrifuge/crypto/ed25519"
	"github.com/centrifuge/go-centrifuge/errors"
	"github.com/centrifuge/go-centrifuge/identity"
	"github.com/centrifuge/go-centrifuge/p2p/common"
	ms "github.com/centrifuge/go-centrifuge/p2p/messenger"
	"github.com/centrifuge/go-centrifuge/p2p/receiver"
	pb "github.com/centrifuge/go-centrifuge/protobufs/gen/go/protocol"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs-addr"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	libp2pPeer "github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

var log = logging.Logger("p2p-server")

// messenger is an interface to wrap p2p messaging implementation
type messenger interface {

	// Init inits the messenger
	Init(protocols ...protocol.ID)

	// SendMessage sends a message through messenger
	SendMessage(ctx context.Context, p libp2pPeer.ID, pmes *pb.P2PEnvelope, protoc protocol.ID) (*pb.P2PEnvelope, error)
}

// peer implements node.Server
type peer struct {
	disablePeerStore bool
	config           config.Service
	idService        identity.ServiceDID
	host             host.Host
	handlerCreator   func() *receiver.Handler
	mes              messenger
}

// Name returns the P2PServer
func (*peer) Name() string {
	return "P2PServer"
}

// Start starts the DHT and libp2p host
func (s *peer) Start(ctx context.Context, wg *sync.WaitGroup, startupErr chan<- error) {
	defer wg.Done()

	nc, err := s.config.GetConfig()
	if err != nil {
		startupErr <- err
		return
	}

	if nc.GetP2PPort() == 0 {
		startupErr <- errors.New("please provide a port to bind on")
		return
	}

	// Make a host that listens on the given multiaddress
	// first obtain the keys configured
	priv, pub, err := s.createSigningKey(nc.GetP2PKeyPair())
	if err != nil {
		startupErr <- err
		return
	}
	s.host, err = makeBasicHost(priv, pub, nc.GetP2PExternalIP(), nc.GetP2PPort())
	if err != nil {
		startupErr <- err
		return
	}

	s.mes = ms.NewP2PMessenger(ctx, s.host, nc.GetP2PConnectionTimeout(), s.handlerCreator().HandleInterceptor)
	err = s.initProtocols()
	if err != nil {
		startupErr <- err
		return
	}

	// Start DHT and properly ignore errors :)
	_ = runDHT(ctx, s.host, nc.GetBootstrapPeers())
	<-ctx.Done()

}

func (s *peer) initProtocols() error {
	tcs, err := s.config.GetAllAccounts()
	if err != nil {
		return err
	}
	var protocols []protocol.ID
	for _, t := range tcs {
		accID, err := t.GetIdentityID()
		if err != nil {
			return err
		}
		DID := identity.NewDIDFromBytes(accID)
		protocols = append(protocols, p2pcommon.ProtocolForDID(&DID))
	}
	s.mes.Init(protocols...)
	return nil
}

func (s *peer) InitProtocolForDID(DID *identity.DID) {
	p := p2pcommon.ProtocolForDID(DID)
	s.mes.Init(p)
}

func (s *peer) createSigningKey(pubKey, privKey string) (priv crypto.PrivKey, pub crypto.PubKey, err error) {
	// Create the signing key for the host
	publicKey, privateKey, err := cented25519.GetSigningKeyPair(pubKey, privKey)
	if err != nil {
		return nil, nil, errors.New("failed to get keys: %v", err)
	}

	var key []byte
	key = append(key, privateKey...)
	key = append(key, publicKey...)

	priv, err = crypto.UnmarshalEd25519PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}

	pub = priv.GetPublic()
	return priv, pub, nil
}

// makeBasicHost creates a LibP2P host with a peer ID listening on the given port
func makeBasicHost(priv crypto.PrivKey, pub crypto.PubKey, externalIP string, listenPort int) (host.Host, error) {

	libp2pPeer.AdvancedEnableInlining = false
	// Obtain Peer ID from public key
	// We should be using the following method to get the ID, but looks like is not compatible with
	// secio when adding the pub and pvt keys, fail as id+pub/pvt key is checked to match and method defaults to
	// IDFromPublicKey(pk)
	//pid, err := peer.IDFromEd25519PublicKey(pub)
	pid, err := libp2pPeer.IDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}

	// Create a peerstore
	ps := pstore.NewPeerstore(
		pstoremem.NewKeyBook(),
		pstoremem.NewAddrBook(),
		pstoremem.NewPeerMetadata())

	// Add the keys to the peerstore
	// for this peer ID.
	err = ps.AddPubKey(pid, pub)
	if err != nil {
		log.Infof("Could not enable encryption: %v\n", err)
		return nil, err
	}

	err = ps.AddPrivKey(pid, priv)
	if err != nil {
		log.Infof("Could not enable encryption: %v\n", err)
		return nil, err
	}

	var extMultiAddr ma.Multiaddr
	if externalIP == "" {
		log.Warning("External IP not defined, Peers might not be able to resolve this node if behind NAT\n")
	} else {
		extMultiAddr, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", externalIP, listenPort))
		if err != nil {
			return nil, errors.New("failed to create multiaddr: %v", err)
		}
	}

	addressFactory := func(addrs []ma.Multiaddr) []ma.Multiaddr {
		if extMultiAddr != nil {
			// We currently support a single protocol and transport, if we add more to support then we will need to adapt this code
			addrs = []ma.Multiaddr{extMultiAddr}
		}
		return addrs
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DefaultMuxers,
		libp2p.AddrsFactory(addressFactory),
	}

	bhost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", bhost.ID().Pretty()))
	if err != nil {
		return nil, errors.New("failed to get addr: %v", err)
	}

	log.Infof("P2P Server at: %s %s\n", hostAddr.String(), bhost.Addrs())
	return bhost, nil
}

func runDHT(ctx context.Context, h host.Host, bootstrapPeers []string) error {
	// Run it as a Bootstrap Node
	dhtClient := dht.NewDHT(ctx, h, ds.NewMapDatastore())
	log.Infof("Bootstrapping %s\n", bootstrapPeers)

	for _, addr := range bootstrapPeers {
		iaddr, _ := ipfsaddr.ParseString(addr)
		pinfo, _ := pstore.InfoFromP2pAddr(iaddr.Multiaddr())
		if err := h.Connect(ctx, *pinfo); err != nil {
			log.Info("Bootstrapping to peer failed: ", err)
		}
	}

	// Using the sha256 of our "topic" as our rendezvous value
	cidPref, _ := cid.NewPrefixV1(cid.Raw, mh.SHA2_256).Sum([]byte("centrifuge-dht"))

	// First, announce ourselves as participating in this topic
	log.Info("Announcing ourselves...")
	tctx, cancel := context.WithTimeout(ctx, time.Second*10)
	if err := dhtClient.Provide(tctx, cidPref, true); err != nil {
		// Important to keep this as Non-Fatal error, otherwise it will fail for a node that behaves as well as bootstrap one
		log.Infof("Error: %s\n", err.Error())
	}
	cancel()

	// Now, look for others who have announced
	log.Info("Searching for other peers ...")
	tctx, cancel = context.WithTimeout(ctx, time.Second*10)
	peers, err := dhtClient.FindProviders(tctx, cidPref)
	if err != nil {
		log.Error(err)
	}
	cancel()
	log.Infof("Found %d peers!\n", len(peers))

	// Now connect to them, so they are added to the PeerStore
	for _, pe := range peers {
		log.Infof("Peer %s %s\n", pe.ID.Pretty(), pe.Addrs)

		if pe.ID == h.ID() {
			// No sense connecting to ourselves
			continue
		}

		tctx, cancel := context.WithTimeout(ctx, time.Second*5)
		if err := h.Connect(tctx, pe); err != nil {
			log.Info("Failed to connect to peer: ", err)
		}
		cancel()
	}

	log.Info("Bootstrapping and discovery complete!")
	return nil
}
