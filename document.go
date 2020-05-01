package go_openrpc_reflect

import (
	"errors"
	"fmt"
	"go/ast"
	"net"
	"reflect"
	"sort"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
	meta_schema "github.com/open-rpc/meta-schema"
)

// MetaRegisterer implements methods that must come from the mind of the developer.
// They describe the document (well, provide document description values) that cannot be
// parsed from anything available.
//
type MetaRegisterer interface {
	ServerRegisterer
	GetInfo() func() (info *meta_schema.InfoObject)
	GetExternalDocs() func() (exdocs *meta_schema.ExternalDocumentationObject)
}

// ServerRegisterer implements a method translating a slice of net Listeners into
// document `.servers`.
type ServerRegisterer interface {
	GetServers() func(listeners []net.Listener) (*meta_schema.Servers, error)
}

type ReceiverRegisterer interface {
	MethodRegisterer
	ReceiverMethods(name string, receiver interface{}) ([]meta_schema.MethodObject, error)
}

type MethodRegisterer interface {
	ContentDescriptorRegisterer
	IsMethodEligible(method reflect.Method) bool
	GetMethodName(moduleName string, r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	GetMethodTags(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectTags, error)
	GetMethodDescription(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	GetMethodSummary(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	GetMethodDeprecated(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (bool, error)
	GetMethodParamStructure(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	GetMethodErrors(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectErrors, error)
	GetMethodExternalDocs(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.ExternalDocumentationObject, error)
	GetMethodServers(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.Servers, error)
	GetMethodLinks(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectLinks, error)
	GetMethodExamples(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectExamples, error)
	GetMethodParams(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) ([]meta_schema.ContentDescriptorObject, error)
	GetMethodResult(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (meta_schema.ContentDescriptorObject, error)
}

type ContentDescriptorRegisterer interface {
	SchemaRegisterer
	GetContentDescriptorName(r reflect.Value, m reflect.Method, field *ast.Field) (string, error)
	GetContentDescriptorSummary(r reflect.Value, m reflect.Method, field *ast.Field) (string, error)
	GetContentDescriptorDescription(r reflect.Value, m reflect.Method, field *ast.Field) (string, error)
	GetContentDescriptorRequired(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error)
	GetContentDescriptorDeprecated(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error)
	GetSchema(r reflect.Value, m reflect.Method, field *ast.Field, ty reflect.Type) (schema meta_schema.JSONSchema, err error)
}

type SchemaRegisterer interface {
	// Since our implementation will be piggy-backed on jsonschema.Reflector,
	// this is where our field-by-field Getter abstraction ends.
	// If we didn't rely so heavily on this dependency, we would use
	// a pattern like method and content descriptor, where fields are defined
	// individually per reflector.
	// JSON Schemas have a lot of fields.
	//
	// And including meta_schema.go, we're using 3 data types to develop this object.
	// - alecthomas/jsonschema.Schema: use .Reflector to reflect the schema from its Go declaration.
	// - openapi/spec.Schema: the "official" spec data type from swagger
	// - <generated> meta_schema.go.JSONSchema as eventual local implementation type.
	//
	// Since the native language for this data type is JSON, I (the developer) assume
	// that using the standard lib to Un/Marshal between these data types is as good a glue as any.
	// Unmarshaling will be slow, but should only ever happen once up front, so I'm not concerned with performance.
	//
	// SchemaIgnoredTypes reply will be passed directly to the jsonschema.Reflector.IgnoredTypes field.
	SchemaIgnoredTypes() []interface{}
	// SchemaTypeMap will be passed directory to the jsonschema.Reflector.TypeMapper field.
	SchemaTypeMap() func(ty reflect.Type) *jsonschema.Type
	// SchemaMutations will be run in a depth-first walk on the reflected schema.
	// They will be run in order.
	// Function wrapping allows closure fn to have context of root schema.
	SchemaMutations(ty reflect.Type) []func(*spec.Schema) func(*spec.Schema) error
}

type Service int

const (
	Standard Service = iota
	Ethereum
)

type RPCEthereum struct {
	*Document
}

func (d *RPCEthereum) Discover() (*meta_schema.OpenrpcDocument, error) {
	return d.Document.Discover()
}

type RPC struct {
	*Document
}

type RPCEthereumArg int

func (d *RPC) Discover(arg RPCEthereumArg, document *meta_schema.OpenrpcDocument) error {
	doc, err := d.Document.Discover()
	if err != nil {
		return err
	}
	*document = *doc
	return err
}

type Document struct {
	meta          MetaRegisterer
	reflector     ReceiverRegisterer
	receiverNames []string
	receivers     []interface{}
	listeners     []net.Listener
}

func (d *Document) RPCDiscover(kind Service) (receiver interface{}) {
	switch kind {
	case Standard:
		return &RPC{d}
	case Ethereum:
		return &RPCEthereum{d}
	}
	return nil
}

func (d *Document) RegisterReceiver(receiver interface{}) {
	d.RegisterReceiverName("", receiver)
}

func (d *Document) RegisterReceiverName(name string, receiver interface{}) {
	if d.receivers == nil {
		d.receivers = []interface{}{}
	}
	if d.receiverNames == nil {
		d.receiverNames = []string{}
	}
	d.receiverNames = append(d.receiverNames, name)
	d.receivers = append(d.receivers, receiver)
}

func (d *Document) RegisterListener(listener net.Listener) {
	if d.listeners == nil {
		d.listeners = []net.Listener{}
	}
	d.listeners = append(d.listeners, listener)
}

func (d *Document) WithMeta(meta MetaRegisterer) *Document {
	d.meta = meta
	return d
}

func (d *Document) WithReflector(reflector ReceiverRegisterer) *Document {
	d.reflector = reflector
	return d
}

var errMissingInterface = errors.New("missing interface")

func (d *Document) Discover() (*meta_schema.OpenrpcDocument, error) {

	if d.meta == nil {
		return nil, fmt.Errorf("meta: %v", errMissingInterface)
	}

	openRPCDocumentVersion := meta_schema.OpenrpcEnum0
	out := &meta_schema.OpenrpcDocument{
		Openrpc:      &openRPCDocumentVersion,
		Info:         d.meta.GetInfo()(),         // This will panic if the developer misuses it (leaves it nil).
		ExternalDocs: d.meta.GetExternalDocs()(), // This too.
	}

	getServersFn := d.meta.GetServers()
	servers, err := getServersFn(d.listeners)
	if err != nil {
		return nil, err
	}
	out.Servers = servers

	// Return no error if no receivers registered.
	// > While it is required, the array may be empty (to handle security filtering, for example).
	// > https://spec.open-rpc.org/#openrpc-object
	if d.reflector == nil {
		return out, nil
	}
	if d.receivers == nil || len(d.receivers) == 0 {
		return out, nil
	}

	// Iterate all registered receivers (aka 'modules'),
	// building and collecting eligible methods for each.
	methods := []meta_schema.MethodObject{}
	for i, rec := range d.receivers {
		name := d.receiverNames[i]
		ms, err := d.reflector.ReceiverMethods(name, rec)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ms...)
	}

	sort.Slice(methods, func(i, j int) bool {
		return *methods[i].Name < *methods[j].Name
	})

	// Assign by slice address.
	m := meta_schema.Methods(methods)
	out.Methods = &m

	return out, nil
}
