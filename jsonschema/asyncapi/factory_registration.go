package asyncapi

import (
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
)

// init registers all AsyncAPI schema types with the marshaller factory system
func init() {
	marshaller.RegisterType(func() *Schema { return &Schema{} })
	marshaller.RegisterType(func() *JSONSchema { return &JSONSchema{} })
	marshaller.RegisterType(func() *ExternalDocumentation { return &ExternalDocumentation{} })
	marshaller.RegisterType(func() *MultiFormatSchema { return &MultiFormatSchema{} })
	marshaller.RegisterType(func() *SchemaDefinition { return &SchemaDefinition{} })
	marshaller.RegisterType(func() *SchemaUnion { return &SchemaUnion{} })

	marshaller.RegisterType(func() *values.EitherValue[Schema, core.Schema, bool, bool] {
		return &values.EitherValue[Schema, core.Schema, bool, bool]{}
	})
	marshaller.RegisterType(func() *Items { return &Items{} })
	marshaller.RegisterType(func() *Dependency { return &Dependency{} })
	marshaller.RegisterType(func() *Type { return &Type{} })
	marshaller.RegisterType(func() *[]*JSONSchema { return &[]*JSONSchema{} })

	marshaller.RegisterType(func() *sequencedmap.Map[string, *JSONSchema] {
		return &sequencedmap.Map[string, *JSONSchema]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Dependency] {
		return &sequencedmap.Map[string, *Dependency]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *SchemaUnion] {
		return &sequencedmap.Map[string, *SchemaUnion]{}
	})
}
