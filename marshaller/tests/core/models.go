package core

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	valuescore "github.com/speakeasy-api/openapi/values/core"
	"gopkg.in/yaml.v3"
)

// TestPrimitiveModel covers all primitive marshaller.Node field types
type TestPrimitiveModel struct {
	marshaller.CoreModel `model:"testPrimitiveModel"`

	StringField     marshaller.Node[string]   `key:"stringField"`
	StringPtrField  marshaller.Node[*string]  `key:"stringPtrField"`
	BoolField       marshaller.Node[bool]     `key:"boolField"`
	BoolPtrField    marshaller.Node[*bool]    `key:"boolPtrField"`
	IntField        marshaller.Node[int]      `key:"intField"`
	IntPtrField     marshaller.Node[*int]     `key:"intPtrField"`
	Float64Field    marshaller.Node[float64]  `key:"float64Field"`
	Float64PtrField marshaller.Node[*float64] `key:"float64PtrField"`
	Extensions      core.Extensions           `key:"extensions"`
}

// TestRequiredPointerModel specifically tests required pointer field behavior
type TestRequiredPointerModel struct {
	marshaller.CoreModel `model:"testRequiredPointerModel"`

	RequiredPtr marshaller.Node[*string] `key:"requiredPtr" required:"true"`
	OptionalPtr marshaller.Node[*string] `key:"optionalPtr"`
	Extensions  core.Extensions          `key:"extensions"`
}

// TestComplexModel covers complex marshaller.Node field types
type TestComplexModel struct {
	marshaller.CoreModel `model:"testComplexModel"`

	NestedModel            marshaller.Node[*TestPrimitiveModel]                                `key:"nestedModel"`
	NestedModelValue       marshaller.Node[TestPrimitiveModel]                                 `key:"nestedModelValue"`
	ArrayField             marshaller.Node[[]string]                                           `key:"arrayField"`
	NodeArrayField         marshaller.Node[[]marshaller.Node[string]]                          `key:"nodeArrayField"`
	StructArrayField       marshaller.Node[[]*TestPrimitiveModel]                              `key:"structArrayField"`
	MapPrimitiveField      marshaller.Node[*sequencedmap.Map[string, string]]                  `key:"mapField"`
	MapNodeField           marshaller.Node[*sequencedmap.Map[string, marshaller.Node[string]]] `key:"mapNodeField"`
	MapStructField         marshaller.Node[*sequencedmap.Map[string, *TestPrimitiveModel]]     `key:"mapStructField"`
	EitherField            marshaller.Node[*valuescore.EitherValue[string, int]]               `key:"eitherField"`
	EitherModelOrPrimitive marshaller.Node[*valuescore.EitherValue[TestPrimitiveModel, int]]   `key:"eitherModelOrPrimitive" required:"true"`
	RawNodeField           marshaller.Node[*yaml.Node]                                         `key:"rawNodeField"`
	ValueField             marshaller.Node[valuescore.Value]                                   `key:"valueField"`
	ValuesField            marshaller.Node[[]marshaller.Node[valuescore.Value]]                `key:"valuesField"`
	Extensions             core.Extensions                                                     `key:"extensions"`
}

// TestEmbeddedMapModel covers embedded sequenced map scenarios with no extra fields
type TestEmbeddedMapModel struct {
	marshaller.CoreModel `model:"testEmbeddedMapModel"`

	*sequencedmap.Map[string, marshaller.Node[string]]
}

// TestEmbeddedMapWithFieldsModel covers embedded sequenced map with additional fields
type TestEmbeddedMapWithFieldsModel struct {
	marshaller.CoreModel `model:"testEmbeddedMapWithFieldsModel"`
	*sequencedmap.Map[string, marshaller.Node[*TestPrimitiveModel]]

	NameField  marshaller.Node[string] `key:"name"`
	Extensions core.Extensions         `key:"extensions"`
}

// TestEmbeddedMapWithExtensionsModel covers embedded sequenced map with extensions only
type TestEmbeddedMapWithExtensionsModel struct {
	marshaller.CoreModel `model:"testEmbeddedMapWithExtensionsModel"`
	*sequencedmap.Map[string, marshaller.Node[string]]

	Extensions core.Extensions `key:"extensions"`
}

// TestNonCoreModel represents a normal Go struct (not implementing CoreModeler)
type TestNonCoreModel struct {
	Name        string  `json:"name"`
	Value       int     `json:"value"`
	Description *string `json:"description,omitempty"`
}

// TestCustomUnmarshalModel implements custom Unmarshal method
type TestCustomUnmarshalModel struct {
	marshaller.CoreModel `model:"testCustomUnmarshalModel"`

	CustomField marshaller.Node[string] `key:"customField"`
	Extensions  core.Extensions         `key:"extensions"`

	// Custom state for testing
	UnmarshalCalled bool
}

var _ interfaces.CoreModel = (*TestCustomUnmarshalModel)(nil)

// Unmarshal implements custom unmarshalling logic
func (m *TestCustomUnmarshalModel) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	m.UnmarshalCalled = true

	// Use standard unmarshalling for the base
	return marshaller.UnmarshalModel(ctx, node, m)
}

// TestEitherValueModel covers EitherValue scenarios
type TestEitherValueModel struct {
	marshaller.CoreModel `model:"testEitherValueModel"`

	StringOrInt    marshaller.Node[*valuescore.EitherValue[string, int]]                `key:"stringOrInt"`
	ArrayOrString  marshaller.Node[*valuescore.EitherValue[[]string, string]]           `key:"arrayOrString"`
	StructOrString marshaller.Node[*valuescore.EitherValue[TestPrimitiveModel, string]] `key:"structOrString"`
	Extensions     core.Extensions                                                      `key:"extensions"`
}

// TestValidationModel covers field validation scenarios
type TestValidationModel struct {
	marshaller.CoreModel `model:"testValidationModel"`

	RequiredField  marshaller.Node[string]              `key:"requiredField" required:"true"`
	OptionalField  marshaller.Node[*string]             `key:"optionalField"`
	RequiredArray  marshaller.Node[[]string]            `key:"requiredArray" required:"true"`
	OptionalArray  marshaller.Node[[]string]            `key:"optionalArray"`
	RequiredStruct marshaller.Node[*TestPrimitiveModel] `key:"requiredStruct" required:"true"`
	OptionalStruct marshaller.Node[*TestPrimitiveModel] `key:"optionalStruct"`
	Extensions     core.Extensions                      `key:"extensions"`
}

// TestEmbeddedMapPointerModel represents core model with pointer embedded sequenced map
// This tests the legacy pointer embed pattern to ensure backward compatibility
type TestEmbeddedMapPointerModel struct {
	marshaller.CoreModel `model:"testEmbeddedMapPointerModel"`
	*sequencedmap.Map[string, marshaller.Node[string]]
}

// TestEmbeddedMapWithFieldsPointerModel represents core model with pointer embedded sequenced map and additional fields
// This tests the legacy pointer embed pattern with fields to ensure backward compatibility
type TestEmbeddedMapWithFieldsPointerModel struct {
	marshaller.CoreModel `model:"testEmbeddedMapWithFieldsPointerModel"`
	*sequencedmap.Map[string, marshaller.Node[*TestPrimitiveModel]]

	NameField  marshaller.Node[string] `key:"name"`
	Extensions core.Extensions         `key:"extensions"`
}

// TestAliasModel covers alias scenarios
type TestAliasModel struct {
	marshaller.CoreModel `model:"testAliasModel"`

	AliasField  marshaller.Node[string]              `key:"aliasField"`
	AliasArray  marshaller.Node[[]string]            `key:"aliasArray"`
	AliasStruct marshaller.Node[*TestPrimitiveModel] `key:"aliasStruct"`
	Extensions  core.Extensions                      `key:"extensions"`
}

// TestRequiredNilableModel specifically tests required tag with nilable types
type TestRequiredNilableModel struct {
	marshaller.CoreModel `model:"testRequiredNilableModel"`

	RequiredPtr     marshaller.Node[*string]                              `key:"requiredPtr" required:"true"`
	RequiredSlice   marshaller.Node[[]string]                             `key:"requiredSlice" required:"true"`
	RequiredMap     marshaller.Node[*sequencedmap.Map[string, string]]    `key:"requiredMap" required:"true"`
	RequiredStruct  marshaller.Node[*TestPrimitiveModel]                  `key:"requiredStruct" required:"true"`
	RequiredEither  marshaller.Node[*valuescore.EitherValue[string, int]] `key:"requiredEither" required:"true"`
	RequiredRawNode marshaller.Node[*yaml.Node]                           `key:"requiredRawNode" required:"true"`
	OptionalPtr     marshaller.Node[*string]                              `key:"optionalPtr"`
	OptionalSlice   marshaller.Node[[]string]                             `key:"optionalSlice"`
	OptionalMap     marshaller.Node[*sequencedmap.Map[string, string]]    `key:"optionalMap"`
	OptionalStruct  marshaller.Node[*TestPrimitiveModel]                  `key:"optionalStruct"`
	Extensions      core.Extensions                                       `key:"extensions"`
}

// TestTypeConversionCoreModel represents core model with string keys (like openapi/core/paths.go)
// This simulates the issue where core uses string keys but high-level model expects HTTPMethod keys
type TestTypeConversionCoreModel struct {
	marshaller.CoreModel `model:"testTypeConversionCoreModel"`

	*sequencedmap.Map[string, marshaller.Node[*TestPrimitiveModel]]
	HTTPMethodField marshaller.Node[*string] `key:"httpMethodField"`
	Extensions      core.Extensions          `key:"extensions"`
}

// TestSimpleArrayModel is a minimal model with only an array field for testing array sync behavior
type TestSimpleArrayModel struct {
	marshaller.CoreModel `model:"testSimpleArrayModel"`

	ArrayField marshaller.Node[[]string] `key:"arrayField"`
}

// TestItemModel represents a simple item with name and description
type TestItemModel struct {
	marshaller.CoreModel `model:"testItemModel"`

	Name        marshaller.Node[string] `key:"name"`
	Description marshaller.Node[string] `key:"description"`
}

// TestArrayOfObjectsModel is a minimal model with an array of objects for testing array sync behavior
type TestArrayOfObjectsModel struct {
	marshaller.CoreModel `model:"testArrayOfObjectsModel"`

	Items marshaller.Node[[]*TestItemModel] `key:"items"`
}
