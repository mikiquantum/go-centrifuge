package entity

import (
	"encoding/json"
	"github.com/centrifuge/centrifuge-protobufs/gen/go/coredocument"
	"github.com/centrifuge/go-centrifuge/identity"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/ptypes/timestamp"
	"reflect"
)

const prefix string = "entity"

type Address struct {
	Main bool
	CustomFields map[string]interface{}
	Address1 string
	Address2 string
	Country  string
	ZipCode      string
}

type Contact struct {
	Name string
	Email string
	Phone string
	Title string
}

type PaymentMethod struct {
	Predetermined bool
	CustomFields map[string]interface{}
	Address Address
	HolderName string
	BankKey string
	BankAccountNumber string
	Currency string
	EthereumAddress common.Address
}

type Entity struct {
	Identity identity.CentID
	CustomFields map[string]interface{}
	Addresses []Address
	PaymentMethods []PaymentMethod
	Contacts []Contact
	DateCreated      *timestamp.Timestamp

	CoreDocument *coredocumentpb.CoreDocument
}

// ID returns document identifier.
// Note: this is not a unique identifier for each version of the document.
func (e *Entity) ID() ([]byte, error) {
	return nil, nil
}

// JSON marshals Entity into a json bytes
func (e *Entity) JSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON unmarshals the json bytes into Entity
func (e *Entity) FromJSON(jsonData []byte) error {
	return json.Unmarshal(jsonData, e)
}

// Type gives the Entity type
func (e *Entity) Type() reflect.Type {
	return reflect.TypeOf(e)
}

