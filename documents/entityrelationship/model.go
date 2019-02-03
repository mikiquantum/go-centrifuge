package entityrelationship

import (
	"encoding/json"
	"github.com/centrifuge/centrifuge-protobufs/gen/go/coredocument"
	"github.com/centrifuge/go-centrifuge/identity"
	"github.com/golang/protobuf/ptypes/timestamp"
	"math/big"
	"reflect"
)

const prefix string = "entityrelationship"

type EntityRelationship struct {
	EntityID []byte
	CustomFields map[string]interface{}
	Identity identity.CentID
	ExpirationBlockHeight big.Int
	DateCreated      *timestamp.Timestamp

	CoreDocument *coredocumentpb.CoreDocument
}

// ID returns document identifier.
// Note: this is not a unique identifier for each version of the document.
func (er *EntityRelationship) ID() ([]byte, error) {
	return nil, nil
}

// JSON marshals EntityRelationship into a json bytes
func (er *EntityRelationship) JSON() ([]byte, error) {
	return json.Marshal(er)
}

// FromJSON unmarshals the json bytes into EntityRelationship
func (er *EntityRelationship) FromJSON(jsonData []byte) error {
	return json.Unmarshal(jsonData, er)
}

// Type gives the EntityRelationship type
func (er *EntityRelationship) Type() reflect.Type {
	return reflect.TypeOf(er)
}