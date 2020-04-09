package go_openrpc_reflect

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"unicode"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

func DefaultParseOptions() *DocumentProviderParseOpts {
	return &DocumentProviderParseOpts{
		SchemaMutationFromTypeFns: []func(s *spec.Schema, ty reflect.Type){
			SchemaMutationSetDescriptionFromType,
			SchemaMutationNilableFromType,
		},

		ContentDescriptorMutationFns: nil,
		ContentDescriptorTypeSkipFn:  nil,
		TypeMapper: func(r reflect.Type) *jsonschema.Type {

			// Handle interface{}, which can be anything.
			if isEmptyInterfaceType(r) {
				return &jsonschema.Type{
					OneOf: []*jsonschema.Type{
						{Type: "array"},
						{Type: "object"},
						{Type: "string"},
						{Type: "number"},
						{Type: "integer"},
						{Type: "boolean"},
						{Type: "null"},
					},
				}
			}
			return nil
		},

		MethodBlackList:    nil,
		SchemaIgnoredTypes: nil,
		SchemaMutationFns: []func(*spec.Schema) error{
			SchemaMutationRequireDefaultOn,
			SchemaMutationExpand,
			SchemaMutationRemoveDefinitionsField,
		},
	}
}

var DefaultServerServiceProvider = &ServerDescriptorT{
	ServiceOpenRPCInfoFn: func() goopenrpcT.Info { return goopenrpcT.Info{} },
	ServiceOpenRPCExternalDocsFn: func() *goopenrpcT.ExternalDocs {
		return &goopenrpcT.ExternalDocs{
			Description: "GPLv3",
			URL:         "https://github.com/ethereum/go-ethereum/blob/COPYING.md",
		}
	},
}

var nullSchema = spec.Schema{
	SchemaProps: spec.SchemaProps{
		Type: []string{"null"},
	}}

var nullContentDescriptor = &goopenrpcT.ContentDescriptor{
	Content: goopenrpcT.Content{
		Name:   "Null",
		Schema: nullSchema,
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

func newMethod() *goopenrpcT.Method {
	return &goopenrpcT.Method{
		Name:           "REQUIRED",
		Tags:           []goopenrpcT.Tag{},
		Summary:        "",
		Description:    "",
		ExternalDocs:   goopenrpcT.ExternalDocs{},
		Params:         nil, // Required to set, leave nil.
		Result:         nil, // Required to set, leave nil.
		Deprecated:     false,
		Servers:        []goopenrpcT.Server{},
		Errors:         []goopenrpcT.Error{},
		Links:          []goopenrpcT.Link{},
		ParamStructure: "by-position",
		Examples:       []goopenrpcT.ExamplePairing{},
	}
}

func SchemaMutationRemoveDefinitionsField(s *spec.Schema) error {
	s.Definitions = nil
	return nil
}

func SchemaMutationSetDescriptionFromType(s *spec.Schema, ty reflect.Type) {
	if s.Description == "" {
		s.Description = fullTypeDescription(ty)
	}
}

func SchemaMutationNilableFromType(s *spec.Schema, ty reflect.Type) {
	/*
		Golang-specific schema mutations.
		This logic is not pluggable because it's language-specific,
		and should be applied to every schema no matter what.
	*/

	// Move pointer and slice type schemas to a child
	// of a oneOf schema with a sibling null schema.
	// Pointer and slice types can be nil.
	if ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		parentSch := spec.Schema{
			SchemaProps: spec.SchemaProps{
				OneOf: []spec.Schema{
					*s,
					nullSchema,
				},
			},
		}
		*s = parentSch
	}
}

func SchemaMutationExpand(s *spec.Schema) error {
	err := spec.ExpandSchema(s, s, nil)
	if err != nil {
		b, _ := json.MarshalIndent(s, "", "  ")
		return fmt.Errorf("schema Expand mutation error: %v schema:\n%s", err, string(b))
	}
	return nil
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

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

func isEmptyInterfaceType(ty reflect.Type) bool {
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	switch ty.Kind() {
	case reflect.Interface:
		if ty.NumMethod() == 0 {
			return true
		}
	}
	return false
}

// Is t context.Context or *context.Context?
func isContextType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == contextType
}

// Does t satisfy the error interface?
func isErrorType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Implements(errorType)
}

