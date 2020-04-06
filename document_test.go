package openrpc_go_document

import (
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/gregdhill/go-openrpc/types"
)

// ---
/*
	Mock types and methods and functions.
*/

// MyService will be a receiver.
type MyService struct {
}

// MyVal is an mock struct type.
// Some of its fields have comments, and it demonstrates how json/jsonschema tags work.
type MyVal struct {
	Name string `jsonschema:"required"`
	Age  int    `json:"oldness" jsonschema:"required"`

	// CellsN describes how many cells in the body.
	CellN *big.Int `jsonschema:"required"`

	Exists bool // jsonschema:not_required
	thoughts []string
}

// Fetch is a mock method.
func (m *MyService) Fetch() (result *MyVal, err error) {
	return &MyVal{}, nil
}

// Do is a very sparse mock method.
func (m *MyService) Do(string) {

}

// Make is a busy method with lots of comments.
// More notes. Is Deprecated.
func (m *MyService) Make(ctx context.Context /*name is thing*/, name string /*name is thing*/, cellsN *big.Int) (myResult MyVal, err error) {
	/*
		Make does things!
	*/
	return MyVal{Name: name, CellN: cellsN}, nil
}

// Produce is a pretty plain method.
func (m MyService) Produce(name string) error {
	return nil
}

// callCounter is a private method.
func (m *MyService) callCounter() (n int, err error) {
	return 14, nil
}

// LoneRandomer is a public function (without a receiver).
// Yes! It can still be documented, if you want.
func LoneRandomer(name string) (n int, err error) {
	return 42, nil
}


// go/rpc
type GoRPCRecvr struct{}
type GoRPCArg string
type GoRPCRes map[string]interface{}

func (r *GoRPCRecvr) Add(input GoRPCArg, response *GoRPCRes) (err error) {
	return nil
}

func (r *GoRPCRecvr) RandomNum(response *GoRPCRes) error {
	res := map[string]interface{}{
		"answer": 42,
	}
	*response = GoRPCRes(res)
	return nil
}

// ---
/*
	Actual tests.
*/

func TestCallback_HasReceiver(t *testing.T) {
	cb := Callback{NoReceiverValue, reflect.ValueOf(LoneRandomer)}
	if cb.HasReceiver() {
		t.Fatal("bad")
	}
}

func TestDocument_Discover(t *testing.T) {
	serv := new(MyService)
	callbacks := GoRPCServiceMethods(serv)()
	callbacks["custom_method"] = Callback{NoReceiverValue, reflect.ValueOf(LoneRandomer)}
	callbacksFn := func() map[string]Callback {
		return callbacks
	}
	doc := DocumentProvider(&ServerProviderService{
		ServiceCallbacks: callbacksFn,
		ServiceOpenRPCInfo: func() types.Info {
			return types.Info{}
		},
		ServiceOpenRPCExternalDocs: func() types.ExternalDocs {
			return types.ExternalDocs{}
		},
	}, &DocumentProviderParseOpts{
		SchemaMutationFns: []func(s *spec.Schema) error{
			SchemaMutationRequireDefaultOn,
			SchemaMutationExpand,
			SchemaMutationRemoveDefinitionsField,
		},
		ContentDescriptorMutationFns: nil,
		MethodBlackList:              nil,
		TypeMapper:                   nil,
		SchemaIgnoredTypes:           nil,
		ContentDescriptorSkipFn:      nil,
	})

	err := doc.Discover()
	if err != nil {
		t.Fatal(err)
	}

	doc.FlattenSchemas()

	definition := doc.Spec1()
	b, _ := json.MarshalIndent(definition, "", "  ")
	t.Logf(string(b))
}
