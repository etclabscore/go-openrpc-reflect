package go_openrpc_reflect

import (
	"context"
	"go/ast"
	"reflect"
	"unicode"

	meta_schema "github.com/open-rpc/meta-schema"
)

type EthereumReflectorT struct {
	StandardReflectorT
}

var EthereumReflector = &EthereumReflectorT{}

func (e *EthereumReflectorT) ReceiverMethods(name string, receiver interface{}) ([]meta_schema.MethodObject, error) {
	if e.FnReceiverMethods != nil {
		return e.FnReceiverMethods(name, receiver)
	}
	return receiverMethods(e, name, receiver)
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

// ------------------------------------------------------------------------------

func (e *EthereumReflectorT) IsMethodEligible(method reflect.Method) bool {
	if e.FnIsMethodEligible != nil {
		return e.FnIsMethodEligible(method)
	}
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

func (e *EthereumReflectorT) GetMethodName(moduleName string, r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (string, error) {
	if e.FnGetMethodName != nil {
		return e.FnGetMethodName(moduleName, r, m, astFunc)
	}
	if moduleName == "" {
		ty := r.Type()
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}
		moduleName = firstToLower(ty.Name())
	}
	return moduleName + "_" + firstToLower(m.Name), nil
}

func (e *EthereumReflectorT) GetMethodParams(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) ([]meta_schema.ContentDescriptorObject, error) {
	if e.FnGetMethodParams != nil {
		return e.FnGetMethodParams(r, m, astFunc)
	}
	if astFunc.Type.Params == nil {
		return []meta_schema.ContentDescriptorObject{}, nil
	}

	out := []meta_schema.ContentDescriptorObject{}

	expanded := expandedFieldNamesFromList(astFunc.Type.Params.List)

	for i, field := range expanded {
		ty := m.Type.In(i + 1)

		// go-ethereum/rpc skips the first parameter if it is context.Context,
		// which is used for subscriptions.
		if i+1 == 1 && ty == contextType {
			continue
		}
		cd, err := buildContentDescriptorObject(e, r, m, field, ty)
		if err != nil {
			return nil, err
		}
		out = append(out, cd)
	}
	return out, nil
}

func (e *EthereumReflectorT) GetMethodResult(r reflect.Value, m reflect.Method, astFunc *ast.FuncDecl) (meta_schema.ContentDescriptorObject, error) {
	if e.FnGetMethodResult != nil {
		return e.FnGetMethodResult(r, m, astFunc)
	}
	if astFunc.Type.Results == nil {
		return nullContentDescriptor, nil
	}

	expandedFields := expandedFieldNamesFromList(astFunc.Type.Results.List)

	if len(expandedFields) == 0 {
		return nullContentDescriptor, nil
	}

	if m.Type.NumOut() == 0 || m.Type.Out(0) == errType {
		return nullContentDescriptor, nil
	}

	return buildContentDescriptorObject(e, r, m, expandedFields[0], m.Type.Out(0))
}
