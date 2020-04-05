package openrpc_go_document

import (
	"fmt"
	"go/ast"
	"regexp"
	"runtime"
)

type NamedField struct {
	Name  string
	Field *ast.Field
}

func documentGetAstFunc(f Callback, astFile *ast.File, rf *runtime.Func) *ast.FuncDecl {
	rfName := runtimeFuncBaseName(rf)

	for _, decl := range astFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name == nil || fn.Name.Name != rfName {
			continue
		}

		if !f.HasReceiver() {
			return fn
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
		if !reRec.MatchString(f.Rcvr().String()) {
			continue
		}
		return fn
	}
	return nil
}

func expandASTField(f *ast.Field) []*NamedField {
	if f == nil {
		return nil
	}

	out := []*NamedField{}

	/*
		Names can be like from the following

		func add(a, b int, base uint)
		func add(a int, b int, base uint)
		func add(int, int, uint)

		So we need to collect them all for each field (eg int), with default names
		in case they're unnamed.

		In case a field has multiple names, we need to expand
		the returns to include all iterations.
	*/

	// If the field is unnamed, then set from the type.
	defaultFName := fmt.Sprintf(`%s`, f.Type)

	fNames := []string{}
	if len(f.Names) > 0 {
		for _, fName := range f.Names {
			if fName == nil {
				panic("nil-fname")
			}
			fNames = append(fNames, fName.Name)
			if fName.Obj != nil {
			}
		}
	} else {
		fNames = append(fNames, defaultFName)
	}

	for _, name := range fNames {
		out = append(out, &NamedField{
			Name:  name,
			Field: f,
		})
	}
	return out
}
