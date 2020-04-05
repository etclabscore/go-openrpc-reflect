package openrpc_go_document

import (
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/gregdhill/go-openrpc/types"
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
func (m *MyService) Make(ctx context.Context /*name is thing*/, name string /*name is thing*/, cellsN *big.Int) (myResult MyVal, err error) {
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

func LoneRandomer(name string) (n int, err error) {
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

func TestCallback_HasReceiver(t *testing.T) {
	cb := Callback{NoReceiverValue, reflect.ValueOf(LoneRandomer)}
	if cb.HasReceiver() {
		t.Fatal("bad")
	}
}

func TestDocument_Discover(t *testing.T) {
	serv := new(MyService)
	callbacks := GoRPCServiceMethods(serv)()
	callbacks["custom_method"] = Callback{NoReceiverValue, reflect.ValueOf(LoneRandomer)}
	callbacksFn := func() map[string]Callback {
		return callbacks
	}
	doc := DocumentProvider(&ServerProvider{
		Callbacks: callbacksFn,
		OpenRPCInfo: func() types.Info {
			return types.Info{}
		},
		OpenRPCExternalDocs: func() types.ExternalDocs {
			return types.ExternalDocs{}
		},
	}, &DocumentProviderParseOpts{
		SchemaMutationFns: []func(s *spec.Schema) error{
			SchemaMutationRequireDefaultOn,
			SchemaMutationExpand,
			SchemaMutationRemoveDefinitionsField,
		},
		ContentDescriptorMutationFns: nil,
		MethodBlackList:              nil,
		TypeMapper:                   nil,
		SchemaIgnoredTypes:           nil,
		ContentDescriptorSkipFn:      nil,
	})

	err := doc.Discover()
	if err != nil {
		t.Fatal(err)
	}

	doc.FlattenSchemas()

	definition := doc.Spec1()
	b, _ := json.MarshalIndent(definition, "", "  ")
	t.Logf(string(b))
}

// ---

//func TestCallback(t *testing.T) {
//	synthesizeMethod := func(f Callback) *goopenrpcT.Method {
//		t.Logf("----------------------------------------------------------------")
//		t.Logf("")
//
//		t.Logf("REFLECT")
//		t.Logf("")
//		t.Logf("Receiver:")
//		t.Logf(`f.Receiver().String() %s`, f.Rcvr().String())
//		t.Logf(`f.Receiver().Type().Name() %s`, f.Rcvr().Type().Name())
//		t.Logf(`f.Receiver().Type().String() %s`, f.Rcvr().Type().String())
//		t.Logf(`f.Receiver().IsValid() %v`, f.Rcvr().IsValid())
//		t.Logf(`f.Receiver().NumMethod() %d`, f.Rcvr().NumMethod())
//		t.Logf(`f.Receiver().Kind().String() %s`, f.Rcvr().Kind().String())
//		t.Logf(`f.Receiver().IsNil() %v`, f.Rcvr().IsNil())
//		t.Logf(`f.Receiver().IsZero() %v`, f.Rcvr().IsZero())
//		t.Logf(`f.Receiver().IsValid() %v`, f.Rcvr().IsValid())
//		t.Logf("")
//		t.Logf("Func:")
//		t.Logf(`f.Func().String() %s`, f.Func().String())
//		t.Logf(`f.Func().Type().Name() %s`, f.Func().Type().Name())
//		t.Logf(`f.Func().Type().String() %s`, f.Func().Type().String())
//		t.Logf(`f.Func().IsValid() %v`, f.Func().IsValid())
//		t.Logf(`f.Func().NumMethod() %d`, f.Func().NumMethod())
//		t.Logf(`f.Func().Kind().String() %s`, f.Func().Kind().String())
//		t.Logf(`f.Func().IsNil() %v`, f.Func().IsNil())
//		t.Logf(`f.Func().IsZero() %v`, f.Func().IsZero())
//		t.Logf(`f.Func().IsValid() %v`, f.Func().IsValid())
//		//t.Logf(`f.Func().Len() %d`, f.Func().Len()) // panicsiii
//
//		argTypes := f.getArgTypes()
//		for _, a := range argTypes {
//			t.Logf(`<-  a.String() %s`, a.String())
//		}
//		retTyptes := f.getRetTypes()
//		for _, r := range retTyptes {
//			t.Logf(`  ->  r.String() %s`, r.String())
//		}
//
//		t.Logf("")
//		t.Logf("RUNTIME")
//		t.Logf("")
//
//		runtimeFunc := runtime.FuncForPC(f.Func().Pointer())
//		runtimeFile, runtimeLine := runtimeFunc.FileLine(runtimeFunc.Entry())
//
//		t.Logf(`runtimeFunc.Name() %s`, runtimeFunc.Name())
//		t.Logf(`runtimeFuncBaseName(runtimeFunc) %s`, runtimeFuncBaseName(runtimeFunc))
//		t.Logf(`runtimeFileLine %s:%d`, runtimeFile, runtimeLine)
//		t.Logf("")
//
//		tokenFileSet := token.NewFileSet()
//		astFile, err := parser.ParseFile(tokenFileSet, runtimeFile, nil, parser.ParseComments)
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		astFuncDecl := documentGetAstFunc(f, astFile, runtimeFunc)
//		if astFuncDecl == nil {
//			t.Error("<< nil-astfuncdecl")
//			t.Logf("")
//			return nil
//		}
//
//		t.Logf("")
//		t.Logf("AST")
//		t.Logf("")
//
//		t.Logf(`astFuncDecl.Name.String() %s`, astFuncDecl.Name.String())
//		t.Logf(`astFuncDecl.Type.Params.NumFields() %d`, astFuncDecl.Type.Params.NumFields())
//		t.Logf(`astFuncDecl.Doc.Text() %s`, astFuncDecl.Doc.Text())
//		t.Logf("")
//
//		expandASTFieldLog := func(logPre string, f *ast.Field) []*NamedField {
//			if f == nil {
//				return nil
//			}
//
//			out := []*NamedField{}
//
//			fNames := []string{}
//			if len(f.Names) > 0 {
//				for _, fName := range f.Names {
//					if fName == nil {
//						panic("nil-fname")
//					}
//					fNames = append(fNames, fName.Name)
//					t.Logf(`%s f.Name.Name %s`, logPre, fName.Name)
//					t.Logf(`%s f.Name.String() %s`, logPre, fName.String())
//					if fName.Obj != nil {
//						t.Logf(`%s f.Name.Obj.Name %s`, logPre, fName.Obj.Name)
//					}
//				}
//			} else {
//				defaultFName := fmt.Sprintf(`%s %s`, logPre, f.Type)
//				t.Logf(`%s fname default = f.Type %s`, logPre, defaultFName)
//				fNames = append(fNames, defaultFName)
//			}
//
//			for _, name := range fNames {
//				out = append(out, &NamedField{
//					Name:  name,
//					Field: f,
//				})
//				t.Logf(`%s f. name %v`, logPre, name)
//				t.Logf(`%s f.Type value: %v`, logPre, f.Type)
//				t.Logf(`%s f.Type string: %s`, logPre, f.Type)
//				t.Logf(`%s f.Doc.Text() %s`, logPre, f.Doc.Text())
//				t.Logf(`%s f.Comment.Text() %s`, logPre, f.Comment.Text())
//				t.Logf(`%s f.Tag %v`, logPre, f.Tag)
//				t.Logf("")
//			}
//			return out
//		}
//
//		argASTFields := []*NamedField{}
//		if astFuncDecl.Type != nil &&
//			astFuncDecl.Type.Params != nil &&
//			astFuncDecl.Type.Params.List != nil {
//			for _, f := range astFuncDecl.Type.Params.List {
//				expandASTFieldLog("<- ", f)
//				argASTFields = append(argASTFields, expandASTField(f)...)
//			}
//		}
//
//		retASTFields := []*NamedField{}
//		if astFuncDecl.Type != nil &&
//			astFuncDecl.Type.Results != nil &&
//			astFuncDecl.Type.Results.List != nil {
//			for _, f := range astFuncDecl.Type.Results.List {
//				expandASTFieldLog("  -> ", f)
//				retASTFields = append(retASTFields, expandASTField(f)...)
//			}
//		}
//
//		t.Logf("METHOD")
//		t.Logf("")
//
//
//		description := func() string {
//			return fmt.Sprintf("`%s`", f.Func().Type().String())
//		}
//
//
//		contentDescriptor := func(ty reflect.Type, astNamedField *NamedField) *goopenrpcT.ContentDescriptor {
//			astNamedField.Field.Pos()
//
//			return &goopenrpcT.ContentDescriptor{
//				Content: goopenrpcT.Content{
//					Name:        astNamedField.Name,
//					Summary:     astNamedField.Field.Comment.Text(),
//					Required:    true,
//					Description: "mydescription", // fullDescriptionOfType(ty),
//					Schema:      typeToSchema(ty),
//				},
//			}
//		}
//
//		params := func(skipFn func(isArgs bool, descriptor *goopenrpcT.ContentDescriptor) bool) []*goopenrpcT.ContentDescriptor {
//			out := []*goopenrpcT.ContentDescriptor{}
//			for i, a := range argTypes {
//				cd := contentDescriptor(a, argASTFields[i])
//				if skipFn(true, cd) {
//					continue
//				}
//				out = append(out, cd)
//			}
//			return out
//		}
//
//		nullContentDescriptor := &goopenrpcT.ContentDescriptor{
//			Content: goopenrpcT.Content{
//				Name: "Null",
//				Schema: spec.Schema{
//					SchemaProps: spec.SchemaProps{
//						Type: []string{"Null"},
//					},
//				},
//			},
//		}
//		rets := func(skipFn func(isArgs bool, descriptor *goopenrpcT.ContentDescriptor) bool) []*goopenrpcT.ContentDescriptor {
//			out := []*goopenrpcT.ContentDescriptor{}
//			for i, r := range retTyptes {
//				cd := contentDescriptor(r, retASTFields[i])
//				if skipFn(false, cd) {
//					continue
//				}
//				out = append(out, cd)
//			}
//			if len(out) == 0 {
//				out = append(out, nullContentDescriptor)
//			}
//			return out
//		}
//
//		method := goopenrpcT.Method{
//			Name:        runtimeFunc.Name(), // FIXME or give me a comment.
//			Tags:        nil,
//			Summary:     methodSummary(astFuncDecl),
//			Description: description(),
//			ExternalDocs: goopenrpcT.ExternalDocs{
//				Description: fmt.Sprintf("%d", runtimeLine),
//				URL:         fmt.Sprintf("file://%s", runtimeFile), // TODO: Provide WORKING external docs links to Github (actually a wrapper/injection to make this configurable).
//			},
//			Params:         params(defaultContentDescriptorSkip),
//			Result:         rets(defaultContentDescriptorSkip)[0],
//			Deprecated:     methodDeprecated(astFuncDecl),
//			Servers:        nil,
//			Errors:         nil,
//			Links:          nil,
//			ParamStructure: "",
//			Examples:       nil,
//		}
//		b, _ := json.MarshalIndent(method, "", "  ")
//		t.Logf(string(b))
//
//		//spew.Config.DisableMethods = true
//		//t.Logf(spew.Sdump(astFuncDecl))
//
//		return &method
//	}
//
//	serv := new(MyService)
//
//	/* Don't do it this way. Bad. */
//	//a := Callback{
//	//	Receiver: reflect.ValueOf(serv),
//	//	Fn:       reflect.ValueOf(serv.Make),
//	//}
//	//
//	//synthesizeMethod(a)
//
//	/* Do it this way. Good. */
//	servVal := reflect.ValueOf(serv)
//	fnVal, ok := reflect.TypeOf(serv).MethodByName("Make")
//	if !ok {
//		t.Fatal("notok")
//	}
//
//	b := Callback{
//		Receiver: servVal,
//		Fn:       fnVal.Func,
//	}
//	synthesizeMethod(b)
//
//	//fnVal, ok = reflect.TypeOf(serv).MethodByName("Do")
//	//if !ok {
//	//	t.Fatal("notok")
//	//}
//	//c := Callback{
//	//	Receiver: servVal,
//	//	Fn:       fnVal.Func,
//	//}
//	//synthesizeMethod(c)
//
//	d := Callback{
//		Receiver: servVal,
//		Fn:       reflect.ValueOf(LoneRandomer),
//	}
//	synthesizeMethod(d)
//
//
//}
//func TestDiscover1(t *testing.T) {
//	//
//	//myserv := new(MyService)
//	////goserv := new(GoRPCRecvr)
//	//
//	//
//	//name := "serv_example"
//
//}
