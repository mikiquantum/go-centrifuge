package coredocument

import (
	"fmt"

	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/coredocument"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/errors"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/tools"
	"github.com/centrifuge/precise-proofs/proofs"
)

// Validate checks that all required fields are set before doing any processing with core document
func Validate(document *coredocumentpb.CoreDocument) (valid bool, errMsg string, errs map[string]string) {
	if document == nil {
		return false, errors.NilDocument, nil
	}

	errs = make(map[string]string)

	if tools.IsEmptyByteSlice(document.DocumentIdentifier) {
		errs["cd_identifier"] = errors.RequiredField
	}

	// TODO(ved): where do we fill these
	//if tools.IsEmptyByteSlice(document.DocumentRoot) {
	//	errs["cd_root"] = errors.RequiredField
	//}

	if tools.IsEmptyByteSlice(document.CurrentIdentifier) {
		errs["cd_current_identifier"] = errors.RequiredField
	}

	if tools.IsEmptyByteSlice(document.NextIdentifier) {
		errs["cd_next_identifier"] = errors.RequiredField
	}

	if tools.IsEmptyByteSlice(document.DataRoot) {
		errs["cd_data_root"] = errors.RequiredField
	}

	// double check the identifiers
	isSameBytes := tools.IsSameByteSlice

	// Problem (re-using an old identifier for NextIdentifier): CurrentIdentifier or DocumentIdentifier same as NextIdentifier
	if isSameBytes(document.NextIdentifier, document.DocumentIdentifier) ||
		isSameBytes(document.NextIdentifier, document.CurrentIdentifier) {
		errs["cd_overall"] = errors.IdentifierReUsed
	}

	// lets not do verbose check like earlier since these will be
	// generated by us mostly
	salts := document.CoredocumentSalts
	if salts == nil ||
		!tools.CheckMultiple32BytesFilled(
			salts.CurrentIdentifier,
			salts.DataRoot,
			salts.NextIdentifier,
			salts.DocumentIdentifier,
			salts.PreviousRoot) {
		errs["cd_salts"] = errors.RequiredField
	}

	if len(errs) < 1 {
		return true, "", nil
	}

	return false, "Invalid CoreDocument", errs
}

// FillIdentifiers fills in missing identifiers for the given CoreDocument.
// It does checks on document consistency (e.g. re-using an old identifier).
// In the case of an error, it returns the error and an empty CoreDocument.
func FillIdentifiers(document coredocumentpb.CoreDocument) (coredocumentpb.CoreDocument, error) {
	isEmptyId := tools.IsEmptyByteSlice

	// check if the document identifier is empty
	if !isEmptyId(document.DocumentIdentifier) {
		// check and fill current and next identifiers
		if isEmptyId(document.CurrentIdentifier) {
			document.CurrentIdentifier = document.DocumentIdentifier
		}

		if isEmptyId(document.NextIdentifier) {
			document.NextIdentifier = tools.RandomSlice(32)
		}

		return document, nil
	}

	// check if current and next identifier are empty
	if !isEmptyId(document.CurrentIdentifier) {
		return document, fmt.Errorf("no DocumentIdentifier but has CurrentIdentifier")
	}

	// check if the next identifier is empty
	if !isEmptyId(document.NextIdentifier) {
		return document, fmt.Errorf("no CurrentIdentifier but has NextIdentifier")
	}

	// fill the identifiers
	document.DocumentIdentifier = tools.RandomSlice(32)
	document.CurrentIdentifier = document.DocumentIdentifier
	document.NextIdentifier = tools.RandomSlice(32)
	return document, nil
}

// New returns a new core document from the proto message
func New() *coredocumentpb.CoreDocument {
	doc, _ := FillIdentifiers(coredocumentpb.CoreDocument{})
	salts := &coredocumentpb.CoreDocumentSalts{}
	proofs.FillSalts(salts)
	doc.CoredocumentSalts = salts
	return &doc
}
