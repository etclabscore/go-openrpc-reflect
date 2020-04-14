package go_openrpc_reflect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"net"
	"reflect"
	"regexp"
	"runtime"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
	meta_schema "github.com/open-rpc/meta-schema"
)

// ReceiverReflectorT holds a field for each ReceiverRegisterer interface method.
// StandardReflectorT embeds this struct, together with self.FnMethod != nil checks, as a way to easily override
// defaults from the consuming application side. See example4_test.go for an an example.
type ReceiverReflectorT struct {
	FnReceiverMethods func (name string, receiver interface{}) ([]meta_schema.MethodObject, error)
	FnIsMethodEligible func(method reflect.Method) bool
	FnGetMethodName func(moduleName string, r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	FnGetMethodTags func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectTags, error)
	FnGetMethodDescription func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	FnGetMethodSummary func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	FnGetMethodDeprecated func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (bool, error)
	FnGetMethodParamStructure func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error)
	FnGetMethodErrors func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectErrors, error)
	FnGetMethodExternalDocs func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.ExternalDocumentationObject, error)
	FnGetMethodServers func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.Servers, error)
	FnGetMethodLinks func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectLinks, error)
	FnGetMethodExamples func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectExamples, error)
	FnGetMethodParams func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) ([]meta_schema.ContentDescriptorObject, error)
	FnGetMethodResult func(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (meta_schema.ContentDescriptorObject, error)
	FnGetContentDescriptorName func(r reflect.Value, m reflect.Method, field *ast.Field) (string, error)
	FnGetContentDescriptorSummary func(r reflect.Value, m reflect.Method, field *ast.Field) (string, error)
	FnGetContentDescriptorDescription func(r reflect.Value, m reflect.Method, field *ast.Field) (string, error)
	FnGetContentDescriptorRequired func(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error)
	FnGetContentDescriptorDeprecated func(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error)
	FnGetSchema func(r reflect.Value, m reflect.Method, field *ast.Field, ty reflect.Type) (schema meta_schema.JSONSchema, err error)
	FnSchemaIgnoredTypes func () []interface{}
	FnSchemaTypeMap func () func(ty reflect.Type) *jsonschema.Type
	FnSchemaMutations func (ty reflect.Type) []func(*spec.Schema) error
}

type StandardReflectorT struct{
	ReceiverReflectorT
}

var StandardReflector = &StandardReflectorT{}

func (c *StandardReflectorT) GetServers() func (listeners []net.Listener) (*meta_schema.Servers, error) {
	return func (listeners []net.Listener) (*meta_schema.Servers, error) {
		if listeners == nil {
			return nil, nil
		}
		if len(listeners) == 0 {
			return nil, nil
		}
		servers := []meta_schema.ServerObject{}
		for _, listener := range listeners {
			if listener == nil {
				continue
			}
			addr := listener.Addr().String()
			network := listener.Addr().Network()
			servers = append(servers, meta_schema.ServerObject{
				Url:  (*meta_schema.ServerObjectUrl)(&addr),
				Name: (*meta_schema.ServerObjectName)(&network),
			})
		}
		return (*meta_schema.Servers)(&servers), nil
	}
}

func (c *StandardReflectorT) ReceiverMethods(name string, receiver interface{}) ([]meta_schema.MethodObject, error) {
	if c.FnReceiverMethods != nil {
		return c.FnReceiverMethods(name, receiver)
	}
	return receiverMethods(c, name, receiver)
}

// ------------------------------------------------------------------------------

func (c *StandardReflectorT) IsMethodEligible(method reflect.Method) bool {
	if c.FnIsMethodEligible != nil {
		return c.FnIsMethodEligible(method)
	}
	// Method must be exported.
	if !isExportedMethod(method) {
		return false
	}

	mtype := method.Type

	// Method needs three ins: receiver, *args, *reply.
	if mtype.NumIn() != 3 {
		return false
	}
	// First arg need not be a pointer.
	argType := mtype.In(1)
	if !isExportedOrBuiltinType(argType) {

		return false
	}
	// Second arg must be a pointer.
	replyType := mtype.In(2)
	if replyType.Kind() != reflect.Ptr {
		return false
	}
	// Reply type must be exported.
	if !isExportedOrBuiltinType(replyType) {
		return false
	}
	// Method needs one out.
	if mtype.NumOut() != 1 {
		return false
	}
	// The return type of the method must be error.
	if returnType := mtype.Out(0); returnType != errType {
		return false
	}
	return true
}

func (c *StandardReflectorT) GetMethodName(moduleName string, r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error) {
	if c.FnGetMethodName != nil {
		return c.FnGetMethodName(moduleName, r, m, funcDecl)
	}
	if moduleName == "" {
		ty := r.Type()
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}
		moduleName = ty.Name()
	}
	return moduleName + "." + m.Name, nil
}

func (c *StandardReflectorT) GetMethodParams(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) ([]meta_schema.ContentDescriptorObject, error) {
	if c.FnGetMethodParams != nil {
		return c.FnGetMethodParams(r, m, funcDecl)
	}
	// A case where expanded fields arg expression would fail (if anyof `funcDecl.Type.Params` == nil)
	// should be caught by the IsMethodEligible condition.
	if funcDecl.Type.Params == nil {
		panic("unreachable")
	}

	expandedFields := expandedFieldNamesFromList(funcDecl.Type.Params.List)

	// We always want only the first param.
	nf := expandedFields[0]
	ty := m.Type.In(1)
	cd, err := buildContentDescriptorObject(c, r, m, nf, ty)
	if err != nil {
		return nil, err
	}

	// Spec says params are always a list.
	return []meta_schema.ContentDescriptorObject{cd}, nil
}

func (c *StandardReflectorT) GetMethodResult(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (cd meta_schema.ContentDescriptorObject, err error) {
	if c.FnGetMethodResult != nil {
		return c.FnGetMethodResult(r, m, funcDecl)
	}
	if funcDecl.Type.Params == nil {
		panic("unreachable")
	}

	expandedFields := expandedFieldNamesFromList(funcDecl.Type.Params.List)

	// We always want only the second param.
	nf := expandedFields[1]
	ty := m.Type.In(2)
	return buildContentDescriptorObject(c, r, m, nf, ty)
}

func (c *StandardReflectorT) GetMethodDescription(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error) {
	if c.FnGetMethodDescription != nil {
		return c.FnGetMethodDescription(r, m, funcDecl)
	}
	tokenFileSet := token.NewFileSet()

	printed := []byte{}
	buf := bytes.NewBuffer(printed)
	err := printer.Fprint(buf, tokenFileSet, funcDecl)
	if err != nil {
		return "", err
	}
	printed = buf.Bytes()

	return fmt.Sprintf("```go\n%s\n```", string(printed)), nil
}

func (c *StandardReflectorT) GetMethodSummary(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error) {
	if c.FnGetMethodSummary != nil {
		return c.FnGetMethodSummary(r, m, funcDecl)
	}
	if funcDecl.Doc != nil {
		return funcDecl.Doc.Text(), nil
	}
	return "", nil
}

func (c *StandardReflectorT) GetMethodDeprecated(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (bool, error) {
	if c.FnGetMethodDeprecated != nil {
		return c.FnGetMethodDeprecated(r, m, funcDecl)
	}
	var comment string
	if funcDecl.Doc != nil {
		comment = funcDecl.Doc.Text()
	}
	if comment == "" {
		return false, nil
	}
	matched, _ := regexp.MatchString(`(?im)deprecated`, funcDecl.Doc.Text())
	return matched, nil
}

func (c *StandardReflectorT) GetMethodExternalDocs(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.ExternalDocumentationObject, error) {
	if c.FnGetMethodExternalDocs != nil {
		return c.FnGetMethodExternalDocs(r, m, funcDecl)
	}
	// NOTE: This will NOT work for forks. Hm.

	// If we can assemble a github.com/ url for the method, then
	// return that prefixed before the printed code.
	runtimeFunc := runtime.FuncForPC(m.Func.Pointer())

	githubURL, err := githubLinkFromValue(r, runtimeFunc)
	if err == nil {
		description := "Github remote link"
		u := githubURL.String()
		return &meta_schema.ExternalDocumentationObject{
			Description: (*meta_schema.ExternalDocumentationObjectDescription)(&description),
			Url:         (*meta_schema.ExternalDocumentationObjectUrl)(&u),
		}, nil
	}

	return nil, nil
}

/*
TODO: These.
*/
func (c *StandardReflectorT) GetMethodTags(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectTags, error) {
	if c.FnGetMethodTags != nil {
		return c.FnGetMethodTags(r, m, funcDecl)
	}
	return nil, nil
}

func (c *StandardReflectorT) GetMethodParamStructure(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (string, error) {
	if c.FnGetMethodParamStructure != nil {
		return c.FnGetMethodParamStructure(r, m, funcDecl)
	}
	return "by-position", nil
}

func (c *StandardReflectorT) GetMethodErrors(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectErrors, error) {
	if c.FnGetMethodErrors != nil {
		return c.FnGetMethodErrors(r, m, funcDecl)
	}
	return nil, nil
}

func (c *StandardReflectorT) GetMethodServers(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.Servers, error) {
	if c.FnGetMethodServers != nil {
		return c.FnGetMethodServers(r, m, funcDecl)
	}
	return nil, nil
}

func (c *StandardReflectorT) GetMethodLinks(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectLinks, error) {
	if c.FnGetMethodLinks != nil {
		return c.FnGetMethodLinks(r, m, funcDecl)
	}
	return nil, nil
}

func (c *StandardReflectorT) GetMethodExamples(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectExamples, error) {
	if c.FnGetMethodExamples != nil {
		return c.FnGetMethodExamples(r, m, funcDecl)
	}
	return nil, nil
}

// ------------------------------------------------------------------------------

func (c *StandardReflectorT) GetContentDescriptorName(r reflect.Value, m reflect.Method, field *ast.Field) (string, error) {
	if c.FnGetContentDescriptorName != nil {
		return c.FnGetContentDescriptorName(r, m, field)
	}
	fs := expandedFieldNamesFromList([]*ast.Field{field})
	return fs[0].Names[0].Name, nil
}

func (c *StandardReflectorT) GetContentDescriptorDescription(r reflect.Value, m reflect.Method, field *ast.Field) (string, error) {
	if c.FnGetContentDescriptorDescription != nil {
		return c.FnGetContentDescriptorDescription(r, m, field)
	}
	return printIdentField(field), nil
}

func (c *StandardReflectorT) GetContentDescriptorSummary(r reflect.Value, m reflect.Method, field *ast.Field) (string, error) {
	if c.FnGetContentDescriptorSummary != nil {
		return c.FnGetContentDescriptorSummary(r, m, field)
	}
	summary := field.Comment.Text()
	if summary == "" {
		summary = field.Doc.Text()
	}
	return summary, nil
}

func (c *StandardReflectorT) GetContentDescriptorRequired(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error) {
	if c.FnGetContentDescriptorRequired != nil {
		return c.FnGetContentDescriptorRequired(r, m, field)
	}
	// The standard method signature pattern does not allow for variadic arguments.
	return true, nil
}

func (c *StandardReflectorT) GetContentDescriptorDeprecated(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error) {
	if c.FnGetContentDescriptorDeprecated != nil {
		return c.FnGetContentDescriptorDeprecated(r, m, field)
	}
	var comment string
	if field.Doc != nil {
		comment = field.Doc.Text()
	}
	if comment == "" && field.Comment != nil {
		comment = field.Comment.Text()
	}
	if comment == "" {
		return false, nil
	}
	matched, _ := regexp.MatchString(`(?im)deprecated`, comment)
	return matched, nil
}

func (c *StandardReflectorT) GetSchema(r reflect.Value, m reflect.Method, field *ast.Field, ty reflect.Type) (meta_schema.JSONSchema, error) {
	if c.FnGetSchema != nil {
		return c.FnGetSchema(r, m, field, ty)
	}
	return buildJSONSchemaObject(c, r, m, field, ty)
}

// ------------------------------------------------------------------------------

func (c *StandardReflectorT) SchemaIgnoredTypes() []interface{} {
	if c.FnSchemaIgnoredTypes != nil {
		return c.FnSchemaIgnoredTypes()
	}
	return nil
}

func (c *StandardReflectorT) SchemaTypeMap() func(ty reflect.Type) *jsonschema.Type {
	if c.FnSchemaTypeMap != nil {
		return c.FnSchemaTypeMap()
	}
	return nil
}

func (c *StandardReflectorT) SchemaMutations(ty reflect.Type) []func(*spec.Schema) error {
	if c.FnSchemaMutations != nil {
		return c.FnSchemaMutations(ty)
	}
	return []func(*spec.Schema) error{
		SchemaMutationRequireDefaultOn,
		SchemaMutationExpand,
		SchemaMutationRemoveDefinitionsField,
	}
}

// ------------------------------------------------------------------------------

func SchemaMutationRemoveDefinitionsField(s *spec.Schema) error {
	s.Definitions = nil
	s.Ref = spec.Ref{}
	return nil
}

func SchemaMutationExpand(s *spec.Schema) error {
	err := spec.ExpandSchema(s, s, nil)
	if err != nil {
		b, _ := json.MarshalIndent(s, "", "  ")
		return fmt.Errorf("schema Expand mutation error: %v schema:\n%s", err, string(b))
	}
	return nil
}

func SchemaMutationRequireDefaultOn(s *spec.Schema) error {
	// If we didn't explicitly set any fields as required with jsonschema tags,
	// then we can assume the default, that ALL properties are required.
	if len(s.Required) == 0 {
		for k := range s.Properties {
			s.Required = append(s.Required, k)
		}
	}
	return nil
}