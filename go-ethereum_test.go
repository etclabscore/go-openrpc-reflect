package go_openrpc_refract

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"

	"github.com/etclabscore/go-openrpc-refract/internal/fakemath"
	meta_schema "github.com/open-rpc/meta-schema"
	"github.com/stretchr/testify/assert"
)

func newEthereumMethodTester() *MethodTester {
	return &MethodTester{
		reflector: EthereumReflector,
		service:   &fakemath.Calculator{},
		methods: map[string]string{
			"HasBatteries": "calculator_hasBatteries",
			"Add":          "calculator_add",
			"Mul":          "calculator_mul",
			"BigMul":       "calculator_bigMul",
			"Div":          "calculator_div",
			"IsZero":       "calculator_isZero",
			"History":      "calculator_history",
			"Last":         "calculator_last",
			"GetRecord":    "calculator_getRecord",
			"Reset":        "calculator_reset",
		},
		deprecated: []string{"Div"},
		descriptionMatches: map[string]string{
			".{1}": "(?m)^.*[a-z]+.*$", // Non empty.
			".{2}": `func\s+\(.*\)`,    // Contains func declaration.
		},
		summaryMatches: map[string]string{
			"HasBatteries": `whether the calculator has`,
			"Add":          `two integers together`,
			"Div":          `Warning`,
		},
		externalDocsMatches: map[string]string{
			"BigMul": `(?m)^https\:\/\/.*\.com.*\/internal\/fakemath\/fakemath\.go\#L\d+`,
		},
	}
}

func TestEthereumReflectorT(t *testing.T) {
	testMethodRegisterer(t, newEthereumMethodTester())
}

func TestEthereumReflectorT_IsMethodEligible(t *testing.T) {
	for _, m := range []string{
		"HasBatteries",
		"Add",
		"BigMul",
		"Div",
		"Last",
		"GetRecord",
	} {
		method, ok := reflect.TypeOf(&fakemath.Calculator{}).MethodByName(m)
		assert.True(t, ok)
		assert.True(t, EthereumReflector.IsMethodEligible(method), method.Name)
	}

	for _, m := range []string{
		"ThreePseudoRandomNumbers",
		"LatestError",
	} {
		method, ok := reflect.TypeOf(&fakemath.Calculator{}).MethodByName(m)
		assert.True(t, ok)
		assert.False(t, EthereumReflector.IsMethodEligible(method), method.Name)
	}

}

// This doesn't exist because the Ethereum style uses the Standard style
// for content descriptors and schemas; the tests at standard_test.go are sufficient.
func newEthereumContentDescriptorTester() *ContentDescriptorSelector {
	return &ContentDescriptorSelector{}
}

func TestEthereumReflectorT_ReceiverMethods(t *testing.T) {
	methods, err := EthereumReflector.ReceiverMethods("", &fakemath.Calculator{})
	if !assert.NoError(t, err) {
		t.Fatal("ethereum methods")
	}
	if !assert.Len(t, methods, len(newEthereumMethodTester().methods)) {
		t.Fatal("ethereum methods")
	}

	type T struct {
		Methods []meta_schema.MethodObject `json:"methods"`
	}
	b, err := json.MarshalIndent(T{methods}, "", "  ")
	assert.NoError(t, err)

	t.Log(string(b))

	jsonTests := map[string]interface{}{
		`methods.#(name=="calculator_add").name`:           "calculator_add",
		`methods.#(name=="calculator_add").paramStructure`: "by-position",
		`methods.#(name=="calculator_add").deprecated`:     false,

		`methods.#(name=="calculator_add").params.#`:                    float64(2),
		`methods.#(name=="calculator_add").params.0.name`:               "argA",
		`methods.#(name=="calculator_add").params.0.description`:        "int",
		`methods.#(name=="calculator_add").params.0.required`:           true,
		`methods.#(name=="calculator_add").params.0.deprecated`:         false,
		`methods.#(name=="calculator_add").params.0.schema.type`:        "integer",
		`methods.#(name=="calculator_add").params.0.schema.definitions`: nil,

		`methods.#(name=="calculator_add").result.name`:              "int",
		`methods.#(name=="calculator_add").result.description`:       "int",
		`methods.#(name=="calculator_add").result.schema.type`:       "integer",
		`methods.#(name=="calculator_add").externalDocs.description`: regexp.MustCompile(`(?m)^Github remote link$`),
		`methods.#(name=="calculator_add").externalDocs.url`:         regexp.MustCompile(`(?m)^http.*github.*fakemath.*fakemath\.go`),

		`methods.#(name=="calculator_bigMul").name`:                                 "calculator_bigMul",
		`methods.#(name=="calculator_bigMul").summary`:                              regexp.MustCompile(`returns.*the product of.*`),
		`methods.#(name=="calculator_bigMul").params.#`:                             float64(2),
		`methods.#(name=="calculator_bigMul").params.0.name`:                        "argA",
		`methods.#(name=="calculator_bigMul").params.0.description`:                 "*big.Int",
		`methods.#(name=="calculator_bigMul").params.0.schema.type`:                 "object",
		`methods.#(name=="calculator_bigMul").params.0.schema.additionalProperties`: map[string]interface{}{},

		`methods.#(name=="calculator_div").deprecated`: true,

		`methods.#(name=="calculator_hasBatteries").result.name`:        "bool",
		`methods.#(name=="calculator_hasBatteries").result.schema.type`: "boolean",

		`methods.#(name=="calculator_history").params.#`:                                   float64(0),
		`methods.#(name=="calculator_history").result.name`:                                "[]HistoryItem",
		`methods.#(name=="calculator_history").result.description`:                         "[]HistoryItem",
		`methods.#(name=="calculator_history").result.schema.type`:                         "array",
		`methods.#(name=="calculator_history").result.schema.items.type`:                   "object",
		`methods.#(name=="calculator_history").result.schema.items.properties.Args.type`:   "array",
		`methods.#(name=="calculator_history").result.schema.items.properties.Method.type`: "string",

		`methods.#(name=="calculator_last").params.#`:                             float64(0),
		`methods.#(name=="calculator_last").result.name`:                          "calculation",
		`methods.#(name=="calculator_last").result.description`:                   "*HistoryItem",
		`methods.#(name=="calculator_last").result.schema.properties.Args.type`:   "array",
		`methods.#(name=="calculator_last").result.schema.properties.Method.type`: "string",

		`methods.#(name=="calculator_reset").params.#`:           float64(0),
		`methods.#(name=="calculator_reset").result.name`:        "Null",
		`methods.#(name=="calculator_reset").result.schema.type`: "null",
	}

	testJSON(t, b, jsonTests)
}

func TestEthereumReflectorT_GetMethodParams(t *testing.T) {
	reflector := EthereumReflector
	service := &fakemath.Calculator{}
	cases := []struct {
		service    interface{}
		methodName string
		count      int
		params     map[string]interface{}
	}{
		{
			service:    service,
			methodName: "Add",
			count:      2,
			params: map[string]interface{}{
				"params.0.name":               "argA",
				"params.0.description":        "int",
				"params.0.summary":            "",
				"params.0.required":           true,
				"params.0.deprecated":         false,
				"params.0.schema.type":        "integer",
				"params.0.schema.definitions": nil,

				"params.1.name":               "argB",
				"params.1.description":        "int",
				"params.1.summary":            "",
				"params.1.required":           true,
				"params.1.deprecated":         false,
				"params.1.schema.type":        "integer",
				"params.1.schema.definitions": nil,
			},
		},
		{
			service:    &fakemath.Calculator{},
			methodName: "BigMul",
			count:      2,
			params: map[string]interface{}{
				"params.0.name":        "argA",
				"params.0.description": "*big.Int",
				"params.0.summary":     "",
				"params.0.required":    true,
				"params.0.deprecated":  false,
				"params.0.schema.type": "object",

				"params.1.name":        "argB",
				"params.1.description": "*big.Int",
				"params.1.summary":     "",
				"params.1.required":    true,
				"params.1.deprecated":  false,
				"params.1.schema.type": "object",
			},
		},
		{
			service:    &fakemath.Calculator{},
			methodName: "HasBatteries",
			count:      0,
			params:     map[string]interface{}{},
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
		assert.Len(t, gotParams, c.count)

		type T struct {
			Params []meta_schema.ContentDescriptorObject `json:"params"`
		}

		b, _ := json.MarshalIndent(T{gotParams}, "", "  ")

		t.Log(string(b))

		testJSON(t, b, c.params)
	}
}

func TestEthereumReflectorT_GetMethodResult(t *testing.T) {
	reflector := EthereumReflector
	service := &fakemath.Calculator{}
	cases := []struct {
		service    interface{}
		methodName string
		result     map[string]interface{}
	}{
		{service, "Add", map[string]interface{}{
			"name":        "int",
			"description": "int",
			"summary":     "",
			"required":    true,
			"deprecated":  false,
			"schema.type": "integer",
		},
		},
		{service, "Div", map[string]interface{}{
			"name":        "Null",
			"description": "Null",
			"summary":     nil,
			"required":    true,
			"deprecated":  false,
			"schema.type": "null",
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
