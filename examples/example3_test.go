package examples

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	go_openrpc_reflect "github.com/etclabscore/go-openrpc-reflect"
	"github.com/etclabscore/go-openrpc-reflect/internal/fakearithmetic"
	ethereumRPC "github.com/ethereum/go-ethereum/rpc"
)

// ExampleDocument_DiscoverStandard demonstrates a basic application implementation
// of the OpenRPC document service.
func ExampleDocument_DiscoverEthereum() {

	// Assign a new go-ethereum/rpc lib rpc server.
	server := ethereumRPC.NewServer()

	// Use the "base" receiver instead of the RPC-wrapped receiver ("CalculatorRPC").
	// Ethereum has different opinions about how to register methods of a receiver
	// to make an RPC service, so we don't need to use the wrapper the standard way
	// requires.
	calculatorService := new(fakearithmetic.Calculator)

	err := server.RegisterName("calculator", calculatorService)
	if err != nil {
		log.Fatal(err)
	}

	/*
		< The non-boilerplate code. <<EOL
	*/
	// Instantiate our document with sane defaults.
	doc := &go_openrpc_reflect.Document{}
	doc.WithMeta(ExampleMetaReflector)                      // Note that this is the TEST registerer. The Meta interface must be defined by the application.
	doc.WithReflector(go_openrpc_reflect.EthereumReflector) // Use a sane standard designed for the ethereum/go-ethereum/rpc package.

	// Register our calculator service to the rpc.Server and rpc.Doc
	// I've grouped these together because in larger applications
	// multiple receivers may be registered on a single server,
	// and typically receiver registration is done in some kind of loop.
	doc.RegisterReceiver(calculatorService)

	// Now here's the good bit.
	// Register the OpenRPC Document service back to the rpc.Server.
	err = server.RegisterName("rpc", &RPCEthereum{doc})
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

	m := make(map[string]interface{})
	err = json.Unmarshal(back, &m)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(m["openrpc"])
	// Output: 1.2.6
}

func makeRequest(request string, listener net.Listener) ([]byte, error) {

	deadline := time.Now().Add(10 * time.Second)

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(deadline)

	log.Println("--> ", request)

	_, err = conn.Write([]byte(request))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).CloseWrite()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024*1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	log.Println("<-- ", string(buf[:]))

	pretty := make(map[string]interface{})
	err = json.Unmarshal(buf[:n], &pretty)
	if err != nil {
		return nil, err
	}

	if _, ok := pretty["error"]; ok {
		return nil, fmt.Errorf("%v", pretty)
	}

	bufPretty, err := json.MarshalIndent(pretty["result"], "", "  ")
	if err != nil {
		return nil, err
	}
	return bufPretty, nil
}
