package go_openrpc_reflect

import (
	"fmt"
	"log"
	"net"
	"reflect"

	"github.com/alecthomas/jsonschema"
	"github.com/etclabscore/go-openrpc-reflect/internal/fakemath"
	ethereumRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/tidwall/gjson"
)

// ExampleDocument_DiscoverStandard demonstrates a basic application implementation
// of the OpenRPC document service, this time with a custom SchemaTypeMap function, which
// is passed as a configuration option to the jsonschema library for reflecting types to jsonschemas.
func ExampleDocument_DiscoverEthereum2() {

	// Assign a new go-ethereum/rpc lib rpc server.
	server := ethereumRPC.NewServer()

	// Use the "base" receiver instead of the RPC-wrapped receiver ("CalculatorRPC").
	// Ethereum has different opinions about how to register methods of a receiver
	// to make an RPC service, so we don't need to use the wrapper the standard way
	// requires.
	calculatorService := new(fakemath.Calculator)

	err := server.RegisterName("calculator", calculatorService)
	if err != nil {
		log.Fatal(err)
	}

	/*
		< The non-boilerplate code. <<EOL
	*/
	// Instantiate our document with sane defaults.
	doc := &Document{}
	doc.WithMeta(TestMetaRegisterer)     // Note that this is the TEST registerer. The Meta interface must be defined by the application.

	//  Instantiate registration defaults.
	appReflector := EthereumReflectorT{}

	// Override as needed.
	appReflector.FnSchemaTypeMap = func () func(ty reflect.Type) *jsonschema.Type {
		return func(ty reflect.Type) *jsonschema.Type {
			if ty.Kind() == reflect.Ptr {
				ty = ty.Elem()
			}
			if ty.Kind() == reflect.Int {
				return &jsonschema.Type{Type: "integer", Title: "myInteger"}
			}
			return nil
		}
	}
	doc.WithReflector(&appReflector)

	// Register our calculator service to the rpc.Server and rpc.Doc
	// I've grouped these together because in larger applications
	// multiple receivers may be registered on a single server,
	// and typically receiver registration is done in some kind of loop.
	doc.RegisterReceiver(calculatorService)

	// Now here's the good bit.
	// Register the OpenRPC Document service back to the rpc.Server.
	err = server.RegisterName("rpc", doc.RPCDiscover(Ethereum))
	if err != nil {
		log.Fatal(err)
	}
	/* EOL */

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Listener:", err)
	}
	defer listener.Close()

	go server.ServeListener(listener)

	// Send a test request.
	request := `{"jsonrpc":"2.0","id":1,"method":"rpc_discover", "params":[]}` + "\n"

	back, err := makeRequest(request, listener)
	if err != nil {
		log.Fatal(err)
	}
	got := gjson.GetBytes(back, "openrpc")
	fmt.Println(got.Value())

	got = gjson.GetBytes(back, "methods.0.params.0.schema.title")
	fmt.Println(got.Value())

	// Output:
	// 1.2.4
	// myInteger
}
