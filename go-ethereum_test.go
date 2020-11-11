package go_openrpc_reflect

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"

	"github.com/etclabscore/go-openrpc-reflect/internal/fakearithmetic"
	meta_schema "github.com/open-rpc/meta-schema"
	"github.com/stretchr/testify/assert"
)

func newEthereumMethodTester() *MethodTester {
	return &MethodTester{
		reflector: EthereumReflector,
		service:   &fakearithmetic.Calculator{},
		methods: map[string]string{
			"HasBatteries":      "calculator_hasBatteries",
			"Add":               "calculator_add",
			"Mul":               "calculator_mul",
			"BigMul":            "calculator_bigMul",
			"Div":               "calculator_div",
			"IsZero":            "calculator_isZero",
			"History":           "calculator_history",
			"Last":              "calculator_last",
			"GetRecord":         "calculator_getRecord",
			"Reset":             "calculator_reset",
			"SumWithContext":    "calculator_sumWithContext",
			"ConstructCircle":   "calculator_constructCircle",
			"GuessAreaOfCircle": "calculator_guessAreaOfCircle",
		},
		deprecated: []string{"Div"},
		descriptionMatches: map[string]string{
			".{1}": "(?m)^.*[a-z]+.*$", // Non empty.
			".{2}": `func\s+\(.*\)`,    // Contains func declaration.
		},
		summaryMatches: map[string]string{
			"HasBatteries":   `whether the calculator has`,
			"Add":            `two integers together`,
			"Div":            `Warning`,
			"SumWithContext": `Context as its first parameter`,
		},
		externalDocsMatches: map[string]string{
			"BigMul": `(?m)^https\:\/\/.*\.com.*\/internal\/fakearithmetic\/fakearithmetic\.go\#L\d+`,
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
		method, ok := reflect.TypeOf(&fakearithmetic.Calculator{}).MethodByName(m)
		assert.True(t, ok)
		assert.True(t, EthereumReflector.IsMethodEligible(method), method.Name)
	}

	for _, m := range []string{
		"ThreePseudoRandomNumbers",
		"LatestError",
	} {
		method, ok := reflect.TypeOf(&fakearithmetic.Calculator{}).MethodByName(m)
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
	methods, err := EthereumReflector.ReceiverMethods("", &fakearithmetic.Calculator{})
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
		`methods.#(name=="calculator_add").externalDocs.url`:         regexp.MustCompile(`(?m)^http.*github.*fakearithmetic.*fakearithmetic\.go`),

		`methods.#(name=="calculator_bigMul").name`:                                 "calculator_bigMul",
		`methods.#(name=="calculator_bigMul").summary`:                              regexp.MustCompile(`returns.*the product of.*`),
		`methods.#(name=="calculator_bigMul").params.#`:                             float64(2),
		`methods.#(name=="calculator_bigMul").params.0.name`:                        "argA",
		`methods.#(name=="calculator_bigMul").params.0.description`:                 "*big.Int",
		`methods.#(name=="calculator_bigMul").params.0.schema.type`:                 "object",
		`methods.#(name=="calculator_bigMul").params.0.schema.additionalProperties`: false,

		`methods.#(name=="calculator_div").deprecated`: true,

		`methods.#(name=="calculator_hasBatteries").result.name`:        "bool",
		`methods.#(name=="calculator_hasBatteries").result.schema.type`: "boolean",

		`methods.#(name=="calculator_history").params.#`:                                     float64(0),
		`methods.#(name=="calculator_history").result.name`:                                  "[]HistoryItem",
		`methods.#(name=="calculator_history").result.description`:                           "[]HistoryItem",
		`methods.#(name=="calculator_history").result.schema.type`:                           "array",
		`methods.#(name=="calculator_history").result.schema.items.0.type`:                   "object", // fail
		`methods.#(name=="calculator_history").result.schema.items.0.properties.Args.type`:   "array",  // fail
		`methods.#(name=="calculator_history").result.schema.items.0.properties.Method.type`: "string", // fail

		`methods.#(name=="calculator_last").params.#`:                             float64(0),
		`methods.#(name=="calculator_last").result.name`:                          "calculation",
		`methods.#(name=="calculator_last").result.description`:                   "*HistoryItem",
		`methods.#(name=="calculator_last").result.schema.properties.Args.type`:   "array",
		`methods.#(name=="calculator_last").result.schema.properties.Method.type`: "string",

		`methods.#(name=="calculator_reset").params.#`:           float64(0),
		`methods.#(name=="calculator_reset").result.name`:        "Null",
		`methods.#(name=="calculator_reset").result.schema.type`: "null",

		`methods.#(name=="calculator_constructCircle").result.name`: "*fakegeometry.Circle",

		`methods.#(name=="calculator_guessAreaOfCircle").params.0.name`: "*Pi",
		`methods.#(name=="calculator_guessAreaOfCircle").params.1.name`: "*fakegeometry.Circle",
		`methods.#(name=="calculator_guessAreaOfCircle").result.name`:   "float64",
	}

	testJSON(t, b, jsonTests)
}

func TestEthereumReflectorT_GetMethodParams(t *testing.T) {
	reflector := EthereumReflector
	service := &fakearithmetic.Calculator{}
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
			service:    &fakearithmetic.Calculator{},
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
			service:    &fakearithmetic.Calculator{},
			methodName: "HasBatteries",
			count:      0,
			params:     map[string]interface{}{},
		},
		{
			// Show that when the first parameter is context.Context,
			// it is ignored.
			service:    &fakearithmetic.Calculator{},
			methodName: "SumWithContext",
			count:      1,
			params: map[string]interface{}{
				"params.0.name":        "number",
				"params.0.schema.type": "integer",
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
	service := &fakearithmetic.Calculator{}
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
