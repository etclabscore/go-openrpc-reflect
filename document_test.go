package openrpc_go_document

import (
	"context"
	"encoding/json"
	"errors"
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

/*
ETHEREUM RPC Style.
*/

// MyEthereumService will be a receiver.
type MyEthereumService struct {
}

// MyVal is an mock struct type.
// Some of its fields have comments, and it demonstrates how json/jsonschema tags work.
type MyVal struct {
	Name string `jsonschema:"required"`
	Age  int    `json:"oldness" jsonschema:"required"`

	// CellsN describes how many cells in the body.
	CellN *big.Int `jsonschema:"required"`

	Exists   bool // jsonschema:not_required
	thoughts []string
}

// Fetch is a mock method.
func (m *MyEthereumService) Fetch() (result *MyVal, err error) {
	return &MyVal{}, nil
}

// Do is a very sparse mock method.
func (m *MyEthereumService) Do(string) {

}

// Make is a busy method with lots of comments.
// More notes. Is Deprecated.
func (m *MyEthereumService) Make(ctx context.Context /*name is thing*/, name string /*name is thing*/, cellsN *big.Int) (myResult MyVal, err error) {
	/*
		Make does things!
	*/
	return MyVal{Name: name, CellN: cellsN}, nil
}

// Produce will not be included in the standard RPC service: it does not use a pointer receiver.
func (m MyEthereumService) Produce(name string) error {
	return nil
}

//  callCounter will not be included in the standard RPC service: is a private method.
func (m *MyEthereumService) callCounter() (n int, err error) {
	return 14, nil
}

// NoReceiverFunction is a public function (without a receiver).
// Yes! It can still be documented, if you want.
func NoReceiverFunction(name string) (n int, err error) {
	return 42, nil
}

/*
STANDARD RPC Style.
*/

// MyStandardService implements a very basic standard lib rpc service.
type MyStandardService struct{}

type MyStandardAddRPCArg string
type MyStandardAddRPCReply int

func (r *MyStandardService) Add(input MyStandardAddRPCArg, response *MyStandardAddRPCReply) (err error) {
	return nil
}

type MyStandardRandomRPCArg string
type MyStandardRandomRPCReply int

func (r *MyStandardService) RandomNum(noopString MyStandardRandomRPCArg, response *MyStandardRandomRPCReply) error {
	*response = 42
	return nil
}

type MyStandardIsZeroRPCArg int

type MyStandardIsZeroRPCReply struct {
	Explanation  string
	Mansplaining bool
}

func (r *MyStandardService) IsZeroNum(input MyStandardIsZeroRPCArg, response *MyStandardIsZeroRPCReply) error {
	if input == 0 {
		response = &MyStandardIsZeroRPCReply{"Probably close", false}
	}
	return errors.New("could not explain a number that is probably not zero")
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
}

const testTermsOfServiceURI = "https://github.com/etclabscore/openrpc-go-document/blob/master/LICENSE.md"

func TestDocument_Discover(t *testing.T) {

	t.Run("EthereumRPC", func(t *testing.T) {

		// Get our service.
		ethereumService := new(MyEthereumService)

		// Wrap service to create serviceProvider.
		serviceProvider := DefaultEthereumServiceProvider(ethereumService)

		// Adjust settings from default.
		serviceProvider.ServiceOpenRPCInfo = func() goopenrpcT.Info {
			return goopenrpcT.Info{
				Title:          "My Ethereum Service",
				Description:    "Oooohhh!",
				TermsOfService: testTermsOfServiceURI,
				Contact:        goopenrpcT.Contact{},
				License:        goopenrpcT.License{},
				Version:        "v0.0.0-beta",
			}
		}

		// Get our document provider from the serviceProvider.
		doc := DocumentProvider(serviceProvider, DefaultEthereumParseOptions())

		// Discover method introspects methods at runtime and
		// instantiates a reflective OpenRPC schema document.
		err := doc.Discover()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(doc.Spec1().Methods); l != 3 {
			t.Fatal("methods", l)
		}

		if l := len(doc.Spec1().Components.Schemas); l != 0 {
			// Not been flattened yet.
			t.Fatal("schemas", l)
		}

		doc.FlattenSchemas()

		if l := len(doc.Spec1().Components.Schemas); l != 8 {
			// Not been flattened yet.
			t.Fatal("flat schemas", l)
		}

		b, _ := json.MarshalIndent(doc.Spec1(), "", "  ")
		t.Logf(string(b))

		err = ioutil.WriteFile(filepath.Join("testdata", "output", "ethereum.json"), b, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

	})

	t.Run("StandardRPC", func(t *testing.T) {
		standardService := new(MyStandardService)
		provider := DefaultStandardRPCServiceProvider(standardService)
		provider.ServiceOpenRPCInfo = func() goopenrpcT.Info {
			return goopenrpcT.Info{
				Title:          "My Standard Service",
				Description:    "Aaaaahh!",
				TermsOfService: testTermsOfServiceURI,
				Contact:        goopenrpcT.Contact{},
				License:        goopenrpcT.License{},
				Version:        "v0.0.0-beta",
			}
		}
		doc := DocumentProvider(provider, DefaultParseOptions())

		err := doc.Discover()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(doc.Spec1().Methods); l != 3 {
			t.Fatal("methods", l)
		}

		if l := len(doc.Spec1().Components.Schemas); l != 0 {
			// Not been flattened yet.
			t.Fatal("schemas", l)
		}

		doc.FlattenSchemas()

		if l := len(doc.Spec1().Components.Schemas); l != 8 {
			// Not been flattened yet.
			t.Fatal("flat schemas", l)
		}

		b, _ := json.MarshalIndent(doc.Spec1(), "", "  ")
		t.Logf(string(b))

		err = ioutil.WriteFile(filepath.Join("testdata", "output", "standard.json"), b, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	})
}
