package openrpc_go_document

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

// ---
/*
	Mock types and methods and functions.
*/

// MyBasicService will be a receiver.
type MyBasicService struct {
	mocking
	parameterA, parameterB int
	stateFullData map[string]*MyBasicObject
}

type mocking struct {
	callCount int
	methods []string
	args [][]interface{}
}

func (m *mocking) setCalledWith(method string, args ...interface{}) {
	m.callCount++
	if len(m.methods) == 0 {
		m.methods = []string{}
	}
	if len(m.args) == 0 {
		m.args = [][]interface{}{}
	}
	m.methods = append(m.methods, method)
	m.args = append(m.args, []interface{}{args})
}

// MyBasicObject is an mock struct type.
// Some of its fields have comments, and it demonstrates how json/jsonschema tags work.
type MyBasicObject struct {
	Name string `jsonschema:"required"`
	Age  int    `json:"oldness" jsonschema:"required"`

	// CellsN describes how many cells in the body.
	CellN *big.Int `jsonschema:"required"` // CellsN comments on the side.

	Exists   bool // jsonschema:not_required
	thoughts []string
}

// GetAll returns all the objects contained in state.
func (m *MyBasicService) GetAll() (result []*MyBasicObject, err error) {
	m.mocking.setCalledWith("GetAll")
	result = []*MyBasicObject{}
	for _, v := range m.stateFullData {
		result = append(result, v)
	}
	return result, nil
}

// Set places a value in state.
func (m *MyBasicService) Set(value MyBasicObject) (err error) {
	m.mocking.setCalledWith("Set", value)
	if m.stateFullData == nil {
		m.stateFullData = make(map[string]*MyBasicObject)
	}
	m.stateFullData[value.Name] = &value
	return nil
}

// GetDisguisedName gets a sort-of disguised name for a user.
func (m *MyBasicService) GetDisguisedName(name string) (disguise string, err error) {
	m.mocking.setCalledWith("EncodedName", name)
	if v, ok := m.stateFullData[name]; !ok {
		return "", errors.New("no object by that name")
	} else {
		sum := sha1.Sum([]byte(v.Name))
		return fmt.Sprintf(`%s, aka %x`, v.Name, sum[:4]), nil
	}
}

// OperateOnExternalType accepts a type imported from another package as it's argument.
func (m *MyBasicService) OperateOnExternalType(externalObject *goopenrpcT.ExamplePairing) error {

	return nil
}

/*
Implement an API for Go Standard library RPC for our MyBasicService.
*/

type MyBasicServiceRPC struct {
	mocking
	base *MyBasicService
}

// GetAllArg is the argument accepted by the standard RPC service GetAll method.
// Required: false.
type GetAllArg interface{}

// GetAllReply is the kind of response returned by the standard RPC service GetAll method.
type GetAllReply []*MyBasicObject

// GetAll is an RPC wrapper method around the MyBasicService `GetAll` method.
func (rpc *MyBasicServiceRPC) GetAll(arg GetAllArg, reply *GetAllReply) error {
	rpc.mocking.setCalledWith("GetAll", arg)
	got, err := rpc.base.GetAll()
	if err != nil { return err }
	*reply = got
	return nil
}

// ToggleInternalThing is an unobservable method without parameters; it uses neither args nor reply.
// It has primitive types as arguments which you can set to arbitrary values.
func (rpc *MyBasicServiceRPC) ToggleInternalThing(uint, *float64) error {
	rpc.mocking.setCalledWith("ToggleInternalThing")
	return nil
}

type SetArg MyBasicObject
type SetReply interface{}

// Set is an RPC wrapper method.
func (rpc *MyBasicServiceRPC) Set(arg SetArg, reply *SetReply) (err error) {
	rpc.mocking.setCalledWith("Set", arg)
	err = rpc.base.Set(MyBasicObject(arg))
	return
}


type GetDisguisedNameReply string

// GetDisguisedName is also a wrapper method.
func (rpc *MyBasicServiceRPC) GetDisguisedName(name string, reply *GetDisguisedNameReply) error {
	rpc.mocking.setCalledWith("GetDisguisedName", name)
	name, err := rpc.base.GetDisguisedName(name)
	if err != nil {
		return err
	}
	*reply = GetDisguisedNameReply(name)
	return nil
}

type OperateOnExternalTypeReply interface{}
// OperateOnExternalType accepts a type imported from another package as it's argument.
func (m *MyBasicServiceRPC) OperateOnExternalType(arg *goopenrpcT.ExamplePairing, reply *OperateOnExternalTypeReply) error {
	m.mocking.setCalledWith("OperateOnExternalType", arg)
	return nil
}

/*
An exemplary function that can be served without a receiver.
*/

// NoReceiverFunction is a public function (without a receiver).
// Yes! It can still be documented, if you want.
func NoReceiverFunction(name string) (n int, err error) {
	return 42, nil
}

// ---
/*
	Actual tests.
*/

func TestCallback_HasReceiver(t *testing.T) {
	cb := Callback{NoReceiverValue, reflect.ValueOf(NoReceiverFunction)}
	if cb.HasReceiver() {
		t.Fatal("bad")
	}

	cb2 := Callback{reflect.ValueOf(nil), reflect.ValueOf(NoReceiverFunction)}
	if cb2.HasReceiver() {
		t.Fatal("bad")
	}
}

const thing = "aset"

const testTermsOfServiceURI = "https://github.com/etclabscore/openrpc-go-document/blob/master/LICENSE.md"

func TestDocument_Discover(t *testing.T) {

	t.Run("EthereumRPC", func(t *testing.T) {

		// Instantiate our service.
		ethereumService := new(MyBasicService)

		// Server settings are top-level; one ~~server~~ API, one document.
		serverConfigurationP := DefaultServerServiceProvider
		serverConfigurationP.ServiceOpenRPCInfoFn = func() goopenrpcT.Info {
			return goopenrpcT.Info{
				Title:          "My Ethereum-Style Service",
				Description:    "Oooohhh!",
				TermsOfService: testTermsOfServiceURI,
				Contact:        goopenrpcT.Contact{},
				License:        goopenrpcT.License{},
				Version:        "v0.0.0-beta",
			}
		}

		// Get our document provider from the serviceProvider.
		serverDoc := NewReflectDocument(serverConfigurationP)

		rp := EthereumRPCDescriptor
		serverDoc.Reflector.RegisterReceiver(ethereumService, rp)


		// Discover method introspects methods at runtime and
		// instantiates a reflective OpenRPC schema document.
		spec1, err := serverDoc.Reflector.Discover()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(spec1.Methods); l != 4 {
			t.Fatal("methods", l)
		}

		if l := len(spec1.Components.Schemas); l != 0 {
			// Not been flattened yet.
			t.Fatal("schemas", l)
		}

		serverDoc.Reflector.FlattenSchemas()

		if l := len(spec1.Components.Schemas); l != 18 {
			// Not been flattened yet.
			t.Fatal("flat schemas", l)
		}

		b, _ := json.MarshalIndent(spec1, "", "  ")
		t.Logf(string(b))

		err = ioutil.WriteFile(filepath.Join("testdata", "output", "ethereum.json"), b, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

	})

	t.Run("StandardRPC", func(t *testing.T) {

		standardService := new(MyBasicServiceRPC)

		// Server settings are top-level; one ~~server~~ API, one document.
		//
		// Get the server (not service!) provider default.
		serverConfigurationP := DefaultServerServiceProvider

		// Modify the server config.
		serverConfigurationP.ServiceOpenRPCInfoFn = func() goopenrpcT.Info {
			return goopenrpcT.Info{
				Title:          "My Standard Service",
				Description:    "Aaaaahh!",
				TermsOfService: testTermsOfServiceURI,
				Contact:        goopenrpcT.Contact{},
				License:        goopenrpcT.License{},
				Version:        "v0.0.0-beta",
			}
		}

		// Create a new "reflectable" document for this server.
		serverDoc := NewReflectDocument(serverConfigurationP)

		// Get the service provider default for our RPC API style,
		// in this case, the Go standard lib.
		sp := StandardRPCDescriptor

		// Register our receiver-based service standardService.
		serverDoc.Reflector.RegisterReceiver(standardService, sp)

		// serverDoc.Discover() is a shortcut for either Static or Reflected discovery.
		spec1, err := serverDoc.Discover()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(spec1.Methods); l != 5 {
			t.Fatal("methods", l)
		}

		if l := len(spec1.Components.Schemas); l != 0 {
			// Not been flattened yet.
			t.Fatal("schemas", l)
		}

		serverDoc.Reflector.FlattenSchemas()

		if l := len(spec1.Components.Schemas); l != 27 {
			// Not been flattened yet.
			t.Fatal("flat schemas", l)
		}

		b, _ := json.MarshalIndent(spec1, "", "  ")
		t.Logf(string(b))

		err = ioutil.WriteFile(filepath.Join("testdata", "output", "standard.json"), b, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	})
}
