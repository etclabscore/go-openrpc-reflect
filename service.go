package openrpc_go_document

import goopenrpcT "github.com/gregdhill/go-openrpc/types"

type EthereumService struct {
	doc *Document
}

func NewDiscoverService(d *Document) *EthereumService {
	return &EthereumService{doc: d}
}

func (s *EthereumService) Discover() (*goopenrpcT.OpenRPCSpec1, error) {
	return s.doc.Spec1(), nil
}
