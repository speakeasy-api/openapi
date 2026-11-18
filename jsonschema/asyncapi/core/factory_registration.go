package core

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

// init registers all AsyncAPI schema core types with the marshaller factory system
func init() {
	marshaller.RegisterType(func() *Schema { return &Schema{} })
	marshaller.RegisterType(func() *ExternalDocumentation { return &ExternalDocumentation{} })
	marshaller.RegisterType(func() *MultiFormatSchema { return &MultiFormatSchema{} })
	marshaller.RegisterType(func() *SchemaDefinition { return &SchemaDefinition{} })
	marshaller.RegisterType(func() *SchemaUnion { return &SchemaUnion{} })
	marshaller.RegisterType(func() *Items { return &Items{} })
	marshaller.RegisterType(func() *Dependency { return &Dependency{} })

	marshaller.RegisterType(func() *values.EitherValue[Schema, bool] { return &values.EitherValue[Schema, bool]{} })
	marshaller.RegisterType(func() *values.EitherValue[[]marshaller.Node[string], string] {
		return &values.EitherValue[[]marshaller.Node[string], string]{}
	})

	marshaller.RegisterType(func() *sequencedmap.Map[string, *values.EitherValue[Schema, bool]] {
		return &sequencedmap.Map[string, *values.EitherValue[Schema, bool]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Dependency] {
		return &sequencedmap.Map[string, *Dependency]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *SchemaUnion] {
		return &sequencedmap.Map[string, *SchemaUnion]{}
	})
}
