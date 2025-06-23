package oas31

import (
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/values"
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

	// Register EitherValue types used in JSON Schema
	marshaller.RegisterType(func() *values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string] {
		return &values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string]{}
	})
	marshaller.RegisterType(func() *values.EitherValue[Schema, core.Schema, bool, bool] {
		return &values.EitherValue[Schema, core.Schema, bool, bool]{}
	})
}
