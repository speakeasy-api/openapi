package core

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/values/core"
)

// init registers all JSON Schema core types with the marshaller factory system
func init() {
	// Register all JSON Schema core types
	marshaller.RegisterType(func() *Schema { return &Schema{} })
	marshaller.RegisterType(func() *Discriminator { return &Discriminator{} })
	marshaller.RegisterType(func() *ExternalDocumentation { return &ExternalDocumentation{} })
	marshaller.RegisterType(func() *XML { return &XML{} })

	// Register EitherValue types used with JSON Schema
	marshaller.RegisterType(func() *core.EitherValue[Schema, bool] { return &core.EitherValue[Schema, bool]{} })
	marshaller.RegisterType(func() *core.EitherValue[[]marshaller.Node[string], string] {
		return &core.EitherValue[[]marshaller.Node[string], string]{}
	})
}
