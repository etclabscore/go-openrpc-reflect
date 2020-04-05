package openrpc_go_document

import (
	"go/ast"
	"reflect"
	"runtime"
	"strings"
)


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

func runtimeFuncBaseName(rf *runtime.Func) string {
	spl := strings.Split(rf.Name(), ".")
	return spl[len(spl)-1]
}
