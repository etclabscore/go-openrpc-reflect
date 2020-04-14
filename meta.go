package go_openrpc_refract

import (
	"net"

	meta_schema "github.com/open-rpc/meta-schema"
)

// MetaT implements the MetaRegisterer interface.
// An application can use this struct to define an inline
// interface for an OpenRPC document.
type MetaT struct {
	GetServersFn func () func (listeners []net.Listener) (*meta_schema.Servers, error)
	GetInfoFn  func () (info *meta_schema.InfoObject)
	GetExternalDocsFn func () (exdocs *meta_schema.ExternalDocumentationObject)
}

func (m MetaT) GetServers() func (listeners []net.Listener) (*meta_schema.Servers, error) {
	return m.GetServersFn()
}

func (m MetaT) GetInfo() func() (info *meta_schema.InfoObject) {
	return m.GetInfoFn
}

func (m MetaT) GetExternalDocs() func() (exdocs *meta_schema.ExternalDocumentationObject) {
	return m.GetExternalDocsFn
}


