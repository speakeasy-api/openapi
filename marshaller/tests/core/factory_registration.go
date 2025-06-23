package core

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values/core"
)

// init registers all core test model types with the factory system
func init() {
	// Register core test models
	marshaller.RegisterType(func() *TestPrimitiveModel {
		return &TestPrimitiveModel{}
	})

	marshaller.RegisterType(func() *TestComplexModel {
		return &TestComplexModel{}
	})

	marshaller.RegisterType(func() *TestEmbeddedMapModel {
		return &TestEmbeddedMapModel{}
	})

	marshaller.RegisterType(func() *TestEmbeddedMapWithFieldsModel {
		return &TestEmbeddedMapWithFieldsModel{}
	})

	marshaller.RegisterType(func() *TestEmbeddedMapWithExtensionsModel {
		return &TestEmbeddedMapWithExtensionsModel{}
	})

	marshaller.RegisterType(func() *TestNonCoreModel {
		return &TestNonCoreModel{}
	})

	marshaller.RegisterType(func() *TestCustomUnmarshalModel {
		return &TestCustomUnmarshalModel{}
	})

	marshaller.RegisterType(func() *TestEitherValueModel {
		return &TestEitherValueModel{}
	})

	marshaller.RegisterType(func() *TestValidationModel {
		return &TestValidationModel{}
	})

	marshaller.RegisterType(func() *TestAliasModel {
		return &TestAliasModel{}
	})

	marshaller.RegisterType(func() *TestRequiredPointerModel {
		return &TestRequiredPointerModel{}
	})

	marshaller.RegisterType(func() *TestRequiredNilableModel {
		return &TestRequiredNilableModel{}
	})

	marshaller.RegisterType(func() *TestTypeConversionCoreModel {
		return &TestTypeConversionCoreModel{}
	})

	marshaller.RegisterType(func() *marshaller.Node[*TestPrimitiveModel] {
		return &marshaller.Node[*TestPrimitiveModel]{}
	})

	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*TestPrimitiveModel]] {
		return &sequencedmap.Map[string, marshaller.Node[*TestPrimitiveModel]]{}
	})

	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[string]] {
		return &sequencedmap.Map[string, marshaller.Node[string]]{}
	})

	marshaller.RegisterType(func() *core.EitherValue[string, int] {
		return &core.EitherValue[string, int]{}
	})

	marshaller.RegisterType(func() *core.EitherValue[TestPrimitiveModel, int] {
		return &core.EitherValue[TestPrimitiveModel, int]{}
	})
}
