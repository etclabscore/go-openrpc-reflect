package openrpc_go_document

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

// --- Mock types

type MyService struct {
}

type MyVal struct {
	Name string `jsonschema:"required"`

	//
	Age int `json:"oldness" jsonschema:"required"`

	// CellsN describes how many cells in the body.
	CellN *big.Int `jsonschema:"required"`

	// Exists is a metaphor.
	Exists bool // jsonschema:not_required

	// thoughts are private
	thoughts []string
}

func (m *MyService) Fetch() (result *MyVal, err error) {
	return &MyVal{}, nil
}

func (m *MyService) Do(string) {

}

// Make is a way to make nice things simply.
// More notes. Is Deprecated.
func (m *MyService) Make(ctx context.Context /*name is thing*/, name string /*name is thing*/, cellsN *big.Int) (mustResult MyVal, err error) {
	/*
		Make does things!
	*/
	return MyVal{Name: name, CellN: cellsN}, nil
}

func (m MyService) Produce(name string) error {
	return nil
}

func (m *MyService) callCounter() (n int, err error) {
	return 14, nil
}

func LoneRandomer() (n int, err error) {
	return 42, nil
}

// go/rpc

type GoRPCRecvr struct{}
type GoRPCArg string
type GoRPCRes map[string]interface{}

func (r *GoRPCRecvr) Add(input GoRPCArg, response *GoRPCRes) (err error) {
	return nil
}

func (r *GoRPCRecvr) RandomNum(response *GoRPCRes) error {
	res := map[string]interface{}{
		"answer": 42,
	}
	*response = GoRPCRes(res)
	return nil
}

// ---

func TestCallback(t *testing.T) {
	type NamedField struct {
		Name  string
		Field *ast.Field
	}
	synthesizeMethod := func(f Callback) *goopenrpcT.Method {
		t.Logf("----------------------------------------------------------------")
		t.Logf("")

		t.Logf("REFLECT")
		t.Logf("")
		t.Logf("Receiver:")
		t.Logf(`f.Receiver().String() %s`, f.Rcvr().String())
		t.Logf(`f.Receiver().Type().Name() %s`, f.Rcvr().Type().Name())
		t.Logf(`f.Receiver().Type().String() %s`, f.Rcvr().Type().String())
		t.Logf(`f.Receiver().IsValid() %v`, f.Rcvr().IsValid())
		t.Logf(`f.Receiver().NumMethod() %d`, f.Rcvr().NumMethod())
		t.Logf(`f.Receiver().Kind().String() %s`, f.Rcvr().Kind().String())
		t.Logf(`f.Receiver().IsNil() %v`, f.Rcvr().IsNil())
		t.Logf(`f.Receiver().IsZero() %v`, f.Rcvr().IsZero())
		t.Logf(`f.Receiver().IsValid() %v`, f.Rcvr().IsValid())
		t.Logf("")
		t.Logf("Func:")
		t.Logf(`f.Func().String() %s`, f.Func().String())
		t.Logf(`f.Func().Type().Name() %s`, f.Func().Type().Name())
		t.Logf(`f.Func().Type().String() %s`, f.Func().Type().String())
		t.Logf(`f.Func().IsValid() %v`, f.Func().IsValid())
		t.Logf(`f.Func().NumMethod() %d`, f.Func().NumMethod())
		t.Logf(`f.Func().Kind().String() %s`, f.Func().Kind().String())
		t.Logf(`f.Func().IsNil() %v`, f.Func().IsNil())
		t.Logf(`f.Func().IsZero() %v`, f.Func().IsZero())
		t.Logf(`f.Func().IsValid() %v`, f.Func().IsValid())
		//t.Logf(`f.Func().Len() %d`, f.Func().Len()) // panicsiii

		argTypes := documentGetArgTypes(f.Rcvr(), f.Func())
		for _, a := range argTypes {
			t.Logf(`<-  a.String() %s`, a.String())
		}
		retTyptes := documentGetRetTypes(f.Func())
		for _, r := range retTyptes {
			t.Logf(`  ->  r.String() %s`, r.String())
		}

		t.Logf("")
		t.Logf("RUNTIME")
		t.Logf("")

		runtimeFunc := runtime.FuncForPC(f.Func().Pointer())
		runtimeFile, runtimeLine := runtimeFunc.FileLine(runtimeFunc.Entry())

		t.Logf(`runtimeFunc.Name() %s`, runtimeFunc.Name())
		t.Logf(`runtimeFuncBaseName(runtimeFunc) %s`, runtimeFuncBaseName(runtimeFunc))
		t.Logf(`runtimeFileLine %s:%d`, runtimeFile, runtimeLine)
		t.Logf("")

		tokenFileSet := token.NewFileSet()
		astFile, err := parser.ParseFile(tokenFileSet, runtimeFile, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}

		astFuncDecl := documentGetAstFunc(f.Rcvr(), f.Func(), astFile, runtimeFunc)
		if astFuncDecl == nil {
			t.Error("<< nil-astfuncdecl")
			t.Logf("")
			return nil
		}

		t.Logf("")
		t.Logf("AST")
		t.Logf("")

		t.Logf(`astFuncDecl.Name.String() %s`, astFuncDecl.Name.String())
		t.Logf(`astFuncDecl.Type.Params.NumFields() %d`, astFuncDecl.Type.Params.NumFields())
		t.Logf(`astFuncDecl.Doc.Text() %s`, astFuncDecl.Doc.Text())
		t.Logf("")

		expandASTFieldLog := func(logPre string, f *ast.Field) []*NamedField {
			if f == nil {
				return nil
			}

			out := []*NamedField{}

			fNames := []string{}
			if len(f.Names) > 0 {
				for _, fName := range f.Names {
					if fName == nil {
						panic("nil-fname")
					}
					fNames = append(fNames, fName.Name)
					t.Logf(`%s f.Name.Name %s`, logPre, fName.Name)
					t.Logf(`%s f.Name.String() %s`, logPre, fName.String())
					if fName.Obj != nil {
						t.Logf(`%s f.Name.Obj.Name %s`, logPre, fName.Obj.Name)
					}
				}
			} else {
				defaultFName := fmt.Sprintf(`%s %s`, logPre, f.Type)
				t.Logf(`%s fname default = f.Type %s`, logPre, defaultFName)
				fNames = append(fNames, defaultFName)
			}

			for _, name := range fNames {
				out = append(out, &NamedField{
					Name:  name,
					Field: f,
				})
				t.Logf(`%s f. name %v`, logPre, name)
				t.Logf(`%s f.Type value: %v`, logPre, f.Type)
				t.Logf(`%s f.Type string: %s`, logPre, f.Type)
				t.Logf(`%s f.Doc.Text() %s`, logPre, f.Doc.Text())
				t.Logf(`%s f.Comment.Text() %s`, logPre, f.Comment.Text())
				t.Logf(`%s f.Tag %v`, logPre, f.Tag)
				t.Logf("")
			}
			return out
		}

		expandASTField := func(f *ast.Field) []*NamedField {
			if f == nil {
				return nil
			}

			out := []*NamedField{}

			fNames := []string{}
			if len(f.Names) > 0 {
				for _, fName := range f.Names {
					if fName == nil {
						panic("nil-fname")
					}
					fNames = append(fNames, fName.Name)
					//t.Logf(` f.Name.Name %s`, fName.Name)
					//t.Logf(` f.Name.String() %s`, fName.String())
					if fName.Obj != nil {
						//t.Logf(` f.Name.Obj.Name %s`, fName.Obj.Name)
					}
				}
			} else {
				defaultFName := fmt.Sprintf(`%s`, f.Type)
				//t.Logf(` fname default = f.Type %s`, defaultFName)
				fNames = append(fNames, defaultFName)
			}

			for _, name := range fNames {
				out = append(out, &NamedField{
					Name:  name,
					Field: f,
				})
				//t.Logf(` f. name %v`, name)
				//t.Logf(` f.Type value: %v`, f.Type)
				//t.Logf(` f.Type string: %s`, f.Type)
				//t.Logf(` f.Doc.Text() %s`, f.Doc.Text())
				//t.Logf(` f.Comment.Text() %s`, f.Comment.Text())
				//t.Logf(` f.Tag %v`, f.Tag)
				//t.Logf("")
			}
			return out
		}

		argASTFields := []*NamedField{}
		if astFuncDecl.Type != nil &&
			astFuncDecl.Type.Params != nil &&
			astFuncDecl.Type.Params.List != nil {
			for _, f := range astFuncDecl.Type.Params.List {
				expandASTFieldLog("<- ", f)
				argASTFields = append(argASTFields, expandASTField(f)...)
			}
		}

		retASTFields := []*NamedField{}
		if astFuncDecl.Type != nil &&
			astFuncDecl.Type.Results != nil &&
			astFuncDecl.Type.Results.List != nil {
			for _, f := range astFuncDecl.Type.Results.List {
				expandASTFieldLog("  -> ", f)
				retASTFields = append(retASTFields, expandASTField(f)...)
			}
		}

		t.Logf("METHOD")
		t.Logf("")

		summary := func() string {
			if astFuncDecl.Doc != nil {
				return astFuncDecl.Doc.Text()
			}
			return ""
		}

		description := func() string {
			return fmt.Sprintf("`%s`", f.Func().Type().String())
		}

		deprecated := func() bool {
			matched, err := regexp.MatchString(`(?im)deprecated`, summary())
			if err != nil {
				t.Fatal(err)
			}
			return matched
		}

		fullDescriptionOfType := func(ty reflect.Type) string {
			if ty.PkgPath() != "" {
				return fmt.Sprintf(`%s.%s`, ty.PkgPath(), ty.Name())
			}
			return ty.String()
		}

		schema := func(ty reflect.Type) spec.Schema {

			if !jsonschemaPkgSupport(ty) {
				t.Fatal("TODO")
			}

			rflctr := jsonschema.Reflector{
				AllowAdditionalProperties:  false, // false,
				RequiredFromJSONSchemaTags: true,
				ExpandedStruct:             true, // false, // false,
			}
			jsch := rflctr.ReflectFromType(ty)

			// Poor man's glue.
			// Need to get the type from the go struct -> json reflector package
			// to the swagger/go-openapi/jsonschema spec.
			// Do this with JSON marshaling.
			// Hacky? Maybe. Effective? Maybe.
			m, err := json.Marshal(jsch)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf(`%s`, string(m))
			sch := spec.Schema{}
			err = json.Unmarshal(m, &sch)
			if err != nil {
				t.Fatal(err)
			}

			// NOTE: Debug toggling.
			expand := true
			if expand {
				err = spec.ExpandSchema(&sch, &sch, nil)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Again, this should be pluggable.
			handleNoDefinitionsDefault := true
			if handleNoDefinitionsDefault {
				sch.Definitions = nil
			}

			handleRequireDefault := true
			if handleRequireDefault {
				// If we didn't explicitly set any fields as required with jsonschema tags,
				// then we can assume the default, that ALL properties are required.
				if len(sch.Required) == 0 {
					for k := range sch.Properties {
						sch.Required = append(sch.Required, k)
					}
				}
			}

			handleDescriptionDefault := true
			if handleDescriptionDefault {
				if sch.Description == "" {
					sch.Description = fullDescriptionOfType(ty)
				}
			}

			return sch
		}

		contentDescriptor := func(ty reflect.Type, astNamedField *NamedField) *goopenrpcT.ContentDescriptor {
			astNamedField.Field.Pos()

			return &goopenrpcT.ContentDescriptor{
				Content: goopenrpcT.Content{
					Name:        astNamedField.Name,
					Summary:     astNamedField.Field.Comment.Text(),
					Required:    true,
					Description: "mydescription", // fullDescriptionOfType(ty),
					Schema:      schema(ty),
				},
			}
		}

		defaultContentDescriptorSkip := func(isArgs bool, cd *goopenrpcT.ContentDescriptor) bool {
			if isArgs {
				if cd.Schema.Description == "context.Context" {
					return true
				}
			}
			return false
		}

		params := func(skipFn func(isArgs bool, descriptor *goopenrpcT.ContentDescriptor) bool) []*goopenrpcT.ContentDescriptor {
			out := []*goopenrpcT.ContentDescriptor{}
			for i, a := range argTypes {
				cd := contentDescriptor(a, argASTFields[i])
				if skipFn(true, cd) {
					continue
				}
				out = append(out, cd)
			}
			return out
		}

		nullContentDescriptor := &goopenrpcT.ContentDescriptor{
			Content: goopenrpcT.Content{
				Name: "Null",
				Schema: spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{"Null"},
					},
				},
			},
		}
		rets := func(skipFn func(isArgs bool, descriptor *goopenrpcT.ContentDescriptor) bool) []*goopenrpcT.ContentDescriptor {
			out := []*goopenrpcT.ContentDescriptor{}
			for i, r := range retTyptes {
				cd := contentDescriptor(r, retASTFields[i])
				if skipFn(false, cd) {
					continue
				}
				out = append(out, cd)
			}
			if len(out) == 0 {
				out = append(out, nullContentDescriptor)
			}
			return out
		}

		method := goopenrpcT.Method{
			Name:        runtimeFunc.Name(), // FIXME or give me a comment.
			Tags:        nil,
			Summary:     summary(),
			Description: description(),
			ExternalDocs: goopenrpcT.ExternalDocs{
				Description: fmt.Sprintf("%d", runtimeLine),
				URL:         fmt.Sprintf("file://%s", runtimeFile), // TODO: Provide WORKING external docs links to Github (actually a wrapper/injection to make this configurable).
			},
			Params:         params(defaultContentDescriptorSkip),
			Result:         rets(defaultContentDescriptorSkip)[0],
			Deprecated:     deprecated(),
			Servers:        nil,
			Errors:         nil,
			Links:          nil,
			ParamStructure: "",
			Examples:       nil,
		}
		b, _ := json.MarshalIndent(method, "", "  ")
		t.Logf(string(b))

		//spew.Config.DisableMethods = true
		//t.Logf(spew.Sdump(astFuncDecl))

		return &method
	}

	serv := new(MyService)

	/* Don't do it this way. Bad. */
	//a := Callback{
	//	Receiver: reflect.ValueOf(serv),
	//	Fn:       reflect.ValueOf(serv.Make),
	//}
	//
	//synthesizeMethod(a)

	/* Do it this way. Good. */
	servVal := reflect.ValueOf(serv)
	fnVal, ok := reflect.TypeOf(serv).MethodByName("Make")
	if !ok {
		t.Fatal("notok")
	}

	b := Callback{
		Receiver: servVal,
		Fn:       fnVal.Func,
	}
	synthesizeMethod(b)

	//fnVal, ok = reflect.TypeOf(serv).MethodByName("Do")
	//if !ok {
	//	t.Fatal("notok")
	//}
	//c := Callback{
	//	Receiver: servVal,
	//	Fn:       fnVal.Func,
	//}
	//synthesizeMethod(c)

}
func TestDiscover1(t *testing.T) {
	//
	//myserv := new(MyService)
	////goserv := new(GoRPCRecvr)
	//
	//
	//name := "serv_example"

}
