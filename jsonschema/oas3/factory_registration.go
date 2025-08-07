package oas3

import (
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	valuesCore "github.com/speakeasy-api/openapi/values/core"
)

// init registers all JSON Schema OAS 3.1 types with the marshaller factory system
func init() {
	// Register all JSON Schema types
	marshaller.RegisterType(func() *Schema { return &Schema{} })
	marshaller.RegisterType(func() *Discriminator { return &Discriminator{} })
	marshaller.RegisterType(func() *ExternalDocumentation { return &ExternalDocumentation{} })
	marshaller.RegisterType(func() *XML { return &XML{} })
	marshaller.RegisterType(func() *SchemaType { return new(SchemaType) })
	marshaller.RegisterType(func() *[]SchemaType { return &[]SchemaType{} })
	marshaller.RegisterType(func() *valuesCore.EitherValue[*core.Schema, bool] {
		return &valuesCore.EitherValue[*core.Schema, bool]{}
	})
	marshaller.RegisterType(func() *JSONSchema[Referenceable] {
		return &JSONSchema[Referenceable]{}
	})
	marshaller.RegisterType(func() *JSONSchema[Concrete] {
		return &JSONSchema[Concrete]{}
	})

	// Register additional core EitherValue types
	marshaller.RegisterType(func() *valuesCore.EitherValue[bool, float64] {
		return &valuesCore.EitherValue[bool, float64]{}
	})
	marshaller.RegisterType(func() *values.EitherValue[bool, bool, float64, float64] {
		return &values.EitherValue[bool, bool, float64, float64]{}
	})

	// Register EitherValue types used in JSON Schema
	marshaller.RegisterType(func() *values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string] {
		return &values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string]{}
	})

	// Register sequencedmap.Map types used in JSON Schema
	marshaller.RegisterType(func() *sequencedmap.Map[string, *JSONSchema[Referenceable]] {
		return &sequencedmap.Map[string, *JSONSchema[Referenceable]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *JSONSchema[Concrete]] {
		return &sequencedmap.Map[string, *JSONSchema[Concrete]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *valuesCore.EitherValue[core.Schema, bool]] {
		return &sequencedmap.Map[string, *valuesCore.EitherValue[core.Schema, bool]]{}
	})
}
