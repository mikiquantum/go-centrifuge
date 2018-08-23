package purchaseorderservice

import (
	"fmt"

	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/purchaseorder"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/coredocument"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/coredocument/repository"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/coredocument/service"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/errors"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/purchaseorder"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/purchaseorder/repository"
	gerrors "github.com/go-errors/errors"
	google_protobuf2 "github.com/golang/protobuf/ptypes/empty"
	logging "github.com/ipfs/go-log"
	"golang.org/x/net/context"
)

var log = logging.Logger("rest-api")

// Struct needed as it is used to register the grpc services attached to the grpc server
type PurchaseOrderDocumentService struct {
	PurchaseOrderRepository purchaseorderrepository.PurchaseOrderRepository
	CoreDocumentProcessor   coredocument.CoreDocumentProcessorInterface
}

func fillCoreDocIdentifiers(doc *purchaseorderpb.PurchaseOrderDocument) error {
	if doc == nil {
		return errors.NilError(doc)
	}
	filledCoreDoc, err := coredocumentservice.AutoFillDocumentIdentifiers(*doc.CoreDocument)
	if err != nil {
		log.Error(err)
		return err
	}
	doc.CoreDocument = &filledCoreDoc
	return nil
}

// HandleCreatePurchaseOrderProof creates proofs for a list of fields
func (s *PurchaseOrderDocumentService) HandleCreatePurchaseOrderProof(ctx context.Context, createPurchaseOrderProofEnvelope *purchaseorderpb.CreatePurchaseOrderProofEnvelope) (*purchaseorderpb.PurchaseOrderProof, error) {
	orderDoc, err := s.PurchaseOrderRepository.FindById(createPurchaseOrderProofEnvelope.DocumentIdentifier)
	if err != nil {
		return nil, err
	}

	order, err := purchaseorder.NewPurchaseOrder(orderDoc)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	proofs, err := order.CreateProofs(createPurchaseOrderProofEnvelope.Fields)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &purchaseorderpb.PurchaseOrderProof{FieldProofs: proofs, DocumentIdentifier: order.Document.CoreDocument.DocumentIdentifier}, nil

}

// HandleAnchorPurchaseOrderDocument anchors the given purchaseorder document and returns the anchor details
func (s *PurchaseOrderDocumentService) HandleAnchorPurchaseOrderDocument(ctx context.Context, anchorPurchaseOrderEnvelope *purchaseorderpb.AnchorPurchaseOrderEnvelope) (*purchaseorderpb.PurchaseOrderDocument, error) {
	doc := anchorPurchaseOrderEnvelope.Document

	err := fillCoreDocIdentifiers(doc)
	if err != nil {
		log.Error(err)
		return nil, gerrors.Errorf("Error filling document IDs: [%v]", err.Error())
	}

	err = s.PurchaseOrderRepository.Create(doc)
	if err != nil {
		log.Error(err)
		return nil, gerrors.Errorf("Error saving document: [%v]", err.Error())
	}

	anchoredPurchaseOrder, err := s.anchorPurchaseOrderDocument(doc)
	if err != nil {
		log.Error(err)
		return nil, gerrors.Errorf("Error anchoring document: [%v]", err.Error())
	}

	return anchoredPurchaseOrder, nil
}

// HandleSendPurchaseOrderDocument anchors and sends an purchaseorder to the recipient
func (s *PurchaseOrderDocumentService) HandleSendPurchaseOrderDocument(ctx context.Context, sendPurchaseOrderEnvelope *purchaseorderpb.SendPurchaseOrderEnvelope) (*purchaseorderpb.PurchaseOrderDocument, error) {
	doc := sendPurchaseOrderEnvelope.Document

	err := fillCoreDocIdentifiers(doc)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	err = s.PurchaseOrderRepository.Create(doc)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	anchoredPurchaseOrder, err := s.anchorPurchaseOrderDocument(sendPurchaseOrderEnvelope.Document)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	errs := []error{}
	for _, recipient := range sendPurchaseOrderEnvelope.Recipients {
		err1 := s.CoreDocumentProcessor.Send(anchoredPurchaseOrder.CoreDocument, ctx, recipient)
		if err1 != nil {
			errs = append(errs, err1)
		}
	}

	if len(errs) != 0 {
		log.Errorf("%v", errs)
		return nil, fmt.Errorf("%v", errs)
	}
	return anchoredPurchaseOrder, nil
}

func (s *PurchaseOrderDocumentService) HandleGetPurchaseOrderDocument(ctx context.Context, getPurchaseOrderDocumentEnvelope *purchaseorderpb.GetPurchaseOrderDocumentEnvelope) (*purchaseorderpb.PurchaseOrderDocument, error) {
	doc, err := s.PurchaseOrderRepository.FindById(getPurchaseOrderDocumentEnvelope.DocumentIdentifier)
	if err != nil {
		docFound, err1 := coredocumentrepository.GetCoreDocumentRepository().FindById(getPurchaseOrderDocumentEnvelope.DocumentIdentifier)
		if err1 == nil {
			doc1, err1 := purchaseorder.NewPurchaseOrderFromCoreDocument(docFound)
			doc = doc1.Document
			err = err1
		}
		log.Errorf("%v", err)
	}
	return doc, err
}

func (s *PurchaseOrderDocumentService) HandleGetReceivedPurchaseOrderDocuments(ctx context.Context, empty *google_protobuf2.Empty) (*purchaseorderpb.ReceivedPurchaseOrders, error) {
	return nil, nil
}

// anchorPurchaseOrderDocument anchors the given purchaseorder document and returns the anchor details
func (s *PurchaseOrderDocumentService) anchorPurchaseOrderDocument(doc *purchaseorderpb.PurchaseOrderDocument) (*purchaseorderpb.PurchaseOrderDocument, error) {
	// TODO: the calculated merkle root should be persisted locally as well.
	orderDoc, err := purchaseorder.NewPurchaseOrder(doc)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	orderDoc.CalculateMerkleRoot()
	coreDoc := orderDoc.ConvertToCoreDocument()

	err = s.CoreDocumentProcessor.Anchor(coreDoc)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	newPo, err := purchaseorder.NewPurchaseOrderFromCoreDocument(coreDoc)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return newPo.Document, nil
}