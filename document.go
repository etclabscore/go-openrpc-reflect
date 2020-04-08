package openrpc_go_document

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/jsonschema"
	jst "github.com/etclabscore/go-jsonschema-traverse"
	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

type Discoverer interface {
	Discover() (*goopenrpcT.OpenRPCSpec1, error)
}

type DocumentProviderParseOpts struct {
	SchemaMutationFromTypeFns    []func(s *spec.Schema, ty reflect.Type)
	SchemaMutationFns            []func(s *spec.Schema) error
	ContentDescriptorMutationFns []func(isArgs bool, index int, cd *goopenrpcT.ContentDescriptor)

	MethodBlackList             []string
	ContentDescriptorTypeSkipFn func(isArgs bool, index int, ty reflect.Type, cd *goopenrpcT.ContentDescriptor) bool

	// TypeMapper gets passed directly to the jsonschema reflection library.
	TypeMapper func(r reflect.Type) *jsonschema.Type

	// SchemaIgnoredTypes also gets passed directly to the jsonschema reflection library.
	SchemaIgnoredTypes []interface{}
}


/*
RPCServerServiceProvider provides service information common
to a all methods of an API service, ie the server.

It is a single sibling of the
potentially-many ReceiverServiceConfigurationProvider(s).
*/
type RPCServerServiceProvider interface {
	OpenRPCInfo() goopenrpcT.Info
	OpenRPCExternalDocs() *goopenrpcT.ExternalDocs
}

type RPCServerServiceProviderService struct {
	ServiceOpenRPCInfoFn         func() goopenrpcT.Info
	ServiceOpenRPCExternalDocsFn func() *goopenrpcT.ExternalDocs
}

func (s *RPCServerServiceProviderService) OpenRPCInfo() goopenrpcT.Info {
	return s.ServiceOpenRPCInfoFn()
}

func (s *RPCServerServiceProviderService) OpenRPCExternalDocs() *goopenrpcT.ExternalDocs {
	if s.ServiceOpenRPCExternalDocsFn != nil {
		return s.ServiceOpenRPCExternalDocsFn()
	}
	return nil
}

type ReceiverServiceConfigurationProvider interface {
	ParseOptions() *DocumentProviderParseOpts
	MethodName(receiver interface{}, receiverName, methodName string) string
	Callbacks(receiver interface{}) map[string]Callback
	CallbackToMethod(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error)
}

// ReceiverServiceConfigurationProviderService defines a user-defined struct providing necessary
// functions for the document parses to get the information it needs
// to make a complete OpenRPC-schema document.
type ReceiverServiceConfigurationProviderService struct {
	ProviderParseOptions               *DocumentProviderParseOpts
	ServiceCallbacksFullyQualifiedName func(receiver interface{}, receiverName, methodName string) string
	ServiceCallbacksFromReceiverFn     func(receiver interface{}) map[string]Callback
	ServiceCallbackToMethodFn          func(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error)
}

func (s *ReceiverServiceConfigurationProviderService) ParseOptions() *DocumentProviderParseOpts {
	return s.ProviderParseOptions
}

func (s *ReceiverServiceConfigurationProviderService) MethodName(receiver interface{}, receiverName, methodName string) string {
	return s.ServiceCallbacksFullyQualifiedName(receiver, receiverName, methodName)
}

func (s *ReceiverServiceConfigurationProviderService) Callbacks(receiver interface{}) map[string]Callback {
	return s.ServiceCallbacksFromReceiverFn(receiver)
}

func (s *ReceiverServiceConfigurationProviderService) CallbackToMethod(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error) {
	return s.ServiceCallbackToMethodFn(opts, name, cb)
}

type ReflectedDocument struct {
	rpcServerServiceProvider              RPCServerServiceProvider
	receiverServiceConfigurationProviders []ReceiverServiceConfigurationProvider
	receiverNames                         []string
	receiverServices                      []interface{}
	callbacks                             map[string]Callback // cache?
	spec1                                 *goopenrpcT.OpenRPCSpec1
}

func (r *ReflectedDocument) RegisterReceiver(receiver interface{}, provider ReceiverServiceConfigurationProvider) {
	r.registerReceiverWithName("", receiver, provider)
}

func (d *ReflectedDocument) RegisterReceiverWithName(name string, receiver interface{}, provider ReceiverServiceConfigurationProvider) {
	d.registerReceiverWithName(name, receiver, provider)
}

func (s *ReflectedDocument) registerReceiverWithName(name string, receiver interface{}, provider ReceiverServiceConfigurationProvider) {
	if len(s.receiverNames) == 0 {
		s.receiverNames = []string{}
	}
	if len(s.receiverServices) == 0 {
		s.receiverServices = []interface{}{}
	}
	if len(s.receiverServiceConfigurationProviders) == 0 {
		s.receiverServiceConfigurationProviders = []ReceiverServiceConfigurationProvider{}
	}
	s.receiverNames = append(s.receiverNames, name)
	s.receiverServices = append(s.receiverServices, receiver)
	s.receiverServiceConfigurationProviders = append(s.receiverServiceConfigurationProviders, provider)
}

func (r *ReflectedDocument) Discover() (*goopenrpcT.OpenRPCSpec1, error) {
	if r.spec1 != nil {
		return r.spec1, nil
	}

	if r == nil || r.receiverServiceConfigurationProviders == nil {
		return nil, errors.New("server provider undefined")
	}

	r.spec1 = NewSpec()

	r.spec1.Info = r.rpcServerServiceProvider.OpenRPCInfo()
	if eDocs := r.rpcServerServiceProvider.OpenRPCExternalDocs(); eDocs != nil {
		r.spec1.ExternalDocs = *eDocs
	}

	// Set version by runtime, after parse.
	spl := strings.Split(r.spec1.Info.Version, "+")
	r.spec1.Info.Version = fmt.Sprintf("%s-%s-%d", spl[0], time.Now().Format(time.RFC3339), time.Now().Unix())

	r.spec1.Methods = []goopenrpcT.Method{}

	for i := 0; i < len(r.receiverNames); i++ {
		receiverName := r.receiverNames[i]
		receiverService := r.receiverServices[i]
		serviceConfigurationProvider := r.receiverServiceConfigurationProviders[i]

		methods := []goopenrpcT.Method{}

		callbacks := serviceConfigurationProvider.Callbacks(receiverService)
		for methodName, cb := range callbacks {
			if isDiscoverMethodBlacklisted(serviceConfigurationProvider.ParseOptions(), methodName) {
				continue
			}

			// Get fully qualified method name.
			methodName = serviceConfigurationProvider.MethodName(receiverService, receiverName, methodName)

			// Get method
			m, err := serviceConfigurationProvider.CallbackToMethod(serviceConfigurationProvider.ParseOptions(), methodName, cb)
			if err == errParseCallbackAutoGenerate {
				continue
			}
			if m == nil || err != nil {
				return nil, err
			}

			methods = append(methods, *m)
		}

		r.spec1.Methods = append(r.spec1.Methods, methods...)

	}
	sort.Slice(r.spec1.Methods, func(i, j int) bool {
		return r.spec1.Methods[i].Name < r.spec1.Methods[j].Name
	})

	return r.spec1, nil
}

type StaticDocument struct {
	raw []byte
}

func (s *StaticDocument) Discover() (*goopenrpcT.OpenRPCSpec1, error) {
	if len(s.raw) == 0 {
		return nil, errors.New("missing raw document")
	}
	out := &goopenrpcT.OpenRPCSpec1{}
	err := json.Unmarshal(s.raw, out)
	return out, err
}

// Spec1 is a wrapped type around an openrpc schema document.
type Document struct {
	Discoverer
	Static    *StaticDocument
	Reflector *ReflectedDocument
}

// NewReflectDocument initializes a Document type given a receiverServiceConfigurationProviders (eg service or aggregate of services)
// and options to use while parsing the runtime code into openrpc types.
func NewReflectDocument(serverProvider RPCServerServiceProvider) *Document {
	d := &Document{}
	d.Reflector = &ReflectedDocument{
		rpcServerServiceProvider: serverProvider,
	}
	return d
}

func NewStaticDocument(input []byte) *Document {
	d := &Document{}
	d.Static = &StaticDocument{raw: input}
	return d
}

func (d *Document) Discover() (*goopenrpcT.OpenRPCSpec1, error) {
	if d.Static != nil {
		return d.Static.Discover()
	} else if d.Reflector != nil {
		return d.Reflector.Discover()
	}
	return nil, errors.New("empty document")
}

func (d *ReflectedDocument) FlattenSchemas() *ReflectedDocument {

	d.documentMethodsSchemaMutation(func(s *spec.Schema) error {
		id := schemaKey(*s)
		d.spec1.Components.Schemas[id] = *s
		ss := spec.Schema{}
		ss.Ref = spec.Ref{
			Ref: jsonreference.MustCreateRef("#/components/schemas/" + id),
		}
		*s = ss
		return nil
	})

	return d
}

func schemaKey(schema spec.Schema) string {
	b, _ := json.Marshal(schema)
	sum := sha1.Sum(b)
	return fmt.Sprintf(`%s_%s_%x`, schema.Title, strings.Join(schema.Type, "+"), sum[:4])
}

func (r *ReflectedDocument) documentMethodsSchemaMutation(mut func(s *spec.Schema) error) {
	a := jst.NewAnalysisT()
	for i := 0; i < len(r.spec1.Methods); i++ {

		met := r.spec1.Methods[i]

		// Params.
		for ip := 0; ip < len(met.Params); ip++ {
			par := met.Params[ip]
			a.WalkDepthFirst(&par.Schema, mut)
			met.Params[ip] = par
		}

		// Result (single).
		a.WalkDepthFirst(&met.Result.Schema, mut)
	}
}

func (d *ReflectedDocument) Inline() *Document {
	return nil
}
