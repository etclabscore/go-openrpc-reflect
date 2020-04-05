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
	// Assume d.spec1.ContentDescriptors is initialized.
	for i, m := range d.spec1.Methods {
		// Params.
		for j, cd := range m.Params {

			id := schemaKey(cd.Schema)
			d.spec1.Components.Schemas[id] = cd.Schema

			cd.Schema = spec.Schema{}
			cd.Schema.Ref= spec.Ref{
				Ref: jsonreference.MustCreateRef("#/components/schemas/" + id),
			}
			//cp := &goopenrpcT.ContentDescriptor{}
			//cp.Content.Schema.Ref =
			m.Params[j] = cd
		}

		// Result.
		id := schemaKey(m.Result.Schema)
		d.spec1.Components.Schemas[id] = m.Result.Schema

		m.Result.Schema = spec.Schema{}
		m.Result.Schema.Ref = spec.Ref{
			Ref: jsonreference.MustCreateRef("#/components/schemas/" + id),
		}
		d.spec1.Methods[i] = m
	}
	return d
}

func schemaKey(schema spec.Schema) string {
	b, _ := json.Marshal(schema)
	sum := sha1.Sum(b)
	return fmt.Sprintf(`%s_%s_%x`, schema.Title, strings.Join(schema.Type, "+"), sum[:4])
}

func contentDescriptorKey(cd *goopenrpcT.ContentDescriptor) string {
	b, _ := json.Marshal(cd)
	sum := sha1.Sum(b)
	return fmt.Sprintf(`%s_%x`, cd.Name, sum[:4])
}

/*
	TODO:
	FlattenContentDescriptors is not yet possible without goopenrpc implementing
	the alternative Reference and/or OneOf object spec.
*/
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

var errParseCallbackAutoGenerate = errors.New("autogenerated callback")

func (d *Document) GetMethod(name string, cb Callback) (*goopenrpcT.Method, error) {
	pcb, err := newParsedCallback(cb)
	if err != nil {
		if strings.Contains(err.Error(), "autogenerated") {
			return nil, errParseCallbackAutoGenerate
		}
		log.Println("parse callback", err)
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

func (d *Document) documentSchemaMutation(mut func(s *spec.Schema) error) {
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
	for k := range d.spec1.Components.ContentDescriptors {
		cd := d.spec1.Components.ContentDescriptors[k]
		a.WalkDepthFirst(&cd.Schema, mut)
		d.spec1.Components.ContentDescriptors[k] = cd
	}
	for k := range d.spec1.Components.Schemas {
		s := d.spec1.Components.Schemas[k]
		a.WalkDepthFirst(&s, mut)
		d.spec1.Components.Schemas[k] = s
	}
}

//func (d *Document) documentMakeMethod(name string, rcvr reflect.Value, cpFn reflect.Value, rt *runtime.Func, fn *ast.FuncDecl) (goopenrpcT.Method, error) {
//	file, line := rt.FileLine(rt.Entry())
//
//	m := goopenrpcT.Method{
//		Name:    name,
//		Tags:    []goopenrpcT.Tag{},
//		Summary: fn.Doc.Text(),
//		//Description: fmt.Sprintf("```\n%s\n```", string(buf.Bytes())), // rt.Name(),
//		//  fmt.Sprintf("`%s`\n> [%s:%d][file://%s]", rt.Name(), file, line, file),
//		//Description: "some words",
//		ExternalDocs: goopenrpcT.ExternalDocs{
//			Description: rt.Name(),
//			URL:         fmt.Sprintf("file://%s:%d", file, line),
//		},
//		Params:         []*goopenrpcT.ContentDescriptor{},
//		Result:         &goopenrpcT.ContentDescriptor{},
//		Deprecated:     false,
//		Servers:        []goopenrpcT.Server{},
//		Errors:         []goopenrpcT.Error{},
//		Links:          []goopenrpcT.Link{},
//		ParamStructure: "by-position",
//		Examples:       []goopenrpcT.ExamplePairing{},
//	}
//
//	defer func() {
//		//if m.Result.Name == "" {
//		//	m.Result.Name = "null"
//		//	m.Result.Schema.Type = []string{"null"}
//		//	m.Result.Schema.Description = "Null"
//		//}
//	}()
//
//	argTypes := documentGetArgTypes(rcvr, cpFn)
//	if fn.Type.Params != nil {
//		j := 0
//		for _, field := range fn.Type.Params.List {
//			if field == nil {
//				continue
//			}
//			if documentValHasContext(rcvr, cpFn) && strings.Contains(fmt.Sprintf("%s", field.Type), "context") {
//				continue
//			}
//			if len(field.Names) > 0 {
//				for _, ident := range field.Names {
//					if ident == nil {
//						continue
//					}
//					if j > len(argTypes)-1 {
//						log.Println(name, argTypes, field.Names, j)
//						continue
//					}
//					cd, err := d.makeContentDescriptor(argTypes[j], field, argIdent{ident, fmt.Sprintf("%sParameter%d", name, j)})
//					if err != nil {
//						return m, err
//					}
//					j++
//					m.Params = append(m.Params, &cd)
//				}
//			} else {
//				cd, err := d.makeContentDescriptor(argTypes[j], field, argIdent{nil, fmt.Sprintf("%sParameter%d", name, j)})
//				if err != nil {
//					return m, err
//				}
//				j++
//				m.Params = append(m.Params, &cd)
//			}
//
//		}
//	}
//	retTypes := documentGetRetTypes(cpFn)
//	if fn.Type.Results != nil {
//		j := 0
//		for _, field := range fn.Type.Results.List {
//
//			// Always take the first return value. (Second would be an error typically).
//			if len(m.Result.Schema.Type) > 0 {
//				break
//			}
//
//			if field == nil {
//				continue
//			}
//			//if errorType == retTypes[j] || strings.Contains(fmt.Sprintf("%s", field.Type), "error") {
//			//	continue
//			//}
//			if len(field.Names) > 0 {
//				// This really should never ever happen I don't think.
//				// JSON-RPC returns _an_ result. So there can't be > 1 return value.
//				// But just in case.
//				for _, ident := range field.Names {
//					cd, err := d.makeContentDescriptor(retTypes[j], field, argIdent{ident, fmt.Sprintf("%sResult%d", name, j)})
//					if err != nil {
//						return m, err
//					}
//					j++
//					m.Result = &cd
//				}
//			} else {
//				cd, err := d.makeContentDescriptor(retTypes[j], field, argIdent{nil, fmt.Sprintf("%s", retTypes[j].Name())})
//				if err != nil {
//					return m, err
//				}
//				j++
//				m.Result = &cd
//			}
//		}
//	}
//
//	return m, nil
//}
//
//
//func (d *Document) makeContentDescriptor(ty reflect.Type, field *ast.Field, ident argIdent) (goopenrpcT.ContentDescriptor, error) {
//	cd := goopenrpcT.ContentDescriptor{}
//	if !jsonschemaPkgSupport(ty) {
//		return cd, fmt.Errorf("unsupported iface: %v %v %v", spew.Sdump(ty), spew.Sdump(field), spew.Sdump(ident))
//	}
//
//	schemaType := fmt.Sprintf("%s:%s", ty.PkgPath(), ty.Name())
//	switch tt := field.Type.(type) {
//	case *ast.SelectorExpr:
//		schemaType = fmt.Sprintf("%v.%v", tt.X, tt.Sel)
//		schemaType = fmt.Sprintf("%s:%s", ty.PkgPath(), schemaType)
//	case *ast.StarExpr:
//		schemaType = fmt.Sprintf("%v", tt.X)
//		schemaType = fmt.Sprintf("*%s:%s", ty.PkgPath(), schemaType)
//		if reflect.ValueOf(ty).Type().Kind() == reflect.Ptr {
//			schemaType = fmt.Sprintf("%v", ty.Elem().Name())
//			schemaType = fmt.Sprintf("*%s:%s", ty.Elem().PkgPath(), schemaType)
//		}
//	default:
//
//		//ty = ty.Elem() // FIXME: wart warn
//	}
//	//schemaType = fmt.Sprintf("%s:%s", ty.PkgPath(), schemaType)
//
//	cd.Name = ident.Name()
//
//	cd.Summary = field.Comment.Text()              // field.Doc.Text()
//	cd.Description = fmt.Sprintf("%s", schemaType) // field.Comment.Text()
//
//	var typeMapper func(reflect.Type) *jsonschema.Type
//	if d.discoverOpts != nil {
//		typeMapper = d.discoverOpts.TypeMapper
//	}
//
//	var ignoredTypes []interface{}
//	if d.discoverOpts != nil {
//		ignoredTypes = d.discoverOpts.SchemaIgnoredTypes
//	}
//
//	rflctr := jsonschema.Reflector{
//		AllowAdditionalProperties:  true, // false,
//		RequiredFromJSONSchemaTags: true,
//		ExpandedStruct:             false, // false, // false,
//		TypeMapper:                 typeMapper,
//		IgnoredTypes:               ignoredTypes,
//	}
//
//	jsch := rflctr.ReflectFromType(ty)
//
//	// Poor man's glue.
//	// Need to get the type from the go struct -> json reflector package
//	// to the swagger/go-openapi/jsonschema spec.
//	// Do this with JSON marshaling.
//	// Hacky? Maybe. Effective? Maybe.
//	m, err := json.Marshal(jsch)
//	if err != nil {
//		log.Fatal(err)
//	}
//	sch := spec.Schema{}
//	err = json.Unmarshal(m, &sch)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// End Hacky maybe.
//	if schemaType != ":" && (cd.Schema.Description == "" || cd.Schema.Description == ":") {
//		sch.Description = schemaType
//	}
//
//	cd.Schema = sch
//
//	return cd, nil
//}
//
