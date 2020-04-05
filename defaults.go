package openrpc_go_document

import (
	"fmt"
	"go/ast"
	"reflect"
	"regexp"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

var nullContentDescriptor = &goopenrpcT.ContentDescriptor{
	Content: goopenrpcT.Content{
		Name: "Null",
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"Null"},
			},
		},
	},
}

type NoReceiverT interface{}
var NoReceiver = reflect.TypeOf(((*NoReceiverT)(nil))).Elem()
var NoReceiverValue = reflect.ValueOf(new(NoReceiverT))

func NewSpec() *goopenrpcT.OpenRPCSpec1 {
	return &goopenrpcT.OpenRPCSpec1{
		OpenRPC: "1.2.4",
		Info:    goopenrpcT.Info{},
		Servers: []goopenrpcT.Server{},
		Methods: []goopenrpcT.Method{},
		Components: goopenrpcT.Components{
			ContentDescriptors:    make(map[string]*goopenrpcT.ContentDescriptor),
			Schemas:               make(map[string]spec.Schema),
			Examples:              make(map[string]goopenrpcT.Example),
			Links:                 make(map[string]goopenrpcT.Link),
			Errors:                make(map[string]goopenrpcT.Error),
			ExamplePairingObjects: make(map[string]goopenrpcT.ExamplePairing),
			Tags:                  make(map[string]goopenrpcT.Tag),
		},
		ExternalDocs: goopenrpcT.ExternalDocs{},
		Objects:      goopenrpcT.NewObjectMap(),
	}
}

var DefaultDocumentProviderParseOpts = &DocumentProviderParseOpts{
	SchemaMutationFns:            []func(s *spec.Schema) error{
		SchemaMutationRequireDefaultOn,
		SchemaMutationExpand,
		SchemaMutationRemoveDefinitionsField,
	},
	ContentDescriptorMutationFns: nil,
	MethodBlackList:              nil,
	TypeMapper: func(r reflect.Type) *jsonschema.Type {
		switch r.Kind() {
		case reflect.String:
		}
		return nil
	},
	SchemaIgnoredTypes:      nil,
	ContentDescriptorSkipFn: nil,
}

func SchemaMutationRemoveDefinitionsField(s *spec.Schema) error {
	s.Definitions = nil
	return nil
}

func SchemaMutationExpand(s *spec.Schema) error {
	return spec.ExpandSchema(s, s, nil)
}

func SchemaMutationRequireDefaultOn(s *spec.Schema) error {
	// If we didn't explicitly set any fields as required with jsonschema tags,
	// then we can assume the default, that ALL properties are required.
	if len(s.Required) == 0 {
		for k := range s.Properties {
			s.Required = append(s.Required, k)
		}
	}
	return nil
}

// GoRPCServiceMethods gets the methods available following the standard rpc library
// pattern. Receiver type names are joined to method type names by a dot.
func GoRPCServiceMethods(service interface{}) func() map[string]Callback {
	return func() map[string]Callback {

		result := make(map[string]Callback)

		rcvr := reflect.ValueOf(service)
		fmt.Println("rcvr val", rcvr)

		for n := 0; n < rcvr.NumMethod(); n++ {
			m := reflect.TypeOf(service).Method(n)
			methodName := rcvr.Elem().Type().Name() + "." + m.Name
			result[methodName] = Callback{reflect.ValueOf(service), m.Func}
		}
		return result
	}
}

func defaultContentDescriptorSkip(isArgs bool, index int, cd *goopenrpcT.ContentDescriptor) bool {
	if isArgs {
		if cd.Schema.Description == "context.Context" {
			return true
		}
	}
	return false
}

func documentValHasContext(rcvr reflect.Value, val reflect.Value) bool {
	fntype := val.Type()
	// Skip receiver and context.Context parameter (if present).
	firstArg := 0
	if rcvr.IsValid() && !rcvr.IsNil() {
		firstArg++
	}
	return fntype.NumIn() > firstArg && fntype.In(firstArg) == contextType
}

/*
	These following functions summary, description, etc.
	should maybe be overrideable or configurable...
	The general idea is that we're just mapping a base couple data
	types that we have available (reflect, runtime, ast) onto
	the method, content descriptor, or schema.
	As is, these are hardcoded opinions about how to handle these mappings
	from "introspected" values (reflect, runtime, ast) onto a data structure.
*/

func methodSummary(fdecl *ast.FuncDecl) string {
	if fdecl.Doc != nil {
		return fdecl.Doc.Text()
	}
	return ""
}
func methodDeprecated(fdecl *ast.FuncDecl) bool {
	matched, _ := regexp.MatchString(`(?im)deprecated`, methodSummary(fdecl))
	return matched
}

func isDiscoverMethodBlacklisted(d *DocumentProviderParseOpts, name string) bool {
	if d != nil && len(d.MethodBlackList) > 0 {
		for _, b := range d.MethodBlackList {
			if regexp.MustCompile(b).MatchString(name) {
				return true
			}
		}
	}
	return false
}
