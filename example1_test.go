package go_openrpc_reflect

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/etclabscore/go-openrpc-refract/internal/fakemath"
	meta_schema "github.com/open-rpc/meta-schema"
)

// ExampleDocument_DiscoverStandard demonstrates a basic application implementation
// of the OpenRPC document service.
func ExampleDocument_DiscoverStandard() {
	calculatorRPCService := new(fakemath.CalculatorRPC)

	// Assign a new standard lib rpc server.
	server := rpc.NewServer()

	// Set up a listener for our standard lib rpc server.
	// Listen to TPC connections on port 1234
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Listener:", err)
	}

	go func() {
		defer listener.Close()
		log.Printf("Serving RPC server on port %s", listener.Addr().String())

		// Start accept incoming HTTP connections
		err = http.Serve(listener, server)
		if err != nil {
			log.Fatal("Serve:", err)
		}
	}()

	/*
		-------- The non-boilerplate code. <<EOL
		//
	*/

	// Instantiate our document with sane defaults.
	doc := &Document{}
	doc.WithMeta(TestMetaRegisterer)     // Note that this is the TEST registerer. The Meta interface must be defined by the application.
	doc.WithReflector(StandardReflector) // Use a sane standard.

	// Register our calculator service to the rpc.Server and rpc.Doc
	// I've grouped these together because in larger applications
	// multiple receivers may be registered on a single server,
	// and typically receiver registration is done in some kind of loop.
	// NOTE that net/rpc will log warnings like:
	//   > rpc.Register: method "BrokenReset" has 1 input parameters; needs exactly three'
	// This is because internal/fakemath has spurious methods for testing this package.
	err = server.Register(calculatorRPCService)
	if err != nil {
		log.Fatal(err)
	}
	doc.RegisterReceiver(calculatorRPCService)

	// Now here's the good bit.
	// Register the OpenRPC Document service back to the rpc.Server.
	err = server.Register(doc.RPCDiscover(Standard))
	if err != nil {
		log.Fatal(err)
	}

	/*
		// That's it!
		-------- EOL
	*/

	// Now, let's test it with a client.
	client, err := rpc.DialHTTP("tcp", listener.Addr().String())
	if err != nil {
		log.Fatalf("Error in dialing. %s", err)
	}
	defer client.Close()

	reply := meta_schema.OpenrpcDocument{}
	err = client.Call("RPC.Discover", 0, &reply)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(*reply.Openrpc)
	// Output: 1.2.4
}
