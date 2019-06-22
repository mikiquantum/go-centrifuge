package nft

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/centrifuge/go-centrifuge/config"
	"github.com/centrifuge/go-centrifuge/snarks"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/centrifuge/go-centrifuge/anchors"
	"github.com/centrifuge/go-centrifuge/contextutil"
	"github.com/centrifuge/go-centrifuge/documents"
	"github.com/centrifuge/go-centrifuge/errors"
	"github.com/centrifuge/go-centrifuge/ethereum"
	"github.com/centrifuge/go-centrifuge/identity"
	"github.com/centrifuge/go-centrifuge/jobs"
	"github.com/centrifuge/go-centrifuge/queue"
	"github.com/centrifuge/go-centrifuge/utils"
	"github.com/centrifuge/go-centrifuge/utils/byteutils"
	"github.com/centrifuge/go-centrifuge/utils/stringutils"
	"github.com/centrifuge/precise-proofs/proofs/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("nft")

const (
	// ErrNFTMinted error for NFT already minted for registry
	ErrNFTMinted = errors.Error("NFT already minted")

	ABIZKNFT = "[{\"constant\":true,\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"getApproved\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"uri_prefix\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"what\",\"type\":\"bytes32\"},{\"name\":\"data_\",\"type\":\"string\"}],\"name\":\"file\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"from\",\"type\":\"address\"},{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"owner\",\"type\":\"address\"},{\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"tokenOfOwnerByIndex\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"ratings\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"from\",\"type\":\"address\"},{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"a\",\"type\":\"uint256[2]\"},{\"name\":\"b\",\"type\":\"uint256[2][2]\"},{\"name\":\"c\",\"type\":\"uint256[2]\"},{\"name\":\"input\",\"type\":\"uint256[7]\"}],\"name\":\"verifyTx\",\"outputs\":[{\"name\":\"r\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"tokenByIndex\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"x\",\"type\":\"bytes32\"}],\"name\":\"unpack\",\"outputs\":[{\"name\":\"y\",\"type\":\"uint256\"},{\"name\":\"z\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"anchors\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"from\",\"type\":\"address\"},{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"tokenURI\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"usr\",\"type\":\"address\"},{\"name\":\"tkn\",\"type\":\"uint256\"},{\"name\":\"anchor\",\"type\":\"uint256\"},{\"name\":\"data_root\",\"type\":\"bytes32\"},{\"name\":\"signatures_root\",\"type\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"rating\",\"type\":\"uint256\"},{\"name\":\"points\",\"type\":\"uint256[8]\"}],\"name\":\"mint\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"data_root\",\"type\":\"bytes32\"},{\"name\":\"nft_amount\",\"type\":\"uint256\"},{\"name\":\"rating\",\"type\":\"uint256\"},{\"name\":\"points\",\"type\":\"uint256[8]\"}],\"name\":\"verify\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"owner\",\"type\":\"address\"},{\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"what\",\"type\":\"bytes32\"},{\"name\":\"data_\",\"type\":\"bytes32\"}],\"name\":\"file\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"uri\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"data\",\"outputs\":[{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"anchor\",\"type\":\"uint256\"},{\"name\":\"rating\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"anchor\",\"type\":\"uint256\"},{\"name\":\"droot\",\"type\":\"bytes32\"},{\"name\":\"sigs\",\"type\":\"bytes32\"}],\"name\":\"checkAnchor\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"symbol\",\"type\":\"string\"},{\"name\":\"anchors_\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"approved\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"s\",\"type\":\"string\"}],\"name\":\"Verified\",\"type\":\"event\"}]"
)

// structs for zk JSON
type Proof struct {
	Hashes []string `json:"hashes"`
	Right []bool `json:"right"`
	Value string `json:"value"`
	Salt string  `json:"salt,omitempty"`
	Property string `json:"property,omitempty"`
}
type PublicFields struct {
	NFTAmount string `json:"nft_amount"`
	CreditRatingRootHash string `json:"credit_rating_roothash"`
	Rating string `json:"rating"`
	DocumentRootHash string `json:"document_roothash"`
	SignaturesRootHash string `json:"signatures_roothash"`
}
type PrivateFields struct {
	BuyerPubKey string `json:"buyer_pubkey"`
	BuyerSignature string `json:"buyer_signature"`
	BuyerRatingProof Proof `json:"buyer_rating_proof"`
	DocumentInvoiceAmountProof Proof `json:"document_invoice_amount_proof"`
	DocumentInvoiceBuyerProof Proof `json:"document_invoice_buyer_proof"`
}
type ZKJSON struct {
	Public PublicFields `json:"public"`
	Private PrivateFields `json:"private"`
}


// Config is the config interface for nft package
type Config interface {
	GetEthereumContextWaitTimeout() time.Duration
	GetLowEntropyNFTTokenEnabled() bool
	GetEthereumGasLimit(op config.ContractOp) uint64
	GetContractAddress(contractName config.ContractName) common.Address
}

// ethInvoiceUnpaid handles all interactions related to minting of NFTs for unpaid invoices on Ethereum
type ethInvoiceUnpaid struct {
	cfg             Config
	identityService identity.Service
	ethClient       ethereum.Client
	queue           queue.TaskQueuer
	docSrv          documents.Service
	bindContract    func(address common.Address, client ethereum.Client) (*InvoiceUnpaidContract, error)
	jobsManager     jobs.Manager
	blockHeightFunc func() (height uint64, err error)
}

// newEthInvoiceUnpaid creates InvoiceUnpaid given the parameters
func newEthInvoiceUnpaid(
	cfg Config,
	identityService identity.Service,
	ethClient ethereum.Client,
	queue queue.TaskQueuer,
	docSrv documents.Service,
	bindContract func(address common.Address, client ethereum.Client) (*InvoiceUnpaidContract, error),
	jobsMan jobs.Manager,
	blockHeightFunc func() (uint64, error)) *ethInvoiceUnpaid {
	return &ethInvoiceUnpaid{
		cfg:             cfg,
		identityService: identityService,
		ethClient:       ethClient,
		bindContract:    bindContract,
		queue:           queue,
		docSrv:          docSrv,
		jobsManager:     jobsMan,
		blockHeightFunc: blockHeightFunc,
	}
}

// ethereumTX is submitting an Ethereum transaction and starts a task to wait for the transaction result
func (s *ethInvoiceUnpaid) ethereumTX(opts *bind.TransactOpts, contractMethod interface{}, params ...interface{}) func(accountID identity.DID, jobID jobs.JobID, jobsMan jobs.Manager, errOut chan<- error) {
	return func(accountID identity.DID, jobID jobs.JobID, jobMan jobs.Manager, errOut chan<- error) {
		ethTX, err := s.ethClient.SubmitTransactionWithRetries(contractMethod, opts, params...)
		if err != nil {
			errOut <- err
			return
		}

		res, err := ethereum.QueueEthTXStatusTask(accountID, jobID, ethTX.Hash(), s.queue)
		if err != nil {
			errOut <- err
			return
		}

		_, err = res.Get(jobMan.GetDefaultTaskTimeout())
		if err != nil {
			errOut <- err
			return
		}
		errOut <- nil
	}
}

func (s *ethInvoiceUnpaid) filterMintProofs(docProof *documents.DocumentProof) *documents.DocumentProof {
	// Compact properties
	var nonFilteredProofsLiteral = [][]byte{append(documents.CompactProperties(documents.DRTreePrefix), documents.CompactProperties(documents.DocumentDataRootField)...)}
	// Byte array Regex - (signatureTreePrefix + signatureProp) + Index[up to 104 characters (52bytes*2)]
	m0 := append(documents.CompactProperties(documents.SignaturesTreePrefix), []byte{0, 0, 0, 1}...)
	var nonFilteredProofsMatch = []string{fmt.Sprintf("%s(.{104})", hex.EncodeToString(m0))}

	for i, p := range docProof.FieldProofs {
		if !byteutils.ContainsBytesInSlice(nonFilteredProofsLiteral, p.GetCompactName()) && !stringutils.ContainsBytesMatchInSlice(nonFilteredProofsMatch, p.GetCompactName()) {
			if len(docProof.FieldProofs[i].Hashes) > 0 {
				docProof.FieldProofs[i].Hashes = docProof.FieldProofs[i].Hashes[:len(docProof.FieldProofs[i].Hashes)-1]
			} else {
				docProof.FieldProofs[i].SortedHashes = docProof.FieldProofs[i].SortedHashes[:len(docProof.FieldProofs[i].SortedHashes)-1]
			}

		}
	}
	return docProof
}

func (s *ethInvoiceUnpaid) prepareMintRequest(ctx context.Context, tokenID TokenID, cid identity.DID, req MintNFTRequest) (mreq MintRequest, zkPayload ZKJSON, err error) {
	docProofs, err := s.docSrv.CreateProofs(ctx, req.DocumentID, req.ProofFields)
	if err != nil {
		return mreq, zkPayload, err
	}

	model, err := s.docSrv.GetCurrentVersion(ctx, req.DocumentID)
	if err != nil {
		return mreq, zkPayload, err
	}

	pfs, err := model.CreateNFTProofs(cid,
		req.RegistryAddress,
		tokenID[:],
		req.SubmitTokenProof,
		req.GrantNFTReadAccess && req.SubmitNFTReadAccessProof)
	if err != nil {
		return mreq, zkPayload, err
	}

	docProofs.FieldProofs = append(docProofs.FieldProofs, pfs...)
	docProofs = s.filterMintProofs(docProofs)

	anchorID, err := anchors.ToAnchorID(model.CurrentVersion())
	if err != nil {
		return mreq, zkPayload, err
	}

	nextAnchorID, err := anchors.ToAnchorID(model.NextVersion())
	if err != nil {
		return mreq, zkPayload, err
	}

	sigRoot, err := model.CalculateSignaturesRoot()
	if err != nil {
		return mreq, zkPayload, errors.New("failed to calculate sigRoot: %v", err)
	}

	docDataRootHash, err := model.CalculateDocumentDataRoot()
	if err != nil {
		return mreq, zkPayload, errors.New("failed to calculate docRootHash: %v", err)
	}

	proof, err := documents.ConvertDocProofToClientFormat(&documents.DocumentProof{DocumentID: docProofs.DocumentID, VersionID: docProofs.VersionID, FieldProofs: docProofs.FieldProofs})
	if err != nil {
		return mreq, zkPayload, err
	}

	//log.Debug(json.MarshalIndent(proof, "", "  "))

	var buyerPubKey []byte
	var buyerSignature []byte
	signs := model.Signatures()
	for _, v := range signs {
		if utils.IsSameByteSlice(v.SignerId,docProofs.FieldProofs[1].Value) {
			fmt.Printf("%x\n", v.SignerId)
			fmt.Printf("%x\n", v.PublicKey)
			fmt.Printf("%v\n", v.TransitionValidated)
			buyerPubKey = v.PublicKey
			buyerSignature = v.Signature
		}
	}
	if len(buyerPubKey) < 1 {
		return mreq, zkPayload, errors.New("Buyer Signature not found in list")
	}

	// Call to get buyer tree proofs
	buyerProof, err := snarks.GenerateDefaultBuyerRatingProof(hex.EncodeToString(docProofs.FieldProofs[1].Value), hex.EncodeToString(buyerPubKey))
	if err != nil {
		return mreq, zkPayload, err
	}

	// Form the ZK JSON payload for prover
	zkPayload = ZKJSON{
		Public:  PublicFields{
			NFTAmount: "0000000000000000000000000000000000000000000000000000000000000320",
			CreditRatingRootHash: buyerProof.RootHash,
			Rating: "64",
			DocumentRootHash: hex.EncodeToString(docDataRootHash),
			SignaturesRootHash: hex.EncodeToString(sigRoot),
		},
		Private: PrivateFields{
			BuyerPubKey: hex.EncodeToString(buyerPubKey),
			BuyerSignature: hex.EncodeToString(buyerSignature),
			BuyerRatingProof: Proof{
				Hashes: buyerProof.Hashes,
				Right: buyerProof.Right,
				Value: buyerProof.Value,
			},
			DocumentInvoiceAmountProof: Proof{
				Hashes: stringutils.RemoveStringInList(proof.FieldProofs[0].Hashes, "0x"),
				Right: proof.FieldProofs[0].Right,
				Value: strings.Replace(proof.FieldProofs[0].Value, "0x", "", -1),
				Salt: strings.Replace(proof.FieldProofs[0].Salt, "0x", "", -1),
				Property: strings.Replace(proof.FieldProofs[0].Property, "0x", "", -1),
			},
			DocumentInvoiceBuyerProof: Proof{
				Hashes: stringutils.RemoveStringInList(proof.FieldProofs[1].Hashes, "0x"),
				Right: proof.FieldProofs[1].Right,
				Value: strings.Replace(proof.FieldProofs[1].Value, "0x", "", -1),
				Salt: strings.Replace(proof.FieldProofs[1].Salt, "0x", "", -1),
				Property: strings.Replace(proof.FieldProofs[1].Property, "0x", "", -1),
			},
		},
	}
	log.Debug(json.MarshalIndent(zkPayload, "", "  "))

	requestData, err := NewMintRequest(tokenID, req.DepositAddress, anchorID, nextAnchorID, docProofs.FieldProofs)
	if err != nil {
		return mreq, zkPayload, err
	}

	return requestData, zkPayload, nil

}

// GetRequiredInvoiceUnpaidProofFields returns required proof fields for an unpaid invoice mint
func (s *ethInvoiceUnpaid) GetRequiredInvoiceUnpaidProofFields(ctx context.Context) ([]string, error) {
	var proofFields []string

	//acc, err := contextutil.Account(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//accDIDBytes, err := acc.GetIdentityID()
	//if err != nil {
	//	return nil, err
	//}
	//keys, err := acc.GetKeys()
	//if err != nil {
	//	return nil, err
	//}

	//docDataRoot := fmt.Sprintf("%s.%s", documents.DRTreePrefix, documents.DocumentDataRootField)
	//signerID := hexutil.Encode(append(accDIDBytes, keys[identity.KeyPurposeSigning.Name].PublicKey...))
	//signatureSender := fmt.Sprintf("%s.signatures[%s]", documents.SignaturesTreePrefix, signerID)
	//proofFields = []string{"invoice.gross_amount", "invoice.currency", "invoice.date_due", "invoice.sender", "invoice.status", docDataRoot, signatureSender, documents.CDTreePrefix + ".next_version"}
	proofFields = []string{"invoice.gross_amount", "invoice.recipient"}
	return proofFields, nil
}

// MintNFT mints an NFT
func (s *ethInvoiceUnpaid) MintNFT(ctx context.Context, req MintNFTRequest) (*MintNFTResponse, chan bool, error) {
	tc, err := contextutil.Account(ctx)
	if err != nil {
		return nil, nil, err
	}

	if !req.GrantNFTReadAccess && req.SubmitNFTReadAccessProof {
		return nil, nil, errors.New("enable grant_nft_access to generate Read Access Proof")
	}

	tokenID := NewTokenID()
	if s.cfg.GetLowEntropyNFTTokenEnabled() {
		log.Warningf("Security consideration: Using a reduced maximum of %s integer for NFT token ID generation. "+
			"Suggested course of action: disable by setting nft.lowentropy=false in config.yaml file", LowEntropyTokenIDMax)
		tokenID = NewLowEntropyTokenID()
	}

	model, err := s.docSrv.GetCurrentVersion(ctx, req.DocumentID)
	if err != nil {
		return nil, nil, err
	}

	// check if the nft is successfully minted already
	if model.IsNFTMinted(s, req.RegistryAddress) {
		return nil, nil, errors.NewTypedError(ErrNFTMinted, errors.New("registry %v", req.RegistryAddress.String()))
	}

	didBytes, err := tc.GetIdentityID()
	if err != nil {
		return nil, nil, err
	}

	// Mint NFT within transaction
	// We use context.Background() for now so that the transaction is only limited by ethereum timeouts
	did, err := identity.NewDIDFromBytes(didBytes)
	if err != nil {
		return nil, nil, err
	}
	jobID, done, err := s.jobsManager.ExecuteWithinJob(contextutil.Copy(ctx), did, jobs.NilJobID(), "Minting NFT",
		s.minter(ctx, tokenID, model, req))
	if err != nil {
		return nil, nil, err
	}

	return &MintNFTResponse{
		JobID:   jobID.String(),
		TokenID: tokenID.String(),
	}, done, nil
}

func (s *ethInvoiceUnpaid) minter(ctx context.Context, tokenID TokenID, model documents.Model, req MintNFTRequest) func(accountID identity.DID, txID jobs.JobID, txMan jobs.Manager, errOut chan<- error) {
	return func(accountID identity.DID, jobID jobs.JobID, txMan jobs.Manager, errOut chan<- error) {
		err := model.AddNFT(req.GrantNFTReadAccess, req.RegistryAddress, tokenID[:])
		if err != nil {
			errOut <- err
			return
		}

		jobCtx := contextutil.WithJob(ctx, jobID)
		_, _, done, err := s.docSrv.Update(jobCtx, model)
		if err != nil {
			errOut <- err
			return
		}

		isDone := <-done
		if !isDone {
			// some problem occurred in a child task
			errOut <- errors.New("update document failed for document %s and job %s", hexutil.Encode(req.DocumentID), jobID)
			return
		}

		requestData, zkPayload, err := s.prepareMintRequest(jobCtx, tokenID, accountID, req)
		if err != nil {
			errOut <- errors.New("failed to prepare mint request: %v", err)
			return
		}

		zkPayloadB, err := json.MarshalIndent(zkPayload, "", "  ")
		if err != nil {
			errOut <- err
			return
		}
		dir, err := os.Getwd()
		if err != nil {
			errOut <- err
			return
		}
		fmt.Println("Running NFT Crypto")
		err = snarks.CallNFTCrypto(zkPayloadB, dir+"/out")
		if err != nil {
			errOut <- errors.New("failed to call nft.py script: %v", err)
			return
		}
		fmt.Println("Running ZoKrates")
		points, err := snarks.GetZokratesProofs()
		if err != nil {
			errOut <- errors.New("failed to call ZoKrates script: %v", err)
			return
		}

		dataRootB, err := hex.DecodeString(zkPayload.Public.DocumentRootHash)
		sigRootB, err := hex.DecodeString(zkPayload.Public.SignaturesRootHash)
		nftAmountB, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000140")
		buyerRatingB, err := hex.DecodeString(zkPayload.Public.Rating)

		dataRoot, err := utils.SliceToByte32(dataRootB)
		sigRoot, err := utils.SliceToByte32(sigRootB)
		nftAmount := utils.ByteSliceToBigInt(nftAmountB)
		buyerRating := utils.ByteSliceToBigInt(buyerRatingB)

		did, err := contextutil.AccountDID(ctx)
		if err != nil {
			errOut <- errors.New("failed to get DID: %v", err)
			return
		}

		tc, err := contextutil.Account(ctx)
		if err != nil {
			errOut <- errors.New("failed to get Account: %v", err)
			return
		}

		conn := s.ethClient
		opts, err := conn.GetTxOpts(ctx, tc.GetEthereumDefaultAccountName())
		if err != nil {
			errOut <- errors.New("failed to get opts: %v", err)
			return
		}

		opts.GasLimit = s.cfg.GetEthereumGasLimit(config.NftMint)
		zkContract, err := NewZkNFTContract(s.cfg.GetContractAddress(config.InvoiceUnpaidNFT), s.ethClient.GetEthClient())
		_, done, err = s.jobsManager.ExecuteWithinJob(ctx, did, jobID, "Check Job for zMint",
			s.ethereumTX(opts, zkContract.Mint, requestData.To, requestData.TokenID, requestData.AnchorID, dataRoot, sigRoot, nftAmount, buyerRating, points))

		// Call mint method on zkNFT contract
		//txID, done, err := s.identityService.Execute(ctx, req.RegistryAddress, ABIZKNFT, "mint", requestData.To, requestData.TokenID, requestData.AnchorID, dataRoot, sigRoot, nftAmount, buyerRating, points)

		// $NFT_REGISTRY ‘mint(address,uint,uint,bytes32,bytes32,uint,uint,uint[8] memory)(uint)’ $ETH_FROM $TKN_ID $ANCHOR_ID $DATA_ROOT $SIG_ROOT $AMOUNT $RATING $POINTS
		// to common.Address, tokenId *big.Int, tokenURI string, anchorId *big.Int, properties [][]byte, values [][]byte, salts [][32]byte, proofs [][][32]byte
		//txID, done, err := s.identityService.Execute(ctx, req.RegistryAddress, InvoiceUnpaidContractABI, "mint", requestData.To, requestData.TokenID, requestData.AnchorID, requestData.Props, requestData.Values, requestData.Salts, requestData.Proofs)
		if err != nil {
			errOut <- err
			return
		}
		log.Infof("Sent off ethTX to mint [tokenID: %s, anchor: %x, nextAnchor: %s, registry: %s] to invoice unpaid contract.",
			requestData.TokenID, requestData.AnchorID, hexutil.Encode(requestData.NextAnchorID.Bytes()), requestData.To.String())

		log.Debugf("To: %s", requestData.To.String())
		log.Debugf("TokenID: %s", hexutil.Encode(requestData.TokenID.Bytes()))
		log.Debugf("AnchorID: %s", hexutil.Encode(requestData.AnchorID.Bytes()))
		log.Debugf("NextAnchorID: %s", hexutil.Encode(requestData.NextAnchorID.Bytes()))
		log.Debugf("Props: %s", byteSlicetoString(requestData.Props))
		log.Debugf("Values: %s", byteSlicetoString(requestData.Values))
		log.Debugf("Salts: %s", byte32SlicetoString(requestData.Salts))
		log.Debugf("Proofs: %s", byteByte32SlicetoString(requestData.Proofs))

		isDone = <-done
		if !isDone {
			// some problem occurred in a child task
			errOut <- errors.New("mint nft failed for document %s and transaction", hexutil.Encode(req.DocumentID))
			return
		}

		// Check if tokenID exists in registry and owner is deposit address
		owner, err := s.OwnerOf(req.RegistryAddress, tokenID[:])
		if err != nil {
			errOut <- errors.New("error while checking new NFT owner %v", err)
			return
		}
		if owner.Hex() != req.DepositAddress.Hex() {
			errOut <- errors.New("Owner for tokenID %s should be %s, instead got %s", tokenID.String(), req.DepositAddress.Hex(), owner.Hex())
			return
		}

		log.Infof("Document %s minted successfully within transaction", hexutil.Encode(req.DocumentID))

		errOut <- nil
		return
	}
}

// OwnerOf returns the owner of the NFT token on ethereum chain
func (s *ethInvoiceUnpaid) OwnerOf(registry common.Address, tokenID []byte) (owner common.Address, err error) {
	contract, err := s.bindContract(registry, s.ethClient)
	if err != nil {
		return owner, errors.New("failed to bind the registry contract: %v", err)
	}

	opts, cancF := s.ethClient.GetGethCallOpts(false)
	defer cancF()

	return contract.OwnerOf(opts, utils.ByteSliceToBigInt(tokenID))
}

// CurrentIndexOfToken returns the current index of the token in the given registry
func (s *ethInvoiceUnpaid) CurrentIndexOfToken(registry common.Address, tokenID []byte) (*big.Int, error) {
	contract, err := s.bindContract(registry, s.ethClient)
	if err != nil {
		return nil, errors.New("failed to bind the registry contract: %v", err)
	}

	opts, cancF := s.ethClient.GetGethCallOpts(false)
	defer cancF()

	return contract.CurrentIndexOfToken(opts, utils.ByteSliceToBigInt(tokenID))
}

// MintRequest holds the data needed to mint and NFT from a Centrifuge document
type MintRequest struct {

	// To is the address of the recipient of the minted token
	To common.Address

	// TokenID is the ID for the minted token
	TokenID *big.Int

	// AnchorID is the ID of the document as identified by the set up anchorRepository.
	AnchorID *big.Int

	// NextAnchorID is the next ID of the document, when updated
	NextAnchorID *big.Int

	// Props contains the compact props for readRole and tokenRole
	Props [][]byte

	// Values are the values of the leafs that is being proved Will be converted to string and concatenated for proof verification as outlined in precise-proofs library.
	Values [][]byte

	// salts are the salts for the field that is being proved Will be concatenated for proof verification as outlined in precise-proofs library.
	Salts [][32]byte

	// Proofs are the documents proofs that are needed
	Proofs [][][32]byte
}

// NewMintRequest converts the parameters and returns a struct with needed parameter for minting
func NewMintRequest(tokenID TokenID, to common.Address, anchorID anchors.AnchorID, nextAnchorID anchors.AnchorID, proofs []*proofspb.Proof) (MintRequest, error) {
	proofData, err := convertToProofData(proofs)
	if err != nil {
		return MintRequest{}, err
	}

	return MintRequest{
		To:           to,
		TokenID:      tokenID.BigInt(),
		AnchorID:     anchorID.BigInt(),
		NextAnchorID: nextAnchorID.BigInt(),
		Props:        proofData.Props,
		Values:       proofData.Values,
		Salts:        proofData.Salts,
		Proofs:       proofData.Proofs}, nil
}

type proofData struct {
	Props  [][]byte
	Values [][]byte
	Salts  [][32]byte
	Proofs [][][32]byte
}

func convertToProofData(proofspb []*proofspb.Proof) (*proofData, error) {
	var props = make([][]byte, len(proofspb))
	var values = make([][]byte, len(proofspb))
	var salts = make([][32]byte, len(proofspb))
	var proofs = make([][][32]byte, len(proofspb))

	for i, p := range proofspb {
		salt32, err := utils.SliceToByte32(p.Salt)
		if err != nil {
			return nil, err
		}
		property, err := utils.ConvertProofForEthereum(p.SortedHashes)
		if err != nil {
			return nil, err
		}
		props[i] = p.GetCompactName()
		values[i] = p.Value
		// Scenario where it is a hashed field we copy the Hash value into the property value
		if len(p.Value) == 0 && len(p.Salt) == 0 {
			values[i] = p.Hash
		}
		salts[i] = salt32
		proofs[i] = property
	}

	return &proofData{Props: props, Values: values, Salts: salts, Proofs: proofs}, nil
}

func bindContract(address common.Address, client ethereum.Client) (*InvoiceUnpaidContract, error) {
	return NewInvoiceUnpaidContract(address, client.GetEthClient())
}

// Following are utility methods for nft parameter debugging purposes (Don't remove)

func byteSlicetoString(s [][]byte) string {
	str := "["

	for i := 0; i < len(s); i++ {
		str += "\"" + hexutil.Encode(s[i]) + "\",\n"
	}
	str += "]"
	return str
}

func byte32SlicetoString(s [][32]byte) string {
	str := "["

	for i := 0; i < len(s); i++ {
		str += "\"" + hexutil.Encode(s[i][:]) + "\",\n"
	}
	str += "]"
	return str
}

func byteByte32SlicetoString(s [][][32]byte) string {
	str := "["

	for i := 0; i < len(s); i++ {
		str += "\"" + byte32SlicetoString(s[i]) + "\",\n"
	}
	str += "]"
	return str
}
