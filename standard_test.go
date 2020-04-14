package go_openrpc_reflect

import (
	"encoding/json"
	"net"
	"reflect"
	"regexp"
	"testing"

	"github.com/etclabscore/go-openrpc-refract/internal/fakemath"
	meta_schema "github.com/open-rpc/meta-schema"
	"github.com/stretchr/testify/assert"
)

func TestStandardReflectorT_GetServers(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:3000")
	assert.NoError(t, err)

	getServersFn := StandardReflector.GetServers()
	servers, err := getServersFn([]net.Listener{listener})
	assert.Len(t, ([]meta_schema.ServerObject)(*servers), 1)
	assert.Equal(t, string(*([]meta_schema.ServerObject)(*servers)[0].Name), "tcp")
	assert.Equal(t, string(*([]meta_schema.ServerObject)(*servers)[0].Url), "127.0.0.1:3000")
}

func newStandardMethodTester() *MethodTester {
	return &MethodTester{
		reflector: StandardReflector,
		service:   &fakemath.CalculatorRPC{},
		methods: map[string]string{
			"HasBatteries": "CalculatorRPC.HasBatteries",
			"Add":          "CalculatorRPC.Add",
			"BigMul":       "CalculatorRPC.BigMul",
			"Div":          "CalculatorRPC.Div",
			"IsZero":       "CalculatorRPC.IsZero",
		},
		deprecated: []string{"Div"},
		descriptionMatches: map[string]string{
			".{1}": "(?m)^.*[a-z]+.*$", // Non empty.
			".{2}": `func\s+\(.*\).*`,    // Contains func declaration.
		},
		summaryMatches: map[string]string{
			"HasBatteries": `if the calculator has batteries`,
			"Add":          `sums the A and B fields`,
			"Div":          `Use Mul instead`,
			"IsZero":       `throwaway parameters`,
		},
		externalDocsMatches: map[string]string{
			"BigMul": `(?m)^https\:\/\/.*\.com.*\/internal\/fakemath\/fakemath\.go\#L\d+`,
		},
	}
}

func TestStandardReflectorT_ReceiverMethods(t *testing.T) {
	methods, err := StandardReflector.ReceiverMethods("", &fakemath.CalculatorRPC{})
	if !assert.NoError(t, err) {
		t.Fatal("standard methods")
	}
	if !assert.Len(t, methods, len(newStandardMethodTester().methods)) {
		t.Fatal("standard methods")
	}

	type T struct {
		Methods []meta_schema.MethodObject `json:"methods"`
	}
	b, err := json.MarshalIndent(T{methods}, "", "  ")
	assert.NoError(t, err)

	t.Log(string(b))

	jsonTests := map[string]interface{}{
		`methods.#(name=="CalculatorRPC.Add").name`:                              "CalculatorRPC.Add",
		`methods.#(name=="CalculatorRPC.Add").paramStructure`:                    "by-position",
		`methods.#(name=="CalculatorRPC.Add").deprecated`:                        false,
		`methods.#(name=="CalculatorRPC.Add").params.#`:                          float64(1),
		`methods.#(name=="CalculatorRPC.Add").params.0.name`:                     "arg",
		`methods.#(name=="CalculatorRPC.Add").params.0.description`:              "AddArg",
		`methods.#(name=="CalculatorRPC.Add").params.0.required`:                 true,
		`methods.#(name=="CalculatorRPC.Add").params.0.deprecated`:               false,
		`methods.#(name=="CalculatorRPC.Add").params.0.schema.type`:              "object",
		`methods.#(name=="CalculatorRPC.Add").params.0.schema.properties.a.type`: "integer",
		`methods.#(name=="CalculatorRPC.Add").params.0.schema.properties.b.type`: "integer",
		`methods.#(name=="CalculatorRPC.Add").params.0.schema.definitions`:       nil,
		`methods.#(name=="CalculatorRPC.Add").result.name`:                       "reply",
		`methods.#(name=="CalculatorRPC.Add").result.description`:                "*AddReply",
		`methods.#(name=="CalculatorRPC.Add").result.schema.type`:                "integer",
		`methods.#(name=="CalculatorRPC.Add").externalDocs.description`:          regexp.MustCompile(`(?m)^Github remote link$`),
		`methods.#(name=="CalculatorRPC.Add").externalDocs.url`:                  regexp.MustCompile(`(?m)^http.*github.*fakemath.*fakemath\.go`),

		`methods.#(name=="CalculatorRPC.BigMul").name`:                              "CalculatorRPC.BigMul",
		`methods.#(name=="CalculatorRPC.BigMul").params.#`:                          float64(1),
		`methods.#(name=="CalculatorRPC.BigMul").params.0.name`:                     "arg",
		`methods.#(name=="CalculatorRPC.BigMul").params.0.description`:              "BigMulArg",
		`methods.#(name=="CalculatorRPC.BigMul").params.0.schema.type`:              "object",
		`methods.#(name=="CalculatorRPC.BigMul").params.0.schema.properties.B.type`: "object",
		`methods.#(name=="CalculatorRPC.BigMul").params.0.schema.properties.a.type`: "object",

		`methods.#(name=="CalculatorRPC.HasBatteries").result.name`:        "reply",
		`methods.#(name=="CalculatorRPC.HasBatteries").result.schema.type`: "boolean",

		`methods.#(name=="CalculatorRPC.Div").deprecated`: true,

		`methods.#(name=="CalculatorRPC.IsZero").params.0.name`:        "big.Int",
		`methods.#(name=="CalculatorRPC.IsZero").params.0.description`: "big.Int",
		`methods.#(name=="CalculatorRPC.IsZero").params.0.schema.type`: "object",
		`methods.#(name=="CalculatorRPC.IsZero").result.name`:          "*IsZeroArg",
	}

	testJSON(t, b, jsonTests)
}

func TestStandardReflector(t *testing.T) {
	testMethodRegisterer(t, newStandardMethodTester())
}

func TestStandardReflectorT_ContentDescriptor(t *testing.T) {
	cases := []struct {
		ContentDescriptorSelector
		want map[string]string
	}{
		{
			ContentDescriptorSelector: ContentDescriptorSelector{
				StandardReflector,
				&fakemath.CalculatorRPC{},
				"HasBatteries",
				true,
				0,
			},
			want: map[string]string{"name": "arg", "summary": "", "description": "HasBatteriesArg"},
		},
		{
			ContentDescriptorSelector: ContentDescriptorSelector{
				StandardReflector,
				&fakemath.CalculatorRPC{},
				"Add",
				true,
				1,
			},
			want: map[string]string{"name": "reply", "summary": "", "description": "*AddReply"},
		},
		{
			ContentDescriptorSelector: ContentDescriptorSelector{
				EthereumReflector,
				&fakemath.Calculator{},
				"HasBatteries",
				false,
				0,
			},
			want: map[string]string{"name": "bool", "summary": "", "description": "bool"},
		},
		{
			ContentDescriptorSelector: ContentDescriptorSelector{
				EthereumReflector,
				&fakemath.Calculator{},
				"Add",
				true,
				0,
			},
			want: map[string]string{"name": "argA", "summary": "", "description": "int"},
		},
	}

	for _, c := range cases {
		calcV := reflect.ValueOf(c.service)
		method, ok := reflect.TypeOf(c.service).MethodByName(c.methodName)
		if !ok {
			t.Fatal("could not get method by name")
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)

		fields := fdecl.Type.Params.List
		if !c.isArg {
			fields = fdecl.Type.Results.List
		}
		assert.GreaterOrEqual(t, len(fields)-1, c.fieldIndex)

		for k, v := range c.want {
			switch {
			case k == "name":
				gotName, err := c.reflector.GetContentDescriptorName(calcV, method, fields[c.fieldIndex])
				assert.NoError(t, err)
				assert.Equal(t, v, gotName)

			case k == "summary":
				gotSummary, err := c.reflector.GetContentDescriptorSummary(calcV, method, fields[c.fieldIndex])
				assert.NoError(t, err)
				assert.Equal(t, v, gotSummary)

			case k == "description":
				gotDescription, err := c.reflector.GetContentDescriptorDescription(calcV, method, fields[c.fieldIndex])
				assert.NoError(t, err)
				assert.Equal(t, v, gotDescription)
			}
		}
	}
}

func TestStandardReflectorT_GetSchema(t *testing.T) {
	cases := []struct {
		SchemaSelector
		want map[string]string
	}{
		{
			SchemaSelector: SchemaSelector{
				ContentDescriptorSelector: ContentDescriptorSelector{
					StandardReflector,
					&fakemath.CalculatorRPC{},
					"HasBatteries",
					true,
					0,
				},
			},
			want: map[string]string{"type": "string"},
		},
		{
			SchemaSelector: SchemaSelector{
				ContentDescriptorSelector: ContentDescriptorSelector{
					StandardReflector,
					&fakemath.CalculatorRPC{},
					"BigMul",
					true,
					1,
				},
			},
			want: map[string]string{"type": "object"},
		},
	}

	for _, c := range cases {
		calcV := reflect.ValueOf(c.service)
		method, ok := reflect.TypeOf(c.service).MethodByName(c.methodName)
		if !ok {
			t.Fatal("could not get method by name")
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)

		fields := fdecl.Type.Params.List
		if !c.isArg {
			fields = fdecl.Type.Results.List
		}
		assert.GreaterOrEqual(t, len(fields)-1, c.fieldIndex)

		var ty reflect.Type
		if c.isArg {
			ty = method.Type.In(c.fieldIndex + 1) // receiver @ index=0
		} else {
			ty = method.Type.Out(c.fieldIndex)
		}

		schema, err := c.reflector.GetSchema(calcV, method, fields[c.fieldIndex], ty)
		assert.NoError(t, err)

		for k, v := range c.want {
			switch {
			case k == "type":
				assert.Equal(t, v, *schema.Type.Any17L18NF5)
			}
		}
	}
}

func TestStandardReflectorT_GetMethodParams(t *testing.T) {
	reflector := StandardReflector
	cases := []struct {
		service    interface{}
		methodName string
		params     []map[string]interface{}
	}{
		{
			service:    &fakemath.CalculatorRPC{},
			methodName: "Add",
			params: []map[string]interface{}{
				map[string]interface{}{
					"name":                     "arg",
					"description":              "AddArg",
					"summary":                  "",
					"required":                 true,
					"deprecated":               false,
					"schema.type":              "object",
					"schema.properties.a.type": "integer",
					"schema.definitions":       nil,
				},
			},
		},
		{
			service:    &fakemath.CalculatorRPC{},
			methodName: "Div",
			params: []map[string]interface{}{
				map[string]interface{}{
					"name":                     "arg",
					"description":              "DivArg",
					"summary":                  "",
					"required":                 true,
					"deprecated":               false,
					"schema.type":              "object",
					"schema.properties.a.type": "integer",
					"schema.properties.b.type": "integer",
				},
			},
		},
	}
	for _, c := range cases {
		calcV := reflect.ValueOf(c.service)
		method, ok := reflect.TypeOf(c.service).MethodByName(c.methodName)
		if !ok {
			t.Fatal("could not get method by name")
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)

		gotParams, err := reflector.GetMethodParams(calcV, method, fdecl)
		assert.NoError(t, err)
		assert.Len(t, gotParams, len(c.params))
		assert.Len(t, gotParams, 1)

		b, _ := json.MarshalIndent(gotParams[0], "", "  ")
		t.Log(string(b))


		for _, wantTest := range c.params {
			testJSON(t, b, wantTest)
		}
	}
}

func TestStandardReflectorT_GetMethodResult(t *testing.T) {
	reflector := StandardReflector
	cases := []struct {
		service    interface{}
		methodName string
		result     map[string]interface{}
	}{
		{
			service:    &fakemath.CalculatorRPC{},
			methodName: "Add",
			result: map[string]interface{}{
				"name":        "reply",
				"description": "*AddReply",
				"summary":     "",
				"schema.type": "integer",
			},
		},
		{
			service:    &fakemath.CalculatorRPC{},
			methodName: "Div",
			result: map[string]interface{}{
				"name":        "reply",
				"description": "*DivReply",
				"summary":     "",
				"schema.type": "integer",
			},
		},
	}
	for _, c := range cases {
		calcV := reflect.ValueOf(c.service)
		method, ok := reflect.TypeOf(c.service).MethodByName(c.methodName)
		if !ok {
			t.Fatal("could not get method by name")
		}

		fdecl := testMustGetASTFuncDecl(t, calcV, method)

		gotResult, err := reflector.GetMethodResult(calcV, method, fdecl)
		assert.NoError(t, err)

		b, _ := json.MarshalIndent(gotResult, "", "  ")
		t.Log(string(b))

		testJSON(t, b, c.result)
	}
}
