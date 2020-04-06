package openrpc_go_document

import (
	"context"
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

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

type MutateType string

type DocumentProviderParseOpts struct {
	SchemaMutationFns            []func(s *spec.Schema) error
	ContentDescriptorMutationFns []func(isArgs bool, index int, cd *goopenrpcT.ContentDescriptor)
	MethodBlackList              []string

	// TypeMapper gets passed directly to the jsonschema reflection library.
	TypeMapper              func(r reflect.Type) *jsonschema.Type
	SchemaIgnoredTypes      []interface{}
	ContentDescriptorSkipFn func(isArgs bool, index int, cd *goopenrpcT.ContentDescriptor) bool
}

type ServiceProvider interface {
	Callbacks() map[string]Callback
	CallbackToMethod(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error)
	OpenRPCInfo() goopenrpcT.Info
	OpenRPCExternalDocs() *goopenrpcT.ExternalDocs
}

// ServerProviderService defines a user-defined struct providing necessary
// functions for the document parses to get the information it needs
// to make a complete OpenRPC-schema document.
type ServerProviderService struct {
	ServiceCallbacks           func() map[string]Callback
	ServiceCallbackToMethod    func(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error)
	ServiceOpenRPCInfo         func() goopenrpcT.Info
	ServiceOpenRPCExternalDocs func() *goopenrpcT.ExternalDocs
}

func (s *ServerProviderService) Callbacks() map[string]Callback {
	return s.ServiceCallbacks()
}

func (s *ServerProviderService) CallbackToMethod(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error) {
	return s.ServiceCallbackToMethod(opts, name, cb)
}

func (s *ServerProviderService) OpenRPCInfo() goopenrpcT.Info {
	return s.ServiceOpenRPCInfo()
}

func (s *ServerProviderService) OpenRPCExternalDocs() *goopenrpcT.ExternalDocs {
	if s.ServiceOpenRPCExternalDocs != nil {
		return s.ServiceOpenRPCExternalDocs()
	}
	return nil
}

// Spec1 is a wrapped type around an openrpc schema document.
type Document struct {
	serverProvider ServiceProvider
	parseOpts      *DocumentProviderParseOpts
	spec1          *goopenrpcT.OpenRPCSpec1
}

func (d *Document) Spec1() *goopenrpcT.OpenRPCSpec1 {
	return d.spec1
}

// DocumentProvider initializes a Document type given a serverProvider (eg service or aggregate of services)
// and options to use while parsing the runtime code into openrpc types.
func DocumentProvider(serverProvider ServiceProvider, opts *DocumentProviderParseOpts) *Document {
	if serverProvider == nil {
		panic("openrpc-wrap-nil-serverprovider")
	}
	return &Document{serverProvider: serverProvider, parseOpts: opts}
}

func (d *Document) Discover() (err error) {
	if d == nil || d.serverProvider == nil {
		return errors.New("server provider undefined")
	}

	if d.parseOpts == nil {
		d.parseOpts = DefaultParseOptions()
	}

	// TODO: Caching?

	d.spec1 = NewSpec()
	d.spec1.Info = d.serverProvider.OpenRPCInfo()
	if externalDocs := d.serverProvider.OpenRPCExternalDocs(); externalDocs != nil {
		d.spec1.ExternalDocs = *externalDocs
	}

	// Set version by runtime, after parse.
	spl := strings.Split(d.spec1.Info.Version, "+")
	d.spec1.Info.Version = fmt.Sprintf("%s-%s-%d", spl[0], time.Now().Format(time.RFC3339), time.Now().Unix())

	callbacks := d.serverProvider.Callbacks()
	d.spec1.Methods = []goopenrpcT.Method{}

	for k, cb := range callbacks {
		if isDiscoverMethodBlacklisted(d.parseOpts, k) {
			continue
		}

		m, err := d.serverProvider.CallbackToMethod(d.parseOpts, k, cb)
		//m, err := d.GetMethod(k, cb)
		if err == errParseCallbackAutoGenerate {
			continue
		}
		if m == nil || err != nil {
			return err
		}

		d.spec1.Methods = append(d.spec1.Methods, *m)
	}
	sort.Slice(d.spec1.Methods, func(i, j int) bool {
		return d.spec1.Methods[i].Name < d.spec1.Methods[j].Name
	})

	//if d.parseOpts != nil && !d.parseOpts.Inline {
	//	d.spec1.Components.Schemas = make(map[string]spec.Schema)
	//}

	// TODO: Flatten/Inline ContentDescriptors and Schemas

	return nil
}

func (d *Document) FlattenSchemas() *Document {

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

func (d *Document) documentMethodsSchemaMutation(mut func(s *spec.Schema) error) {
	a := jst.NewAnalysisT()
	for i := 0; i < len(d.spec1.Methods); i++ {

		met := d.spec1.Methods[i]

		// Params.
		for ip := 0; ip < len(met.Params); ip++ {
			par := met.Params[ip]
			a.WalkDepthFirst(&par.Schema, mut)
			met.Params[ip] = par
		}

		// Result (single).
		a.WalkDepthFirst(&met.Result.Schema, mut)
	}
	//for k := range d.spec1.Components.ContentDescriptors {
	//	cd := d.spec1.Components.ContentDescriptors[k]
	//	a.WalkDepthFirst(&cd.Schema, mut)
	//	d.spec1.Components.ContentDescriptors[k] = cd
	//}
	//for k := range d.spec1.Components.Schemas {
	//	s := d.spec1.Components.Schemas[k]
	//	a.WalkDepthFirst(&s, mut)
	//	d.spec1.Components.Schemas[k] = s
	//}
}

/*
	TODO:
	FlattenContentDescriptors is not yet possible without goopenrpc implementing
	the alternative Reference and/or OneOf object spec.
*/
//func contentDescriptorKey(cd *goopenrpcT.ContentDescriptor) string {
//	b, _ := json.Marshal(cd)
//	sum := sha1.Sum(b)
//	return fmt.Sprintf(`%s_%x`, cd.Name, sum[:4])
//}
//func (d *Document) FlattenContentDescriptors() *Document {
//
//	// Assume d.spec1.ContentDescriptors is initialized.
//	for i, m := range d.spec1.Methods {
//		for j, cd := range m.Params {
//			id := contentDescriptorKey(cd)
//			d.spec1.Components.ContentDescriptors[id] = cd
//
//			cp := &goopenrpcT.ContentDescriptor{}
//			cp.Content.Schema.Ref = spec.Ref{
//				Ref: jsonreference.MustCreateRef("#/components/contentDescriptors/" + id),
//			}
//			m.Params[j] = cp
//		}
//		id := contentDescriptorKey(m.Result)
//		d.spec1.Components.ContentDescriptors[id] = m.Result
//		cp := &goopenrpcT.ContentDescriptor{}
//		cp.Content.Schema.Ref = spec.Ref{
//			Ref: jsonreference.MustCreateRef("#/components/contentDescriptors/" + id),
//		}
//		m.Result = cp
//		d.spec1.Methods[i] = m
//	}
//	return d
//}

func (d *Document) Inline() *Document {
	return nil
}