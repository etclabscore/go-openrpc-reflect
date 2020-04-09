package go_openrpc_reflect

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"runtime"

	"go/printer"
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

func printIdentField(f *ast.Field) string {
	b := []byte{}
	buf := bytes.NewBuffer(b)
	fs := token.NewFileSet()
	printer.Fprint(buf, fs, f.Type.(ast.Node))
	return buf.String()
}

func expandASTField(f *ast.Field) []*NamedField {
	if f == nil {
		return nil
	}

	out := []*NamedField{}

	if len(f.Names) == 0 {
		out = append(out, &NamedField{
			Name:  printIdentField(f),
			Field: f,
		})
		return out
	}
	for _, ident := range f.Names {
		out = append(out, &NamedField{
			Name:  ident.Name,
			Field: f,
		})
	}
	return out
}
