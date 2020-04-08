package openrpc_go_document

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/alecthomas/jsonschema"
	"github.com/go-openapi/spec"
)

func typeToSchema(opts *DocumentProviderParseOpts, ty reflect.Type) spec.Schema {
	if !jsonschemaPkgSupport(ty) {
		panic("FIXME")
	}

	rflctr := jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             false,
		IgnoredTypes:               opts.SchemaIgnoredTypes,
		TypeMapper:                 opts.TypeMapper,
	}

	jsch := rflctr.ReflectFromType(ty)

	// Poor man's glue.
	// Need to get the type from the go struct -> json reflector package
	// to the swagger/go-openapi/jsonschema spec.
	// Do this with JSON marshaling.
	// Hacky? Maybe. Effective? Maybe.
	m, err := json.Marshal(jsch)
	if err != nil {
		log.Fatal(err)
	}
	sch := spec.Schema{}
	err = json.Unmarshal(m, &sch)
	if err != nil {
		log.Fatal(err)
	}

	// Move pointer and slice type schemas to a child
	// of a oneOf schema with a sibling null schema.
	// Pointer and slice types can be nil.
	if ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		parentSch := spec.Schema{
			SchemaProps:        spec.SchemaProps{
				OneOf: []spec.Schema{
					sch,
					nullSchema,
				},
			},
		}
		sch = parentSch
	}

	// Again, this should be pluggable.
	handleDescriptionDefault := true
	if handleDescriptionDefault {
		if sch.Description == "" {
			sch.Description = fullTypeDescription(ty)
		}
	}

	return sch
}

func fullTypeDescription(ty reflect.Type) string {
	pre := ""
	if ty.Kind() == reflect.Ptr {
		pre = "*"
		ty = ty.Elem()
	}

	out := fmt.Sprintf(`%s%s`, pre, ty.Name())
	pack := ty.PkgPath()
	if pack != "" {
		return fmt.Sprintf("%s.%s", pack, out)
	}
	return out
}
