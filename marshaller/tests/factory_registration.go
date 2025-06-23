package tests

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// init registers high-level test model types with the factory system
func init() {
	// Register high-level test models
	marshaller.RegisterType(func() *TestPrimitiveHighModel {
		return &TestPrimitiveHighModel{}
	})

	marshaller.RegisterType(func() *TestComplexHighModel {
		return &TestComplexHighModel{}
	})

	marshaller.RegisterType(func() *TestEmbeddedMapHighModel {
		return &TestEmbeddedMapHighModel{}
	})

	marshaller.RegisterType(func() *TestEmbeddedMapWithFieldsHighModel {
		return &TestEmbeddedMapWithFieldsHighModel{}
	})

	marshaller.RegisterType(func() *TestValidationHighModel {
		return &TestValidationHighModel{}
	})

	marshaller.RegisterType(func() *TestEitherValueHighModel {
		return &TestEitherValueHighModel{}
	})

	marshaller.RegisterType(func() *TestRequiredPointerHighModel {
		return &TestRequiredPointerHighModel{}
	})

	marshaller.RegisterType(func() *TestRequiredNilableHighModel {
		return &TestRequiredNilableHighModel{}
	})

	marshaller.RegisterType(func() *TestTypeConversionHighModel {
		return &TestTypeConversionHighModel{}
	})

	// Register custom types
	marshaller.RegisterType(func() *HTTPMethod {
		return new(HTTPMethod)
	})

	marshaller.RegisterType(func() *sequencedmap.Map[string, string] {
		return &sequencedmap.Map[string, string]{}
	})
}
