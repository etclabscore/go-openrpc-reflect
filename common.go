package go_openrpc_reflect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/alecthomas/jsonschema"
	go_jsonschema_walk "github.com/etclabscore/go-jsonschema-walk"
	"github.com/go-openapi/spec"
	meta_schema "github.com/open-rpc/meta-schema"
)

var errType = reflect.TypeOf((*error)(nil)).Elem()

var nullContentDescriptor meta_schema.ContentDescriptorObject
var nullSchema meta_schema.JSONSchema

func init() {
	nullS := "Null"

	var nullT interface{}
	nullT = "null"

	required, deprecated := true, false

	nullSchema = meta_schema.JSONSchema{
		Type: &meta_schema.AnyOfAny17L18NF5UnorderedSetOfAny17L18NF5VWcS9ROiRlIv9QVc{
			Any17L18NF5: (*meta_schema.Any17L18NF5)(&nullT),
		}}

	nullContentDescriptor = meta_schema.ContentDescriptorObject{
		Name:        (*meta_schema.ContentDescriptorObjectName)(&nullS),
		Description: (*meta_schema.ContentDescriptorObjectDescription)(&nullS),
		//Summary:     (*meta_schema.ContentDescriptorObjectSummary)(&nullS),
		Schema:     &nullSchema,
		Required:   (*meta_schema.ContentDescriptorObjectRequired)(&required),
		Deprecated: (*meta_schema.ContentDescriptorObjectDeprecated)(&deprecated),
	}
}

func receiverMethods(methodHandler MethodRegisterer, name string, receiver interface{}) ([]meta_schema.MethodObject, error) {
	ty := reflect.TypeOf(receiver)
	rval := reflect.ValueOf(receiver)

	methods := []meta_schema.MethodObject{}
	for m := 0; m < ty.NumMethod(); m++ {
		method := ty.Method(m)
		if !methodHandler.IsMethodEligible(method) {
			continue
		}

		fdecl, err := getAstFuncDecl(rval, method)
		if err != nil {
			return nil, err
		}

		name, err := methodHandler.GetMethodName(name, rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		description, err := methodHandler.GetMethodDescription(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		summary, err := methodHandler.GetMethodSummary(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		tags, err := methodHandler.GetMethodTags(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		paramsStructure, err := methodHandler.GetMethodParamStructure(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		params := []meta_schema.OneOfContentDescriptorObjectReferenceObjectI0Ye8PrQ{}
		paramCDs, err := methodHandler.GetMethodParams(rval, method, fdecl)
		if err != nil {
			return nil, err
		}
		for i := range paramCDs {
			cp := paramCDs[i]
			params = append(params, meta_schema.OneOfContentDescriptorObjectReferenceObjectI0Ye8PrQ{
				ContentDescriptorObject: &cp,
			})
		}

		resultCD, err := methodHandler.GetMethodResult(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		methodErrors, err := methodHandler.GetMethodErrors(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		links, err := methodHandler.GetMethodLinks(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		examples, err := methodHandler.GetMethodExamples(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		deprecated, err := methodHandler.GetMethodDeprecated(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		exDocs, err := methodHandler.GetMethodExternalDocs(rval, method, fdecl)
		if err != nil {
			return nil, err
		}

		me := meta_schema.MethodObject{
			Name:           (*meta_schema.MethodObjectName)(&name),
			Description:    (*meta_schema.MethodObjectDescription)(&description),
			Summary:        (*meta_schema.MethodObjectSummary)(&summary),
			Tags:           tags,
			ParamStructure: (*meta_schema.MethodObjectParamStructure)(&paramsStructure),
			Params:         (*meta_schema.MethodObjectParams)(&params),
			Result:         &meta_schema.MethodObjectResult{ContentDescriptorObject: &resultCD},
			Errors:         methodErrors,
			Links:          links,
			Examples:       examples,
			Deprecated:     (*meta_schema.MethodObjectDeprecated)(&deprecated),
			ExternalDocs:   exDocs,
		}
		methods = append(methods, me)
	}
	return methods, nil
}

func buildContentDescriptorObject(registerer ContentDescriptorRegisterer, r reflect.Value, m reflect.Method, field *ast.Field, ty reflect.Type) (cd meta_schema.ContentDescriptorObject, err error) {
	name, err := registerer.GetContentDescriptorName(r, m, field)
	if err != nil {
		return cd, err
	}

	description, err := registerer.GetContentDescriptorDescription(r, m, field)
	if err != nil {
		return cd, err
	}

	summary, err := registerer.GetContentDescriptorSummary(r, m, field)
	if err != nil {
		return cd, err
	}

	required, err := registerer.GetContentDescriptorRequired(r, m, field)
	if err != nil {
		return cd, err
	}

	deprecated, err := registerer.GetContentDescriptorDeprecated(r, m, field)
	if err != nil {
		return cd, err
	}

	schema, err := registerer.GetSchema(r, m, field, ty)
	if err != nil {
		return cd, err
	}

	//// If name == description, eg. 'hexutil.Bytes' == 'hexutil.Bytes',
	//// that means the field represent an unnamed variable. Mostly likely an unnamed return value.
	//// Assigning a content descriptor name as a Go type name may be undesirable,
	//// particularly since the 'name' field is used in by-name paramStructure cases to key
	//// the parameter object.
	//// So instead of using this default, let's set the name value to be something
	//// generic given the context of the schema instead.
	////
	//if name == description {
	//	// Field is unnamed.
	//	if schema.Title != nil {
	//		name = (string)(*schema.Title)
	//	} else if schema.Type != nil {
	//		if schema.Type.UnorderedSetOfAny17L18NF5VWcS9ROi != nil {
	//			u := schema.Type.UnorderedSetOfAny17L18NF5VWcS9ROi
	//			uu := *u
	//			a := uu[0]
	//			n, ok := a.(string)
	//			if !ok {
	//				panic("notok1")
	//			}
	//			name = n
	//		} else if schema.Type.Any17L18NF5 != nil {
	//			a := schema.Type.Any17L18NF5
	//			aa := *a
	//			n, ok := aa.(string)
	//			if !ok {
	//				panic("notok2")
	//			}
	//			name = n
	//		}
	//	}
	//}

	cd = meta_schema.ContentDescriptorObject{
		Name:        (*meta_schema.ContentDescriptorObjectName)(&name),
		Description: (*meta_schema.ContentDescriptorObjectDescription)(&description),
		Summary:     (*meta_schema.ContentDescriptorObjectSummary)(&summary),
		Schema:      &schema,
		Required:    (*meta_schema.ContentDescriptorObjectRequired)(&required),
		Deprecated:  (*meta_schema.ContentDescriptorObjectDeprecated)(&deprecated),
	}
	return
}

func buildJSONSchemaObject(registerer SchemaRegisterer, r reflect.Value, m reflect.Method, field *ast.Field, ty reflect.Type) (schema meta_schema.JSONSchema, err error) {
	if !jsonschemaPkgSupport(ty) {
		err = json.Unmarshal([]byte(`{"type": "object", "title": "typeUnsupportedByJSONSchema"}`), &schema)
		return
	}

	rflctr := jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             false,
		IgnoredTypes:               registerer.SchemaIgnoredTypes(),
		TypeMapper:                 registerer.SchemaTypeMap(),
	}

	jsch := rflctr.ReflectFromType(ty)

	// Hacky fix to make library types compatible.
	// Not sure I fully understand the JSON Schema spec here.
	// The libraries like to set "additionalProperties": false,
	// but meta_schema wants it to be a JSONSchema.
	if bytes.Equal(jsch.AdditionalProperties, []byte(`true`)) || bytes.Equal(jsch.AdditionalProperties, []byte(`false`)) {
		jsch.AdditionalProperties = []byte(`{}`)
	}

	// Poor man's glue.
	// Need to get the type from the go struct -> json reflector package
	// to the swagger/go-openapi/jsonschema spec.
	// Do this with JSON marshaling.
	// Hacky? Maybe. Effective? Maybe.
	mm, err := json.Marshal(jsch)
	if err != nil {
		return schema, err
	}

	mm = bytes.Replace(mm, []byte(`"additionalProperties":true`), []byte(`"additionalProperties":{}`), -1)
	mm = bytes.Replace(mm, []byte(`"additionalProperties":false`), []byte(`"additionalProperties":{}`), -1)

	err = json.Unmarshal(mm, &schema)
	if err != nil {
		return schema, fmt.Errorf("unmarshal jsch error: %v\n\n%s", err, string(mm))
	}

	if mutations := registerer.SchemaMutations(ty); len(mutations) > 0 {

		jj := spec.Schema{}
		err = json.Unmarshal(mm, &jj)
		if err != nil {
			return schema, err
		}

		a := go_jsonschema_walk.NewWalker()
		for _, m := range mutations {
			// Initialize the mutation the function.
			// This way, the function is able to be aware of the mutation context,
			// ie establish the root schema context.
			mutFn := m(&jj)
			if err := a.DepthFirst(&jj, mutFn); err != nil {
				return schema, err
			}
		}

		out, err := json.Marshal(jj)
		if err != nil {
			return schema, err
		}

		out = bytes.Replace(out, []byte(`"additionalProperties":true`), []byte(`"additionalProperties":{}`), -1)
		out = bytes.Replace(out, []byte(`"additionalProperties":false`), []byte(`"additionalProperties":{}`), -1)

		schema = meta_schema.JSONSchema{} // Reinitialize
		err = json.Unmarshal(out, &schema)
		if err != nil {
			fmt.Println(string(out))
			return schema, fmt.Errorf("error: %v, schema: %s", err, string(out))
		}
	}

	return schema, nil
}

func isExportedMethod(method reflect.Method) bool {
	return method.PkgPath == ""
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}

func getAstFuncDecl(r reflect.Value, m reflect.Method) (*ast.FuncDecl, error) {
	runtimeFunc := runtime.FuncForPC(m.Func.Pointer())
	runtimeFile, _ := runtimeFunc.FileLine(runtimeFunc.Entry())

	tokenFileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(tokenFileSet, runtimeFile, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	rfName := runtimeFuncBaseName(runtimeFunc)

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

		// Ensure that the receiver name matches.
		reRec := regexp.MustCompile(fnRecName + `\s`)
		if !reRec.MatchString(r.String()) {
			continue
		}
		return fn, nil
	}
	return nil, nil
}

func runtimeFuncBaseName(rf *runtime.Func) string {
	spl := strings.Split(rf.Name(), ".")
	return spl[len(spl)-1]
}

func githubLinkFromValue(receiver reflect.Value, runtimeF *runtime.Func) (*url.URL, error) {
	ty := receiver.Type()
	switch ty.Kind() {
	case reflect.Ptr, reflect.Interface:
		ty = ty.Elem()
	}
	packagePath := ty.PkgPath()

	if !strings.HasPrefix(packagePath, "github.com") {
		return nil, fmt.Errorf("'%s': not a github.com package name", packagePath)
	}

	uris := strings.Split(packagePath, "/") // eg. github.com / ethereum / go-ethereum / internal / ethapi / api.go | [.(*MyRec)... ]
	githubURIOwnerName := strings.Join(uris[:3], "/")
	githubURIRevision := "blob/master"
	pkgRelDir := ""
	pkgRelDir = strings.Join(uris[3:], "/")
	if pkgRelDir != "" {
		// Otherwise we get a double // for files at the module root.
		pkgRelDir = "/" + pkgRelDir
	}

	runtimeFile, runtimeLine := runtimeF.FileLine(runtimeF.Entry())
	base := filepath.Base(runtimeFile)

	ref := fmt.Sprintf("https://%s/%s%s/%s#L%d", githubURIOwnerName, githubURIRevision, pkgRelDir, base, runtimeLine)

	return url.Parse(ref)
}

func expandedFieldNamesFromList(in []*ast.Field) (out []*ast.Field) {
	expandedFields := []*ast.Field{}
	for _, f := range in {
		expandedFields = append(expandedFields, fieldsWithNames(f)...)
	}
	return expandedFields
}

// fieldsWithNames expands a field (either parameter or result, in this case) to
// fields which all have at least one name, or at least one field with one name.
// This handles unnamed fields, and fields declared using multiple names with one type.
// Unnamed fields are assigned a name that is the 'printed' identity of the field Type,
// eg. int -> int, bool -> bool
func fieldsWithNames(f *ast.Field) (fields []*ast.Field) {
	if f == nil {
		return nil
	}

	if len(f.Names) == 0 {
		fields = append(fields, &ast.Field{
			Doc:     f.Doc,
			Names:   []*ast.Ident{{Name: printIdentField(f)}},
			Type:    f.Type,
			Tag:     f.Tag,
			Comment: f.Comment,
		})
		return
	}
	for _, ident := range f.Names {
		fields = append(fields, &ast.Field{
			Doc:     f.Doc,
			Names:   []*ast.Ident{ident},
			Type:    f.Type,
			Tag:     f.Tag,
			Comment: f.Comment,
		})
	}
	return
}

func printIdentField(f *ast.Field) string {
	b := []byte{}
	buf := bytes.NewBuffer(b)
	fs := token.NewFileSet()
	printer.Fprint(buf, fs, f.Type.(ast.Node))
	return buf.String()
}

func jsonschemaPkgSupport(r reflect.Type) bool {
	rr := r
	if rr.Kind() == reflect.Ptr {
		rr = rr.Elem()
	}
	switch rr.Kind() {
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
