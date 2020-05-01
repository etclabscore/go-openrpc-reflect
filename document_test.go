package go_openrpc_reflect

import (
	"encoding/json"
	"net"
	"regexp"
	"testing"
	"time"

	"github.com/etclabscore/go-openrpc-reflect/internal/fakemath"
	meta_schema "github.com/open-rpc/meta-schema"
	"github.com/stretchr/testify/assert"
)

//var TestMetaRegisterer = &MetaRegistererTester{}
var TestMetaRegisterer = &MetaT{
	GetServersFn:      getServers,
	GetInfoFn:         getInfo,
	GetExternalDocsFn: getExternalDocs,
}

func getInfo() (info *meta_schema.InfoObject) {
	title := "Calculator API"
	version := time.Now().Format(time.RFC3339)
	return &meta_schema.InfoObject{
		Title:          (*meta_schema.InfoObjectProperties)(&title),
		Description:    nil,
		TermsOfService: nil,
		Version:        (*meta_schema.InfoObjectVersion)(&version),
		Contact:        nil,
		License:        nil,
	}
}

func getExternalDocs() (*meta_schema.ExternalDocumentationObject) {
	return nil
}

func getServers() func (listeners []net.Listener) (*meta_schema.Servers, error) {
	return StandardReflector.GetServers()
}

func newDocument() *Document {
	return &Document{}
}

func TestDocument_Discover(t *testing.T) {
	t.Run("standard", func(t *testing.T) {
		d := newDocument().WithMeta(TestMetaRegisterer).WithReflector(StandardReflector)

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		defer listener.Close()

		d.RegisterListener(listener)

		calculatorRPC := new(fakemath.CalculatorRPC)
		d.RegisterReceiver(calculatorRPC)

		out, err := d.Discover()
		assert.NoError(t, err)

		b, _ := json.MarshalIndent(out, "", "  ")
		t.Log(string(b))

		jsonTests := map[string]interface{}{
			"openrpc":                 "1.2.4",
			"info.title":              "Calculator API",
			"info.version":            regexp.MustCompile(time.Now().Format("2006")),
			"servers.0.url":           listener.Addr().String(),
			"methods.#":               float64(5),
			"methods.0.name":          "CalculatorRPC.Add",
			"methods.0.params.#":      float64(1),
			"methods.0.params.0.name": "arg",
			"methods.0.params.0.schema.properties.a.type": "integer",
			"methods.0.result.name":                       "reply",
			"methods.0.result.schema.type":                "integer",
			"components":                                  nil,
		}

		testJSON(t, b, jsonTests)
	})

	t.Run("ethereum", func(t *testing.T) {
		d := newDocument().WithMeta(TestMetaRegisterer).WithReflector(EthereumReflector)

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		defer listener.Close()
		assert.NoError(t, err)

		d.RegisterListener(listener)

		calculator := new(fakemath.Calculator)
		d.RegisterReceiver(calculator)

		out, err := d.Discover()
		assert.NoError(t, err)

		b, _ := json.MarshalIndent(out, "", "  ")
		str := string(b)
		t.Log(str)

		jsonTests := map[string]interface{}{
			"openrpc":                        "1.2.4",
			"info.title":                     "Calculator API",
			"info.version":                   regexp.MustCompile(time.Now().Format("2006")),
			"servers.0.url":                  listener.Addr().String(),
			"methods.#":                      float64(10),
			"methods.0.name":                 "calculator_add",
			"methods.0.params.#":             float64(2),
			"methods.0.params.0.name":        "argA",
			"methods.0.params.0.schema.type": "integer",
			"methods.0.result.name":          "int",
			"methods.0.result.schema.type":   "integer",
			"components":                     nil,
		}

		testJSON(t, b, jsonTests)
	})
}