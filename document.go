package openrpc_go_document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/jsonschema"
	"github.com/davecgh/go-spew/spew"
	jst "github.com/etclabscore/go-jsonschema-traverse"
	"github.com/go-openapi/spec"
	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

type MutateType string

const (
	SchemaMutateType_Expand            = "schema_expand"
	SchemaMutateType_RemoveDefinitions = "schema_remove_definitions"
)

type DocumentDiscoverOpts struct {
	Inline            bool
	SchemaMutations   []MutateType
	SchemaMutationFns []func(s *spec.Schema) error
	MethodBlackList   []string

	// TypeMapper gets passed directly to the jsonschema reflection library.
	TypeMapper        func(r reflect.Type) *jsonschema.Type
	IgnoredTypes      []interface{}
}

type argIdent struct {
	ident *ast.Ident
	name  string
}

func (a argIdent) Name() string {
	if a.ident != nil {
		return a.ident.Name
	}
	return a.name
}

type ServerProvider interface {
	Methods() map[string][]reflect.Value
	OpenRPCInfo() goopenrpcT.Info
	OpenRPCExternalDocs() goopenrpcT.ExternalDocs
}

type Document struct {
	serverProvider ServerProvider
	discoverOpts   *DocumentDiscoverOpts
	spec1          *goopenrpcT.OpenRPCSpec1
}

func (d *Document) Document() *goopenrpcT.OpenRPCSpec1 {
	return d.spec1
}

func Wrap(serverProvider ServerProvider, opts *DocumentDiscoverOpts) *Document {
	if serverProvider == nil {
		panic("openrpc-wrap-nil-serverprovider")
	}
	return &Document{serverProvider: serverProvider, discoverOpts: opts}
}

func isDiscoverMethodBlacklisted(d *DocumentDiscoverOpts, name string) bool {
	if d != nil && len(d.MethodBlackList) > 0 {
		for _, b := range d.MethodBlackList {
			if regexp.MustCompile(b).MatchString(name) {
				return true
			}
		}
	}
	return false
}

func (d *Document) Discover() (doc *goopenrpcT.OpenRPCSpec1, err error) {
	if d == nil || d.serverProvider == nil {
		return nil, errors.New("server provider undefined")
	}

	// TODO: Caching?

	d.spec1 = NewSpec()
	d.spec1.Info = d.serverProvider.OpenRPCInfo()
	d.spec1.ExternalDocs = d.serverProvider.OpenRPCExternalDocs()

	// Set version by runtime, after parse.
	spl := strings.Split(d.spec1.Info.Version, "+")
	d.spec1.Info.Version = fmt.Sprintf("%s-%s-%d", spl[0], time.Now().Format(time.RFC3339), time.Now().Unix())

	d.spec1.Methods = []goopenrpcT.Method{}
	mets := d.serverProvider.Methods()

	for k, rvals := range mets {
		if rvals == nil || len(rvals) == 0 {
			fmt.Println("skip bad k", k)
			continue
		}

		if isDiscoverMethodBlacklisted(d.discoverOpts, k) {
			continue
		}

		m, err := d.GetMethod(k, rvals)
		if err != nil {
			return nil, err
		}
		d.spec1.Methods = append(d.spec1.Methods, *m)
	}
	sort.Slice(d.spec1.Methods, func(i, j int) bool {
		return d.spec1.Methods[i].Name < d.spec1.Methods[j].Name
	})

	if d.discoverOpts != nil && len(d.discoverOpts.SchemaMutations) > 0 {
		for _, mutation := range d.discoverOpts.SchemaMutations {
			d.documentRunSchemasMutation(mutation)
		}
	}
	//if d.discoverOpts != nil && !d.discoverOpts.Inline {
	//	d.spec1.Components.Schemas = make(map[string]spec.Schema)
	//}

	// TODO: Flatten/Inline ContentDescriptors and Schemas

	return d.spec1, nil
}

func removeDefinitionsFieldSchemaMutation(s *spec.Schema) error {
	s.Definitions = nil
	return nil
}

func expandSchemaMutation(s *spec.Schema) error {
	return spec.ExpandSchema(s, s, nil)
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
func (d *Document) documentRunSchemasMutation(id MutateType) {
	switch id {
	case SchemaMutateType_Expand:
		d.documentSchemaMutation(expandSchemaMutation)
	case SchemaMutateType_RemoveDefinitions:
		d.documentSchemaMutation(removeDefinitionsFieldSchemaMutation)
	}
}

func documentGetAstFunc(rcvr reflect.Value, fn reflect.Value, astFile *ast.File, rf *runtime.Func) *ast.FuncDecl {
	rfName := runtimeFuncName(rf)
	for _, decl := range astFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name == nil || fn.Name.Name != rfName {
			continue
		}
		fnRecName := ""
		for _, l := range fn.Recv.List {
			if fnRecName != "" {
				break
			}
			i, ok := l.Type.(*ast.Ident)
			if ok {
				fnRecName = i.Name
				continue
			}
			s, ok := l.Type.(*ast.StarExpr)
			if ok {
				fnRecName = fmt.Sprintf("%v", s.X)
			}
		}

		if rcvr.IsValid() && !rcvr.IsNil() {
			reRec := regexp.MustCompile(fnRecName + `\s`)
			if !reRec.MatchString(rcvr.String()) {
				continue
			}
		}
		return fn
	}
	return nil
}

func (d *Document) GetMethod(name string, fns []reflect.Value) (*goopenrpcT.Method, error) {
	var recvr reflect.Value
	var fn reflect.Value

	if len(fns) == 2 && fns[0].IsValid() && fns[1].IsValid() {
		recvr, fn = fns[0], fns[1]
	} else if len(fns) == 1 {
		fn = fns[0]
	}

	rtFunc := runtime.FuncForPC(fn.Pointer())
	cbFile, _ := rtFunc.FileLine(rtFunc.Entry())

	tokenFileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(tokenFileSet, cbFile, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	astFuncDecl := documentGetAstFunc(recvr, fn, astFile, rtFunc)

	if astFuncDecl == nil {
		return nil, fmt.Errorf("nil ast func: method name: %s", name)
	}

	method, err := d.documentMakeMethod(name, recvr, fn, rtFunc, astFuncDecl)
	if err != nil {
		return nil, fmt.Errorf("make method error method=%s cb=%s error=%v", name, spew.Sdump(fn), err)
	}
	return &method, nil
}

func NewSpec() *goopenrpcT.OpenRPCSpec1 {
	return &goopenrpcT.OpenRPCSpec1{
		OpenRPC: "1.2.4",
		Info:    goopenrpcT.Info{},
		Servers: []goopenrpcT.Server{},
		Methods: []goopenrpcT.Method{},
		Components: goopenrpcT.Components{
			ContentDescriptors:    make(map[string]*goopenrpcT.ContentDescriptor),
			Schemas:               make(map[string]spec.Schema),
			Examples:              make(map[string]goopenrpcT.Example),
			Links:                 make(map[string]goopenrpcT.Link),
			Errors:                make(map[string]goopenrpcT.Error),
			ExamplePairingObjects: make(map[string]goopenrpcT.ExamplePairing),
			Tags:                  make(map[string]goopenrpcT.Tag),
		},
		ExternalDocs: goopenrpcT.ExternalDocs{},
		Objects:      goopenrpcT.NewObjectMap(),
	}
}

func documentGetArgTypes(rcvr, val reflect.Value) (argTypes []reflect.Type) {
	fntype := val.Type()
	// Skip receiver and context.Context parameter (if present).
	firstArg := 0
	if rcvr.IsValid() && !rcvr.IsNil() {
		firstArg++
	}
	if fntype.NumIn() > firstArg && fntype.In(firstArg) == contextType {
		firstArg++
	}
	// Add all remaining parameters.
	argTypes = make([]reflect.Type, fntype.NumIn()-firstArg)
	for i := firstArg; i < fntype.NumIn(); i++ {
		argTypes[i-firstArg] = fntype.In(i)
	}
	return
}
func documentGetRetTypes(val reflect.Value) (retTypes []reflect.Type) {
	fntype := val.Type()
	// Add all remaining parameters.
	retTypes = make([]reflect.Type, fntype.NumOut())
	for i := 0; i < fntype.NumOut(); i++ {
		retTypes[i] = fntype.Out(i)
	}
	return
}

func documentValHasContext(rcvr reflect.Value, val reflect.Value) bool {
	fntype := val.Type()
	// Skip receiver and context.Context parameter (if present).
	firstArg := 0
	if rcvr.IsValid() && !rcvr.IsNil() {
		firstArg++
	}
	return fntype.NumIn() > firstArg && fntype.In(firstArg) == contextType
}

func (d *Document) documentMakeMethod(name string, rcvr reflect.Value, cb reflect.Value, rt *runtime.Func, fn *ast.FuncDecl) (goopenrpcT.Method, error) {
	file, line := rt.FileLine(rt.Entry())

	m := goopenrpcT.Method{
		Name:    name,
		Tags:    []goopenrpcT.Tag{},
		Summary: fn.Doc.Text(),
		//Description: fmt.Sprintf("```\n%s\n```", string(buf.Bytes())), // rt.Name(),
		//  fmt.Sprintf("`%s`\n> [%s:%d][file://%s]", rt.Name(), file, line, file),
		//Description: "some words",
		ExternalDocs: goopenrpcT.ExternalDocs{
			Description: rt.Name(),
			URL:         fmt.Sprintf("file://%s:%d", file, line),
		},
		Params:         []*goopenrpcT.ContentDescriptor{},
		Result:         &goopenrpcT.ContentDescriptor{},
		Deprecated:     false,
		Servers:        []goopenrpcT.Server{},
		Errors:         []goopenrpcT.Error{},
		Links:          []goopenrpcT.Link{},
		ParamStructure: "by-position",
		Examples:       []goopenrpcT.ExamplePairing{},
	}

	defer func() {
		//if m.Result.Name == "" {
		//	m.Result.Name = "null"
		//	m.Result.Schema.Type = []string{"null"}
		//	m.Result.Schema.Description = "Null"
		//}
	}()

	argTypes := documentGetArgTypes(rcvr, cb)
	if fn.Type.Params != nil {
		j := 0
		for _, field := range fn.Type.Params.List {
			if field == nil {
				continue
			}
			if documentValHasContext(rcvr, cb) && strings.Contains(fmt.Sprintf("%s", field.Type), "context") {
				continue
			}
			if len(field.Names) > 0 {
				for _, ident := range field.Names {
					if ident == nil {
						continue
					}
					if j > len(argTypes)-1 {
						log.Println(name, argTypes, field.Names, j)
						continue
					}
					cd, err := d.makeContentDescriptor(argTypes[j], field, argIdent{ident, fmt.Sprintf("%sParameter%d", name, j)})
					if err != nil {
						return m, err
					}
					j++
					m.Params = append(m.Params, &cd)
				}
			} else {
				cd, err := d.makeContentDescriptor(argTypes[j], field, argIdent{nil, fmt.Sprintf("%sParameter%d", name, j)})
				if err != nil {
					return m, err
				}
				j++
				m.Params = append(m.Params, &cd)
			}

		}
	}
	retTypes := documentGetRetTypes(cb)
	if fn.Type.Results != nil {
		j := 0
		for _, field := range fn.Type.Results.List {

			// Always take the first return value. (Second would be an error typically).
			if len(m.Result.Schema.Type) > 0 {
				break
			}

			if field == nil {
				continue
			}
			//if errorType == retTypes[j] || strings.Contains(fmt.Sprintf("%s", field.Type), "error") {
			//	continue
			//}
			if len(field.Names) > 0 {
				// This really should never ever happen I don't think.
				// JSON-RPC returns _an_ result. So there can't be > 1 return value.
				// But just in case.
				for _, ident := range field.Names {
					cd, err := d.makeContentDescriptor(retTypes[j], field, argIdent{ident, fmt.Sprintf("%sResult%d", name, j)})
					if err != nil {
						return m, err
					}
					j++
					m.Result = &cd
				}
			} else {
				cd, err := d.makeContentDescriptor(retTypes[j], field, argIdent{nil, fmt.Sprintf("%s", retTypes[j].Name())})
				if err != nil {
					return m, err
				}
				j++
				m.Result = &cd
			}
		}
	}

	return m, nil
}

func runtimeFuncName(rf *runtime.Func) string {
	spl := strings.Split(rf.Name(), ".")
	return spl[len(spl)-1]
}

func (d *Document) makeContentDescriptor(ty reflect.Type, field *ast.Field, ident argIdent) (goopenrpcT.ContentDescriptor, error) {
	cd := goopenrpcT.ContentDescriptor{}
	if !jsonschemaPkgSupport(ty) {
		return cd, fmt.Errorf("unsupported iface: %v %v %v", spew.Sdump(ty), spew.Sdump(field), spew.Sdump(ident))
	}

	schemaType := fmt.Sprintf("%s:%s", ty.PkgPath(), ty.Name())
	switch tt := field.Type.(type) {
	case *ast.SelectorExpr:
		schemaType = fmt.Sprintf("%v.%v", tt.X, tt.Sel)
		schemaType = fmt.Sprintf("%s:%s", ty.PkgPath(), schemaType)
	case *ast.StarExpr:
		schemaType = fmt.Sprintf("%v", tt.X)
		schemaType = fmt.Sprintf("*%s:%s", ty.PkgPath(), schemaType)
		if reflect.ValueOf(ty).Type().Kind() == reflect.Ptr {
			schemaType = fmt.Sprintf("%v", ty.Elem().Name())
			schemaType = fmt.Sprintf("*%s:%s", ty.Elem().PkgPath(), schemaType)
		}
	default:

		//ty = ty.Elem() // FIXME: wart warn
	}
	//schemaType = fmt.Sprintf("%s:%s", ty.PkgPath(), schemaType)

	cd.Name = ident.Name()

	cd.Summary = field.Comment.Text()              // field.Doc.Text()
	cd.Description = fmt.Sprintf("%s", schemaType) // field.Comment.Text()

	var typeMapper func(reflect.Type) *jsonschema.Type
	if d.discoverOpts != nil {
		typeMapper = d.discoverOpts.TypeMapper
	}

	var ignoredTypes []interface{}
	if d.discoverOpts != nil {
		ignoredTypes = d.discoverOpts.IgnoredTypes
	}

	rflctr := jsonschema.Reflector{
		AllowAdditionalProperties:  true, // false,
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             false, // false, // false,
		TypeMapper:                 typeMapper,
		IgnoredTypes:               ignoredTypes,
	}

	jsch := rflctr.ReflectFromType(ty)

	// Poor man's type cast.
	// Need to get the type from the go struct -> json reflector package
	// to the swagger/go-openapi/jsonschema spec.
	// Do this with JSON marshaling.
	// Hacky? Maybe. Effective? Maybe.
	m, err := json.Marshal(jsch)
	if err != nil {
		log.Fatal(err)
	}
	sch := spec.Schema{}
	err = json.Unmarshal(m, &sch)
	if err != nil {
		log.Fatal(err)
	}
	// End Hacky maybe.
	if schemaType != ":" && (cd.Schema.Description == "" || cd.Schema.Description == ":") {
		sch.Description = schemaType
	}

	cd.Schema = sch

	return cd, nil
}

func jsonschemaPkgSupport(r reflect.Type) bool {
	switch r.Kind() {
	case reflect.Struct,
		reflect.Map,
		reflect.Slice, reflect.Array,
		reflect.Interface,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool,
		reflect.String,
		reflect.Ptr:
		return true
	default:
		return false
	}
}
