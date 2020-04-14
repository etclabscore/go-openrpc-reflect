package go_openrpc_reflect

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

type MethodTester struct {
	reflector           MethodRegisterer
	service             interface{}
	methods             map[string]string
	deprecated          []string
	descriptionMatches  map[string]string
	summaryMatches      map[string]string
	externalDocsMatches map[string]string
}

func (mt *MethodTester) name() string {
	return reflect.TypeOf(mt.reflector).Elem().Name()
}

func forEachMethod(receiver interface{}, fn func(method reflect.Method)) {
	ty := reflect.TypeOf(receiver)
	for m := 0; m < ty.NumMethod(); m++ {
		fn(ty.Method(m))
	}
}

func testMustGetASTFuncDecl(t *testing.T, r reflect.Value, method reflect.Method) *ast.FuncDecl {
	fdecl, err := getAstFuncDecl(r, method)
	if err != nil {
		t.Fatal(err)
	}
	return fdecl
}

func testMethodRegisterer(t *testing.T, tester *MethodTester) {
	if tester.methods != nil {
		t.Run("method eligibility", func(t *testing.T) {
			testEligibility(t, tester)
		})
		t.Run("method names", func(t *testing.T) {
			testMethodName(t, tester)
		})
	}
	if tester.descriptionMatches != nil {
		t.Run("descriptions", func(t *testing.T) {
			testMethodDescription(t, tester)
		})
	}
	if tester.deprecated != nil {
		t.Run("deprecated flag", func(t *testing.T) {
			testMethodDeprecated(t, tester)
		})
	}
	if tester.summaryMatches != nil {
		t.Run("summary", func(t *testing.T) {
			testMethodSummary(t, tester)
		})
	}
	if tester.externalDocsMatches != nil {
		t.Run("externalDocs", func(t *testing.T) {
			testMethodExternalDocs(t, tester)
		})
	}
}

func testEligibility(t *testing.T, tester *MethodTester) {
	gotEligibleMethods := []string{}
	wantEligibleMethods := []string{}
	for k := range tester.methods {
		wantEligibleMethods = append(wantEligibleMethods, k)
	}
	forEachMethod(tester.service, func(method reflect.Method) {
		if tester.reflector.IsMethodEligible(method) {
			gotEligibleMethods = append(gotEligibleMethods, method.Name)
		}
	})
	assert.ElementsMatch(t, wantEligibleMethods, gotEligibleMethods)
}

func testMethodName(t *testing.T, tester *MethodTester) {
	calcV := reflect.ValueOf(tester.service)
	forEachMethod(tester.service, func(method reflect.Method) {
		if !tester.reflector.IsMethodEligible(method) {
			return
		}

		wantName, ok := tester.methods[method.Name]
		if !ok {
			return
		}
		fdecl := testMustGetASTFuncDecl(t, calcV, method)
		gotName, err := tester.reflector.GetMethodName("", calcV, method, fdecl)
		assert.NoError(t, err)
		assert.Equal(t, wantName, gotName)
	})
}

func testMethodDeprecated(t *testing.T, tester *MethodTester) {
	calcV := reflect.ValueOf(tester.service)
	for _, d := range tester.deprecated {
		method, ok := reflect.TypeOf(tester.service).MethodByName(d)
		if !ok {
			t.Fatal("could not get method by name")
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)

		deprecated, err := tester.reflector.GetMethodDeprecated(calcV, method, fdecl)
		assert.NoError(t, err)
		assert.Equal(t, true, deprecated)
	}
}

func testMethodDescription(t *testing.T, tester *MethodTester) {
	calcV := reflect.ValueOf(tester.service)
	forEachMethod(tester.service, func(method reflect.Method) {
		if !tester.reflector.IsMethodEligible(method) {
			return
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)
		if fdecl == nil {
			t.Fatal(tester.service, method.Name, "fdecl is nil")
		}

		desc, err := tester.reflector.GetMethodDescription(calcV, method, fdecl)
		assert.NoError(t, err)

		for mre, re := range tester.descriptionMatches {
			if regexp.MustCompile(mre).MatchString(method.Name) {
				assert.Regexp(t, re, desc)
			}
		}
	})
}

func testMethodSummary(t *testing.T, tester *MethodTester) {
	calcV := reflect.ValueOf(tester.service)
	forEachMethod(tester.service, func(method reflect.Method) {
		if !tester.reflector.IsMethodEligible(method) {
			return
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)
		if fdecl == nil {
			t.Fatal(tester.service, method.Name, "fdecl is nil")
		}

		summary, err := tester.reflector.GetMethodSummary(calcV, method, fdecl)
		assert.NoError(t, err)

		for mre, re := range tester.summaryMatches {
			if regexp.MustCompile(mre).MatchString(method.Name) {
				assert.Regexp(t, re, summary)
			}
		}
	})
}

func testMethodExternalDocs(t *testing.T, tester *MethodTester) {
	calcV := reflect.ValueOf(tester.service)
	forEachMethod(tester.service, func(method reflect.Method) {
		if !tester.reflector.IsMethodEligible(method) {
			return
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)
		if fdecl == nil {
			t.Fatal(tester.service, method.Name, "fdecl is nil")
		}

		exdocs, err := tester.reflector.GetMethodExternalDocs(calcV, method, fdecl)
		assert.NoError(t, err)

		for mre, re := range tester.externalDocsMatches {
			if regexp.MustCompile(mre).MatchString(method.Name) {
				assert.Regexp(t, re, string(*exdocs.Url))
			}
		}
	})
}

// Helpers.

func testJSON(t *testing.T, jsonBytes []byte, want map[string]interface{}) {
	for k, v := range want {
		got := gjson.GetBytes(jsonBytes, k)
		if re, ok := v.(*regexp.Regexp); ok {
			assert.Regexp(t, re, got.Value().(string), k)
		} else {
			assert.Equal(t, v, got.Value(), k)
		}
	}
}

// Unit tests.

func TestExpandedFieldNamesFromList(t *testing.T) {
	var in []*ast.Field
	expanded := expandedFieldNamesFromList(in)
	assert.NotEqual(t, nil, expanded)
	assert.Len(t, expanded, 0)
}

func testUnnamedArgs(uint, int)          {}
func testTwoNamesOneType(a, b int)       {}
func testTwoNamesTwoTypes(a uint, b int) {}

// TestFieldsWithNames tests fieldsWithNames as well as printIdentField, because
// it the former uses the latter to assign an ident name to an unnamed field.
func TestFieldsWithNames(t *testing.T) {
	cases := []struct {
		fn        interface{}
		wantNames []string
	}{
		{testUnnamedArgs, []string{"uint", "int"}},
		{testTwoNamesOneType, []string{"a", "b"}},
		{testTwoNamesTwoTypes, []string{"a", "b"}},
	}

casesLoop:
	for _, c := range cases {

		// Set up ast.
		rf := runtime.FuncForPC(reflect.ValueOf(c.fn).Pointer())
		rtf, _ := rf.FileLine(rf.Entry())
		tokenFileSet := token.NewFileSet()
		astFile, err := parser.ParseFile(tokenFileSet, rtf, nil, parser.ParseComments)
		if err != nil {
			assert.NoError(t, err)
		}
		rfName := runtimeFuncBaseName(rf)

		// Find the func decl.
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)

			// Not it.
			if !ok {
				continue
			}
			if fn.Name == nil || fn.Name.Name != rfName {
				continue
			}

			// The magic.
			expandedFields := expandedFieldNamesFromList(fn.Type.Params.List)

			// Right number of fields.
			assert.Len(t, expandedFields, len(c.wantNames))
			for i, wantName := range c.wantNames {

				// Exactly one name per field.
				assert.Len(t, expandedFields[i].Names, 1)

				// The name we expect.
				assert.Equal(t, wantName, expandedFields[i].Names[0].Name)
			}

			// Prove our ast iter did actually find the fn we want.
			//
			continue casesLoop
		}
		t.Fatal("did not find test func declaration")
	}
}

type ContentDescriptorSelector struct {
	reflector  ContentDescriptorRegisterer
	service    interface{}
	methodName string
	isArg      bool
	fieldIndex int
}

func (mt *ContentDescriptorSelector) name() string {
	return reflect.TypeOf(mt.reflector).Elem().Name()
}

type SchemaSelector struct {
	ContentDescriptorSelector
}

func (mt *SchemaSelector) name() string {
	return reflect.TypeOf(mt.reflector).Elem().Name()
}
