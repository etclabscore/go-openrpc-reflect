package go_openrpc_reflect

import meta_schema "github.com/open-rpc/meta-schema"

type RPCEthereum struct {
	Doc *Document
}

func (d *RPCEthereum) Discover() (*meta_schema.OpenrpcDocument, error) {
	return d.Doc.Discover()
}

type RPC struct {
	Doc *Document
}

type RPCArg int

func (d *RPC) Discover(rpcArg *RPCArg, document *meta_schema.OpenrpcDocument) error {
	doc, err := d.Doc.Discover()
	if err != nil {
		return err
	}
	*document = *doc
	return err
}
