package go_openrpc_refract

import (
	"go/ast"
	"reflect"
	"unicode"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
	meta_schema "github.com/open-rpc/meta-schema"
)

var EthereumReflector = &EthereumReflectorT{}

type EthereumReflectorT struct{}

func (e EthereumReflectorT) ReceiverMethods(name string, receiver interface{}) ([]meta_schema.MethodObject, error) {
	return receiverMethods(e, name, receiver)
}

// ------------------------------------------------------------------------------

func (e EthereumReflectorT) IsMethodEligible(method reflect.Method) bool {
	// Method must be exported.
	if !isExportedMethod(method) {
		return false
	}

	// All arg types are permitted.
	// If context.Context is the first arg type, it will be skipped.

	// Verify return types. The function must return at most one error
	// and/or one other non-error value.
	outs := make([]reflect.Type, method.Func.Type().NumOut())
	for i := 0; i < method.Func.Type().NumOut(); i++ {
		outs[i] = method.Func.Type().Out(i)
	}
	isErrorType := func(ty reflect.Type) bool {
		return ty == errType
	}

	// If an error is returned, it must be the last returned value.
	switch {
	case len(outs) > 2:
		return false
	case len(outs) == 1 && isErrorType(outs[0]):
		return true
	case len(outs) == 2:
		if isErrorType(outs[0]) || !isErrorType(outs[1]) {
			return false
		}
	}
	return true
}

func firstToLower(str string) string {
	ret := []rune(str)
	if len(ret) > 0 {
		ret[0] = unicode.ToLower(ret[0])
	}
	return string(ret)
}

func (e EthereumReflectorT) GetMethodName(moduleName string, r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (string, error) {
	if moduleName == "" {
		ty := r.Type()
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}
		moduleName = firstToLower(ty.Name())
	}
	return moduleName + "_" + firstToLower(m.Name), nil
}

func (e EthereumReflectorT) GetMethodParams(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) ([]meta_schema.ContentDescriptorObject, error) {
	if astFunc.Type.Params == nil {
		return []meta_schema.ContentDescriptorObject{}, nil
	}

	out := []meta_schema.ContentDescriptorObject{}

	expanded := expandedFieldNamesFromList(astFunc.Type.Params.List)

	for i, field := range expanded {
		ty := m.Type.In(i + 1)
		cd, err := buildContentDescriptorObject(e, r, m, field, ty)
		if err != nil {
			return nil, err
		}
		out = append(out, cd)
	}
	return out, nil
}

func (e EthereumReflectorT) GetMethodResult(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (meta_schema.ContentDescriptorObject, error) {
	if astFunc.Type.Results == nil {
		return nullContentDescriptor, nil
	}

	expandedFields := expandedFieldNamesFromList(astFunc.Type.Results.List)

	if len(expandedFields) == 0 {
		return nullContentDescriptor, nil
	}

	if m.Type.Out(0) == errType {
		return nullContentDescriptor, nil
	}

	return buildContentDescriptorObject(e, r, m, expandedFields[0], m.Type.Out(0))
}

func (e EthereumReflectorT) GetMethodDescription(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (string, error) {
	return StandardReflector.GetMethodDescription(r, m, astFunc)
}

func (e EthereumReflectorT) GetMethodSummary(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (string, error) {
	return StandardReflector.GetMethodSummary(r, m, astFunc)
}

func (e EthereumReflectorT) GetMethodDeprecated(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (bool, error) {
	return StandardReflector.GetMethodDeprecated(r, m, astFunc)
}

func (e EthereumReflectorT) GetMethodExternalDocs(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.ExternalDocumentationObject, error) {
	return StandardReflector.GetMethodExternalDocs(r, m, funcDecl)
}

func (e EthereumReflectorT) GetMethodServers(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.Servers, error) {
	return StandardReflector.GetMethodServers(r, m, funcDecl)
}

func (e EthereumReflectorT) GetMethodLinks(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectLinks, error) {
	return StandardReflector.GetMethodLinks(r, m, funcDecl)
}

func (e EthereumReflectorT) GetMethodExamples(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectExamples, error) {
	return StandardReflector.GetMethodExamples(r, m, funcDecl)
}

func (e EthereumReflectorT) GetMethodTags(r reflect.Value, m reflect.Method, funcDecl *ast.FuncDecl) (*meta_schema.MethodObjectTags, error) {
	return StandardReflector.GetMethodTags(r, m, funcDecl)
}

func (e EthereumReflectorT) GetMethodParamStructure(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (string, error) {
	return StandardReflector.GetMethodParamStructure(r, m, astFunc)
}

func (e EthereumReflectorT) GetMethodErrors(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (*meta_schema.MethodObjectErrors, error) {
	return StandardReflector.GetMethodErrors(r, m, astFunc)
}

// ------------------------------------------------------------------------------

func (e EthereumReflectorT) GetContentDescriptorName(r reflect.Value, m reflect.Method, field *ast.Field) (string, error) {
	return StandardReflector.GetContentDescriptorName(r, m, field)
}

func (e EthereumReflectorT) GetContentDescriptorDescription(r reflect.Value, m reflect.Method, field *ast.Field) (string, error) {
	return StandardReflector.GetContentDescriptorDescription(r, m, field)
}

func (e EthereumReflectorT) GetContentDescriptorSummary(r reflect.Value, m reflect.Method, field *ast.Field) (string, error) {
	return StandardReflector.GetContentDescriptorSummary(r, m, field)
}

func (e EthereumReflectorT) GetContentDescriptorRequired(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error) {
	return StandardReflector.GetContentDescriptorRequired(r, m, field)
}

func (e EthereumReflectorT) GetContentDescriptorDeprecated(r reflect.Value, m reflect.Method, field *ast.Field) (bool, error) {
	return StandardReflector.GetContentDescriptorDeprecated(r, m, field)
}

func (e EthereumReflectorT) GetSchema(r reflect.Value, m reflect.Method, field *ast.Field, ty reflect.Type) (meta_schema.JSONSchema, error) {
	return StandardReflector.GetSchema(r, m, field, ty)
}

// ------------------------------------------------------------------------------

func (e EthereumReflectorT) SchemaIgnoredTypes() []interface{} {
	return StandardReflector.SchemaIgnoredTypes()
}

func (e EthereumReflectorT) SchemaTypeMap() func(ty reflect.Type) *jsonschema.Type {
	return StandardReflector.SchemaTypeMap()
}

func (e EthereumReflectorT) SchemaMutations(ty reflect.Type) []func(*spec.Schema) error {
	return StandardReflector.SchemaMutations(ty)
}
