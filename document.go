package openrpc_go_document

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/jsonschema"
	"github.com/davecgh/go-spew/spew"
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

// ServerProvider defines a user-defined struct providing necessary
// functions for the document parses to get the information it needs
// to make a complete OpenRPC-schema document.
type ServerProvider struct {
	Callbacks           func() map[string]Callback
	OpenRPCInfo         func() goopenrpcT.Info
	OpenRPCExternalDocs func() goopenrpcT.ExternalDocs
}

// Spec1 is a wrapped type around an openrpc schema document.
type Document struct {
	serverProvider *ServerProvider
	discoverOpts   *DocumentProviderParseOpts
	spec1          *goopenrpcT.OpenRPCSpec1
}

func (d *Document) Spec1() *goopenrpcT.OpenRPCSpec1 {
	return d.spec1
}

// DocumentProvider initializes a Document type given a serverProvider (eg service or aggregate of services)
// and options to use while parsing the runtime code into openrpc types.
func DocumentProvider(serverProvider *ServerProvider, opts *DocumentProviderParseOpts) *Document {
	if serverProvider == nil {
		panic("openrpc-wrap-nil-serverprovider")
	}
	return &Document{serverProvider: serverProvider, discoverOpts: opts}
}

func (d *Document) Discover() (err error) {
	if d == nil || d.serverProvider == nil {
		return errors.New("server provider undefined")
	}

	if d.discoverOpts == nil {
		d.discoverOpts = DefaultDocumentProviderParseOpts
	}

	// TODO: Caching?

	d.spec1 = NewSpec()
	d.spec1.Info = d.serverProvider.OpenRPCInfo()
	d.spec1.ExternalDocs = d.serverProvider.OpenRPCExternalDocs()

	// Set version by runtime, after parse.
	spl := strings.Split(d.spec1.Info.Version, "+")
	d.spec1.Info.Version = fmt.Sprintf("%s-%s-%d", spl[0], time.Now().Format(time.RFC3339), time.Now().Unix())

	callbacks := d.serverProvider.Callbacks()
	d.spec1.Methods = []goopenrpcT.Method{}

	for k, cb := range callbacks {
		if isDiscoverMethodBlacklisted(d.discoverOpts, k) {
			continue
		}

		m, err := d.GetMethod(k, cb)
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

	//if d.discoverOpts != nil && !d.discoverOpts.Inline {
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

var errParseCallbackAutoGenerate = errors.New("autogenerated ethereumCallback")

func (d *Document) GetMethod(name string, cb Callback) (*goopenrpcT.Method, error) {
	pcb, err := newParsedCallback(cb)
	if err != nil {
		if strings.Contains(err.Error(), "autogenerated") {
			return nil, errParseCallbackAutoGenerate
		}
		log.Println("parse ethereumCallback", err)
		return nil, err
	}
	method, err := d.makeMethod(name, pcb)
	if err != nil {
		return nil, fmt.Errorf("make method error method=%s cb=%s error=%v", name, spew.Sdump(cb), err)
	}
	return method, nil
}

func (d *Document) makeMethod(name string, pcb *parsedCallback) (*goopenrpcT.Method, error) {

	argTypes := pcb.cb.getArgTypes()
	retTyptes := pcb.cb.getRetTypes()

	argASTFields := []*NamedField{}
	if pcb.fdecl.Type != nil &&
		pcb.fdecl.Type.Params != nil &&
		pcb.fdecl.Type.Params.List != nil {
		for _, f := range pcb.fdecl.Type.Params.List {
			argASTFields = append(argASTFields, expandASTField(f)...)
		}
	}

	retASTFields := []*NamedField{}
	if pcb.fdecl.Type != nil &&
		pcb.fdecl.Type.Results != nil &&
		pcb.fdecl.Type.Results.List != nil {
		for _, f := range pcb.fdecl.Type.Results.List {
			retASTFields = append(retASTFields, expandASTField(f)...)
		}
	}

	description := func() string {
		return fmt.Sprintf("`%s`", pcb.cb.Func().Type().String())
	}

	contentDescriptor := func(ty reflect.Type, astNamedField *NamedField) (*goopenrpcT.ContentDescriptor, error) {
		sch := d.typeToSchema(ty)
		if d.discoverOpts != nil && len(d.discoverOpts.SchemaMutationFns) > 0 {
			for _, mutation := range d.discoverOpts.SchemaMutationFns {
				if err := mutation(&sch); err != nil {
					return nil, err
				}
			}
		}
		return &goopenrpcT.ContentDescriptor{
			Content: goopenrpcT.Content{
				Name:        astNamedField.Name,
				Summary:     astNamedField.Field.Comment.Text(),
				Required:    true,
				Description: "mydescription", // fullDescriptionOfType(ty),
				Schema:      sch,
			},
		}, nil
	}

	params := func(skipFn func(isArgs bool, index int, descriptor *goopenrpcT.ContentDescriptor) bool) ([]*goopenrpcT.ContentDescriptor, error) {
		out := []*goopenrpcT.ContentDescriptor{}
		for i, a := range argTypes {
			cd, err := contentDescriptor(a, argASTFields[i])
			if err != nil {
				return nil, err
			}
			if skipFn != nil && skipFn(true, i, cd) {
				continue
			}
			for _, fn := range d.discoverOpts.ContentDescriptorMutationFns {
				fn(true, i, cd)
			}
			out = append(out, cd)
		}
		return out, nil
	}

	rets := func(skipFn func(isArgs bool, index int, descriptor *goopenrpcT.ContentDescriptor) bool) ([]*goopenrpcT.ContentDescriptor, error) {
		out := []*goopenrpcT.ContentDescriptor{}
		for i, r := range retTyptes {
			cd, err := contentDescriptor(r, retASTFields[i])
			if err != nil {
				return nil, err
			}
			if skipFn != nil && skipFn(false, i, cd) {
				continue
			}
			for _, fn := range d.discoverOpts.ContentDescriptorMutationFns {
				fn(false, i, cd)
			}
			out = append(out, cd)
		}
		if len(out) == 0 {
			out = append(out, nullContentDescriptor)
		}
		return out, nil
	}

	runtimeFile, runtimeLine := pcb.runtimeF.FileLine(pcb.runtimeF.Entry())

	collectedParams, err := params(d.discoverOpts.ContentDescriptorSkipFn)
	if err != nil {
		return nil, err
	}
	collectedResults, err := rets(d.discoverOpts.ContentDescriptorSkipFn)
	if err != nil {
		return nil, err
	}
	res := collectedResults[0] // OpenRPC Document specific
	return &goopenrpcT.Method{
		Name:        name, // pcb.runtimeF.Name(), // FIXME or give me a comment.
		Tags:        nil,
		Summary:     methodSummary(pcb.fdecl),
		Description: description(),
		ExternalDocs: goopenrpcT.ExternalDocs{
			Description: fmt.Sprintf("line=%d", runtimeLine),
			URL:         fmt.Sprintf("file://%s", runtimeFile), // TODO: Provide WORKING external docs links to Github (actually a wrapper/injection to make this configurable).
		},
		Params:         collectedParams,
		Result:         res,
		Deprecated:     methodDeprecated(pcb.fdecl),
		Servers:        nil,
		Errors:         nil,
		Links:          nil,
		ParamStructure: "",
		Examples:       nil,
	}, nil
}
