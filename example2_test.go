package go_openrpc_reflect

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/etclabscore/go-openrpc-reflect/internal/fakemath"
	meta_schema "github.com/open-rpc/meta-schema"
)

// ExampleDocument_DiscoverStandard2 demonstrates an OpenRPC rpc/document
// implementation with a custom configuration.
func ExampleDocument_DiscoverStandard2() {
	calculatorRPCService := new(fakemath.CalculatorRPC)

	// Assign a new standard lib rpc server.
	server := rpc.NewServer()

	// Set up a listener for our standard lib rpc server.
	// Listen to TPC connections on port 1234
	ricksInternetTube, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Listener:", err)
	}

	go func() {
		defer ricksInternetTube.Close()
		log.Printf("Serving RPC server on port %s", ricksInternetTube.Addr().String())

		// Start accept incoming HTTP connections
		log.Fatal(http.Serve(ricksInternetTube, server))
	}()

	/*
		-------- The non-boilerplate code. <<EOL
		//
	*/

	// Instantiate our document with a custom configuration.
	doc := &Document{}
	doc.WithMeta(MetaT{
		// We're starting an ice cream store!
		GetInfoFn: func() (info *meta_schema.InfoObject) {

			customTitle := "Rick's Ice Cream Store JSON-RPC API, v1"

			return &meta_schema.InfoObject{
				Title: (*meta_schema.InfoObjectProperties)(&customTitle),
			}
		},

		GetServersFn: func() func(listeners []net.Listener) (*meta_schema.Servers, error) {
			return func(listeners []net.Listener) (*meta_schema.Servers, error) {
				servers := []meta_schema.ServerObject{}

				addr := ricksInternetTube.Addr().String()
				network := "shh"

				servers = append(servers, meta_schema.ServerObject{
					Url:  (*meta_schema.ServerObjectUrl)(&addr),
					Name: (*meta_schema.ServerObjectName)(&network),
				})
				return (*meta_schema.Servers)(&servers), nil
			}
		},

		// ... but we're not on Github yet.
		GetExternalDocsFn: func() (exdocs *meta_schema.ExternalDocumentationObject) {
			return nil
		},
	})
	//   .RegisterListener(ricksInternetTube)
	//
	// The GetServersFn assigned in the inline MetaT struct above doesn't ever use the 'listeners' argument value;
	// it uses the in-scope 'ricksInternetTube' listener instead.
	// Because the 'listeners' value isn't ever needed, we can skip the step of registering
	// a listener.
	// If we wanted instead to actually use the service listeners registred to build the document,
	// we use this registration method to register available listeners (ie '"servers": []') with the document service
	// so that they can be evaluated on the fly.

	// Use a sane standard for the mapping (reflecting) Go code to OpenRPC JSON schema data type.
	doc.WithReflector(StandardReflector)

	// Register our calculator service to the rpc.Server and rpc.Doc
	// I've grouped these together because in larger applications
	// multiple receivers may be registered on a single server,
	// and typically receiver registration is done in some kind of loop.
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
	client, err := rpc.DialHTTP("tcp", ricksInternetTube.Addr().String())
	if err != nil {
		log.Fatalf("Error in dialing. %s", err)
	}
	defer client.Close()

	reply := meta_schema.OpenrpcDocument{}
	err = client.Call("RPC.Discover", 0, &reply)
	if err != nil {
		log.Fatal(err)
	}

	replyServers := ([]meta_schema.ServerObject)(*reply.Servers)
	expect := replyServers[0]

	fmt.Println(*reply.Info.Title)
	fmt.Println(*expect.Name)
	// Output:
	// Rick's Ice Cream Store JSON-RPC API, v1
	// shh
}
