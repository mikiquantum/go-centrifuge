// +build unit

package invoiceservice_test

import (
	"context"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/coredocument"
	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/invoice"
	cc "github.com/CentrifugeInc/go-centrifuge/centrifuge/context/testing"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/invoice"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/invoice/service"
	clientinvoicepb "github.com/CentrifugeInc/go-centrifuge/centrifuge/protobufs/gen/go/invoice"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/testingutils"
	"github.com/centrifuge/precise-proofs/proofs"
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	cc.TestIntegrationBootstrap()
	result := m.Run()
	cc.TestIntegrationTearDown()
	os.Exit(result)
}

// mockInvoiceRepository implements storage.Repository
type mockInvoiceRepository struct {
	mock.Mock
	replaceDoc *invoicepb.InvoiceDocument
}

func (m *mockInvoiceRepository) Exists(id []byte) bool {
	args := m.Called(id)
	return args.Get(0).(bool)
}

func (m *mockInvoiceRepository) GetKey(id []byte) []byte {
	args := m.Called(id)
	return args.Get(0).([]byte)
}

func (m *mockInvoiceRepository) GetByID(id []byte, doc proto.Message) (err error) {
	args := m.Called(id, doc)
	order := doc.(*invoicepb.InvoiceDocument)
	*order = *m.replaceDoc
	doc = order
	return args.Error(0)
}

func (m *mockInvoiceRepository) Create(id []byte, doc proto.Message) (err error) {
	args := m.Called(id, doc)
	return args.Error(0)
}

func (m *mockInvoiceRepository) Update(id []byte, doc proto.Message) (err error) {
	args := m.Called(id, doc)
	return args.Error(0)
}

func generateMockedOutInvoiceService() (srv *invoiceservice.InvoiceDocumentService, repo *mockInvoiceRepository, coreDocumentProcessor *testingutils.MockCoreDocumentProcessor) {
	repo = new(mockInvoiceRepository)
	coreDocumentProcessor = new(testingutils.MockCoreDocumentProcessor)
	srv = &invoiceservice.InvoiceDocumentService{
		InvoiceRepository:     repo,
		CoreDocumentProcessor: coreDocumentProcessor,
	}
	return srv, repo, coreDocumentProcessor
}
func getTestSetupData() (doc *invoice.Invoice, srv *invoiceservice.InvoiceDocumentService, repo *mockInvoiceRepository, coreDocumentProcessor *testingutils.MockCoreDocumentProcessor) {
	doc = &invoice.Invoice{Document: &invoicepb.InvoiceDocument{}}
	doc.Document.Data = &invoicepb.InvoiceData{
		InvoiceNumber:    "inv1234",
		SenderName:       "Jack",
		SenderZipcode:    "921007",
		SenderCountry:    "AUS",
		RecipientName:    "John",
		RecipientZipcode: "12345",
		RecipientCountry: "Germany",
		Currency:         "EUR",
		GrossAmount:      800,
	}
	salts := new(invoicepb.InvoiceDataSalts)
	proofs.FillSalts(salts)
	doc.Document.CoreDocument = testingutils.GenerateCoreDocument()
	doc.Document.Salts = salts
	srv, repo, coreDocumentProcessor = generateMockedOutInvoiceService()
	return doc, srv, repo, coreDocumentProcessor
}

func TestInvoiceDocumentService_Anchor(t *testing.T) {
	doc, s, mockRepo, mockCDP := getTestSetupData()

	mockRepo.On("Create", doc.Document.CoreDocument.DocumentIdentifier, doc.Document).Return(nil).Once()
	mockRepo.On("Update", doc.Document.CoreDocument.DocumentIdentifier, mock.Anything).Return(nil).Once()
	mockCDP.On("Anchor", mock.Anything).Return(nil).Once()

	anchoredDoc, err := s.HandleAnchorInvoiceDocument(context.Background(), &clientinvoicepb.AnchorInvoiceEnvelope{Document: doc.Document})

	mockRepo.AssertExpectations(t)
	mockCDP.AssertExpectations(t)
	assert.Nil(t, err)
	assert.Equal(t, doc.Document.CoreDocument.DocumentIdentifier, anchoredDoc.CoreDocument.DocumentIdentifier)
}

func TestInvoiceDocumentService_AnchorFails(t *testing.T) {
	doc, s, mockRepo, mockCDP := getTestSetupData()

	mockRepo.On("Create", doc.Document.CoreDocument.DocumentIdentifier, doc.Document).Return(nil).Once()
	mockCDP.On("Anchor", mock.Anything).Return(errors.New("error anchoring")).Once()

	anchoredDoc, err := s.HandleAnchorInvoiceDocument(context.Background(), &clientinvoicepb.AnchorInvoiceEnvelope{Document: doc.Document})

	mockRepo.AssertExpectations(t)
	mockCDP.AssertExpectations(t)
	assert.Error(t, err)
	assert.Nil(t, anchoredDoc)
}

func TestInvoiceDocumentService_AnchorFailsWithNilDocument(t *testing.T) {
	_, s, _, _ := getTestSetupData()

	anchoredDoc, err := s.HandleAnchorInvoiceDocument(context.Background(), &clientinvoicepb.AnchorInvoiceEnvelope{})

	assert.Error(t, err)
	assert.Nil(t, anchoredDoc)
}

func TestInvoiceDocumentService_Send(t *testing.T) {
	doc, s, mockRepo, mockCDP := getTestSetupData()

	recipients := testingutils.GenerateP2PRecipients(1)

	mockRepo.On("Create", doc.Document.CoreDocument.DocumentIdentifier, doc.Document).Return(nil).Once()
	mockRepo.On("Update", doc.Document.CoreDocument.DocumentIdentifier, mock.Anything).Return(nil).Once()
	mockCDP.On("Anchor", mock.Anything).Return(nil).Once()
	mockCDP.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, err := s.HandleSendInvoiceDocument(context.Background(), &clientinvoicepb.SendInvoiceEnvelope{Recipients: recipients, Document: doc.Document})

	mockRepo.AssertExpectations(t)
	mockCDP.AssertExpectations(t)
	assert.Nil(t, err)
}

func TestInvoiceDocumentService_SendFails(t *testing.T) {
	doc, s, mockRepo, mockCDP := getTestSetupData()
	recipients := testingutils.GenerateP2PRecipients(2)

	mockRepo.On("Create", doc.Document.CoreDocument.DocumentIdentifier, doc.Document).Return(nil).Once()
	mockRepo.On("Update", doc.Document.CoreDocument.DocumentIdentifier, mock.Anything).Return(nil).Once()
	mockCDP.On("Anchor", mock.Anything).Return(nil).Once()
	mockCDP.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("error sending")).Twice()

	_, err := s.HandleSendInvoiceDocument(context.Background(), &clientinvoicepb.SendInvoiceEnvelope{Recipients: recipients, Document: doc.Document})

	mockCDP.AssertExpectations(t)
	//the error handling in the send handler simply prints out the list of errors without much formatting
	//OK for now but could be done nicer in the future
	assert.Contains(t, err.Error(), "error sending error sending")
}

func TestInvoiceDocumentService_Send_StoreFails(t *testing.T) {
	doc, s, mockRepo, _ := getTestSetupData()
	recipients := testingutils.GenerateP2PRecipients(2)

	mockRepo.On("Create", doc.Document.CoreDocument.DocumentIdentifier, doc.Document).Return(errors.New("error storing")).Once()

	_, err := s.HandleSendInvoiceDocument(context.Background(), &clientinvoicepb.SendInvoiceEnvelope{Recipients: recipients, Document: doc.Document})

	mockRepo.AssertExpectations(t)
	assert.Contains(t, err.Error(), "error storing")
}

func TestInvoiceDocumentService_Send_AnchorFails(t *testing.T) {
	doc, s, mockRepo, mockCDP := getTestSetupData()
	recipients := testingutils.GenerateP2PRecipients(2)

	mockRepo.On("Create", doc.Document.CoreDocument.DocumentIdentifier, doc.Document).Return(nil).Once()
	mockCDP.On("Anchor", mock.Anything).Return(errors.New("error anchoring")).Once()

	_, err := s.HandleSendInvoiceDocument(context.Background(), &clientinvoicepb.SendInvoiceEnvelope{Recipients: recipients, Document: doc.Document})

	mockRepo.AssertExpectations(t)
	mockCDP.AssertExpectations(t)
	assert.Contains(t, err.Error(), "error anchoring")
}

func TestInvoiceDocumentService_HandleCreateInvoiceProof(t *testing.T) {
	identifier := testingutils.Rand32Bytes()
	inv := invoice.Empty()
	inv.Document.CoreDocument = &coredocumentpb.CoreDocument{
		DocumentIdentifier: identifier,
		CurrentIdentifier:  identifier,
		NextIdentifier:     testingutils.Rand32Bytes(),
	}
	inv.CalculateMerkleRoot()

	s, mockRepo, _ := generateMockedOutInvoiceService()

	proofRequest := &clientinvoicepb.CreateInvoiceProofEnvelope{
		DocumentIdentifier: identifier,
		Fields:             []string{"currency", "sender_country", "gross_amount"},
	}
	mockRepo.On("GetByID", proofRequest.DocumentIdentifier, new(invoicepb.InvoiceDocument)).Return(nil).Once()
	mockRepo.replaceDoc = inv.Document
	invoiceProof, err := s.HandleCreateInvoiceProof(context.Background(), proofRequest)
	mockRepo.AssertExpectations(t)

	assert.Nil(t, err)
	assert.Equal(t, identifier, invoiceProof.DocumentIdentifier)
	assert.Equal(t, len(proofRequest.Fields), len(invoiceProof.FieldProofs))
	assert.Equal(t, proofRequest.Fields[0], invoiceProof.FieldProofs[0].Property)
	sha256Hash := sha256.New()
	valid, err := proofs.ValidateProof(invoiceProof.FieldProofs[0], inv.Document.CoreDocument.DataRoot, sha256Hash)
	assert.True(t, valid)
	assert.Nil(t, err)
}

func TestInvoiceDocumentService_HandleCreateInvoiceProof_NotFilledSalts(t *testing.T) {
	identifier := testingutils.Rand32Bytes()
	inv := invoice.Empty()
	inv.Document.CoreDocument = &coredocumentpb.CoreDocument{
		DocumentIdentifier: identifier,
		CurrentIdentifier:  identifier,
		NextIdentifier:     testingutils.Rand32Bytes(),
	}
	inv.Document.Salts = &invoicepb.InvoiceDataSalts{}
	s, mockRepo, _ := generateMockedOutInvoiceService()

	proofRequest := &clientinvoicepb.CreateInvoiceProofEnvelope{
		DocumentIdentifier: identifier,
		Fields:             []string{"currency", "sender_country", "gross_amount"},
	}
	mockRepo.On("GetByID", proofRequest.DocumentIdentifier, new(invoicepb.InvoiceDocument)).Return(nil).Once()
	mockRepo.replaceDoc = inv.Document
	invoiceProof, err := s.HandleCreateInvoiceProof(context.Background(), proofRequest)
	mockRepo.AssertExpectations(t)
	assert.NotNil(t, err)
	assert.Nil(t, invoiceProof)
}

func TestInvoiceDocumentService_HandleCreateInvoiceProof_NotExistingInvoice(t *testing.T) {
	identifier := testingutils.Rand32Bytes()
	inv := invoice.Empty()
	inv.Document.CoreDocument = &coredocumentpb.CoreDocument{
		DocumentIdentifier: identifier,
		CurrentIdentifier:  identifier,
		NextIdentifier:     testingutils.Rand32Bytes(),
	}
	inv.CalculateMerkleRoot()

	s, mockRepo, _ := generateMockedOutInvoiceService()

	proofRequest := &clientinvoicepb.CreateInvoiceProofEnvelope{
		DocumentIdentifier: identifier,
		Fields:             []string{"currency", "sender_country", "gross_amount"},
	}
	//return an "empty" invoice as mock can't return nil
	mockRepo.On("GetByID", proofRequest.DocumentIdentifier, new(invoicepb.InvoiceDocument)).Return(errors.New("couldn't find invoice")).Once()
	mockRepo.replaceDoc = inv.Document
	invoiceProof, err := s.HandleCreateInvoiceProof(context.Background(), proofRequest)
	mockRepo.AssertExpectations(t)

	assert.Nil(t, invoiceProof)
	assert.Error(t, err)
}
