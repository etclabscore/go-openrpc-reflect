package examples

import (
	"net"
	"time"

	go_openrpc_reflect "github.com/etclabscore/go-openrpc-reflect"
	meta_schema "github.com/open-rpc/meta-schema"
)

type RPC struct {
	Doc *go_openrpc_reflect.Document
}

type RPCArg int // noop

func (d *RPC) Discover(rpcArg *RPCArg, document *meta_schema.OpenrpcDocument) error {
	doc, err := d.Doc.Discover()
	if err != nil {
		return err
	}
	*document = *doc
	return err
}

type RPCEthereum struct {
	Doc *go_openrpc_reflect.Document
}

func (d *RPCEthereum) Discover() (*meta_schema.OpenrpcDocument, error) {
	return d.Doc.Discover()
}

//var ExampleMetaReflector = &MetaRegistererTester{}
var ExampleMetaReflector = &go_openrpc_reflect.MetaT{
	GetServersFn:      getServers,
	GetInfoFn:         getInfo,
	GetExternalDocsFn: getExternalDocs,
}

func getInfo() (info *meta_schema.InfoObject) {
	title := "Calculator API"
	version := time.Now().Format(time.RFC3339)
	return &meta_schema.InfoObject{
		Title:          (*meta_schema.InfoObjectProperties)(&title),
		Description:    nil,
		TermsOfService: nil,
		Version:        (*meta_schema.InfoObjectVersion)(&version),
		Contact:        nil,
		License:        nil,
	}
}

func getExternalDocs() (*meta_schema.ExternalDocumentationObject) {
	return nil
}

func getServers() func (listeners []net.Listener) (*meta_schema.Servers, error) {
	return go_openrpc_reflect.StandardReflector.GetServers()
}

func newDocument() *go_openrpc_reflect.Document {
	return &go_openrpc_reflect.Document{}
}
