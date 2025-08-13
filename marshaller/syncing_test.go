package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSync_PrimitiveTypes_Success(t *testing.T) {
	t.Parallel()

	// Create a high-level model with data
	highModel := tests.TestPrimitiveHighModel{
		StringField:     "synced string",
		StringPtrField:  pointer.From("synced ptr string"),
		BoolField:       true,
		BoolPtrField:    pointer.From(false),
		IntField:        99,
		IntPtrField:     pointer.From(88),
		Float64Field:    9.99,
		Float64PtrField: pointer.From(8.88),
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify all fields were synced correctly
	require.Equal(t, "synced string", coreModel.StringField.Value)
	require.NotNil(t, coreModel.StringPtrField.Value)
	require.Equal(t, "synced ptr string", *coreModel.StringPtrField.Value)
	require.True(t, coreModel.BoolField.Value)
	require.NotNil(t, coreModel.BoolPtrField.Value)
	require.False(t, *coreModel.BoolPtrField.Value)
	require.Equal(t, 99, coreModel.IntField.Value)
	require.NotNil(t, coreModel.IntPtrField.Value)
	require.Equal(t, 88, *coreModel.IntPtrField.Value)
	require.InDelta(t, 9.99, coreModel.Float64Field.Value, 0.001)
	require.NotNil(t, coreModel.Float64PtrField.Value)
	require.InDelta(t, 8.88, *coreModel.Float64PtrField.Value, 0.001)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `stringField: synced string
stringPtrField: synced ptr string
boolField: true
boolPtrField: false
intField: 99
intPtrField: 88
float64Field: 9.99
float64PtrField: 8.88
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_PrimitiveTypes_NilPointers_Success(t *testing.T) {
	t.Parallel()

	// Create a high-level model with nil pointer fields
	highModel := tests.TestPrimitiveHighModel{
		StringField:     "required string",
		StringPtrField:  nil, // nil pointer
		BoolField:       true,
		BoolPtrField:    nil, // nil pointer
		IntField:        42,
		IntPtrField:     nil, // nil pointer
		Float64Field:    3.14,
		Float64PtrField: nil, // nil pointer
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify required fields were synced
	require.Equal(t, "required string", coreModel.StringField.Value)
	require.True(t, coreModel.BoolField.Value)
	require.Equal(t, 42, coreModel.IntField.Value)
	require.InDelta(t, 3.14, coreModel.Float64Field.Value, 0.001)

	// Verify nil pointer fields are nil in core model
	require.Nil(t, coreModel.StringPtrField.Value)
	require.Nil(t, coreModel.BoolPtrField.Value)
	require.Nil(t, coreModel.IntPtrField.Value)
	require.Nil(t, coreModel.Float64PtrField.Value)

	// Verify the core model's RootNode contains the correct YAML (nil fields should be omitted)
	expectedYAML := `stringField: required string
boolField: true
intField: 42
float64Field: 3.14
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_ComplexTypes_Success(t *testing.T) {
	t.Parallel()

	// Create nested model
	nestedModel := &tests.TestPrimitiveHighModel{
		StringField:  "nested synced",
		BoolField:    true,
		IntField:     200,
		Float64Field: 2.22,
	}

	// Create nested model value
	nestedModelValue := tests.TestPrimitiveHighModel{
		StringField:  "value synced",
		BoolField:    false,
		IntField:     300,
		Float64Field: 3.33,
	}

	// Create a high-level model with complex data
	highModel := tests.TestComplexHighModel{
		NestedModel:      nestedModel,
		NestedModelValue: nestedModelValue,
		ArrayField:       []string{"sync1", "sync2", "sync3"},
		NodeArrayField:   []string{"node1", "node2"},
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify nested model was synced
	require.NotNil(t, coreModel.NestedModel.Value)
	nestedCore := coreModel.NestedModel.Value
	require.Equal(t, "nested synced", nestedCore.StringField.Value)
	require.True(t, nestedCore.BoolField.Value)
	require.Equal(t, 200, nestedCore.IntField.Value)
	require.InDelta(t, 2.22, nestedCore.Float64Field.Value, 0.001)

	// Verify nested model value was synced
	nestedValueCore := coreModel.NestedModelValue.Value
	require.Equal(t, "value synced", nestedValueCore.StringField.Value)
	require.False(t, nestedValueCore.BoolField.Value)
	require.Equal(t, 300, nestedValueCore.IntField.Value)
	require.InDelta(t, 3.33, nestedValueCore.Float64Field.Value, 0.001)

	// Verify array field was synced
	arrayValue := coreModel.ArrayField.Value
	require.Len(t, arrayValue, 3)
	require.Equal(t, "sync1", arrayValue[0])
	require.Equal(t, "sync2", arrayValue[1])
	require.Equal(t, "sync3", arrayValue[2])

	// Verify node array field was synced
	nodeArrayValue := coreModel.NodeArrayField.Value
	require.Len(t, nodeArrayValue, 2)
	require.Equal(t, "node1", nodeArrayValue[0].Value)
	require.Equal(t, "node2", nodeArrayValue[1].Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `nestedModel:
    stringField: nested synced
    boolField: true
    intField: 200
    float64Field: 2.22
nestedModelValue:
    stringField: value synced
    boolField: false
    intField: 300
    float64Field: 3.33
arrayField:
    - sync1
    - sync2
    - sync3
nodeArrayField:
    - node1
    - node2
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_RequiredNilableTypes_Success(t *testing.T) {
	t.Parallel()

	// Create nested struct
	nestedStruct := &tests.TestPrimitiveHighModel{
		StringField:  "nested required synced",
		BoolField:    true,
		IntField:     500,
		Float64Field: 5.55,
	}

	// Create a high-level model with required nilable types
	highModel := tests.TestRequiredNilableHighModel{
		RequiredPtr:    pointer.From("required synced"),
		RequiredSlice:  []string{"req1", "req2"},
		RequiredStruct: nestedStruct,
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify required fields were synced
	require.NotNil(t, coreModel.RequiredPtr.Value)
	require.Equal(t, "required synced", *coreModel.RequiredPtr.Value)

	sliceValue := coreModel.RequiredSlice.Value
	require.Len(t, sliceValue, 2)
	require.Equal(t, "req1", sliceValue[0])
	require.Equal(t, "req2", sliceValue[1])

	require.NotNil(t, coreModel.RequiredStruct.Value)
	structCore := coreModel.RequiredStruct.Value
	require.Equal(t, "nested required synced", structCore.StringField.Value)
	require.True(t, structCore.BoolField.Value)
	require.Equal(t, 500, structCore.IntField.Value)
	require.InDelta(t, 5.55, structCore.Float64Field.Value, 0.001)

	// Verify optional fields are nil
	require.Nil(t, coreModel.OptionalPtr.Value)
	require.Nil(t, coreModel.OptionalSlice.Value)
	require.Nil(t, coreModel.OptionalMap.Value)
	require.Nil(t, coreModel.OptionalStruct.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `requiredPtr: required synced
requiredSlice:
    - req1
    - req2
requiredStruct:
    stringField: nested required synced
    boolField: true
    intField: 500
    float64Field: 5.55
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_RequiredPointer_Success(t *testing.T) {
	t.Parallel()

	// Create a high-level model with required pointer
	highModel := tests.TestRequiredPointerHighModel{
		RequiredPtr: pointer.From("required synced ptr"),
		OptionalPtr: pointer.From("optional synced ptr"),
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify required pointer field
	require.NotNil(t, coreModel.RequiredPtr.Value)
	require.Equal(t, "required synced ptr", *coreModel.RequiredPtr.Value)

	// Verify optional pointer field
	require.NotNil(t, coreModel.OptionalPtr.Value)
	require.Equal(t, "optional synced ptr", *coreModel.OptionalPtr.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `requiredPtr: required synced ptr
optionalPtr: optional synced ptr
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMapWithFields_Success(t *testing.T) {
	t.Parallel()

	// Create dynamic values for the embedded map
	dynamicVal1 := &tests.TestPrimitiveHighModel{
		StringField:  "synced dynamic 1",
		BoolField:    true,
		IntField:     111,
		Float64Field: 1.11,
	}

	dynamicVal2 := &tests.TestPrimitiveHighModel{
		StringField:  "synced dynamic 2",
		BoolField:    false,
		IntField:     222,
		Float64Field: 2.22,
	}

	// Create a high-level model with embedded map and fields
	highModel := tests.TestEmbeddedMapWithFieldsHighModel{
		NameField: "synced name",
	}

	// Initialize the embedded map
	highModel.Map = *sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
	highModel.Set("syncKey1", dynamicVal1)
	highModel.Set("syncKey2", dynamicVal2)

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify regular field
	require.Equal(t, "synced name", coreModel.NameField.Value)

	// Verify embedded map was synced
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 2, coreModel.Len())
	require.True(t, coreModel.Has("syncKey1"))
	require.True(t, coreModel.Has("syncKey2"))

	// Verify dynamic field values
	syncedVal1, ok1 := coreModel.Get("syncKey1")
	require.True(t, ok1)
	require.NotNil(t, syncedVal1)
	syncedCore1 := syncedVal1.Value
	require.Equal(t, "synced dynamic 1", syncedCore1.StringField.Value)
	require.True(t, syncedCore1.BoolField.Value)

	syncedVal2, ok2 := coreModel.Get("syncKey2")
	require.True(t, ok2)
	require.NotNil(t, syncedVal2)
	syncedCore2 := syncedVal2.Value
	require.Equal(t, "synced dynamic 2", syncedCore2.StringField.Value)
	require.False(t, syncedCore2.BoolField.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `syncKey1:
    stringField: synced dynamic 1
    boolField: true
    intField: 111
    float64Field: 1.11
syncKey2:
    stringField: synced dynamic 2
    boolField: false
    intField: 222
    float64Field: 2.22
name: synced name
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMap_Success(t *testing.T) {
	t.Parallel()

	// Create a high-level model with embedded map
	highModel := tests.TestEmbeddedMapHighModel{}

	// Initialize the embedded map
	highModel.Map = *sequencedmap.New[string, string]()
	highModel.Set("syncKey1", "synced value1")
	highModel.Set("syncKey2", "synced value2")
	highModel.Set("syncKey3", "synced value3")

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify embedded map was synced
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 3, coreModel.Len())
	require.True(t, coreModel.Has("syncKey1"))
	require.True(t, coreModel.Has("syncKey2"))
	require.True(t, coreModel.Has("syncKey3"))

	// Verify values
	val1, ok1 := coreModel.Get("syncKey1")
	require.True(t, ok1)
	require.Equal(t, "synced value1", val1.Value)

	val2, ok2 := coreModel.Get("syncKey2")
	require.True(t, ok2)
	require.Equal(t, "synced value2", val2.Value)

	val3, ok3 := coreModel.Get("syncKey3")
	require.True(t, ok3)
	require.Equal(t, "synced value3", val3.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `syncKey1: synced value1
syncKey2: synced value2
syncKey3: synced value3
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_Validation_Success(t *testing.T) {
	t.Parallel()

	// Create nested structs
	requiredStruct := &tests.TestPrimitiveHighModel{
		StringField:  "synced required nested",
		BoolField:    true,
		IntField:     600,
		Float64Field: 6.66,
	}

	optionalStruct := &tests.TestPrimitiveHighModel{
		StringField:  "synced optional nested",
		BoolField:    false,
		IntField:     700,
		Float64Field: 7.77,
	}

	// Create a high-level model with validation data
	highModel := tests.TestValidationHighModel{
		RequiredField:  "synced required",
		OptionalField:  pointer.From("synced optional"),
		RequiredArray:  []string{"sync1", "sync2"},
		OptionalArray:  []string{"opt1", "opt2"},
		RequiredStruct: requiredStruct,
		OptionalStruct: optionalStruct,
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify required fields
	require.Equal(t, "synced required", coreModel.RequiredField.Value)
	require.NotNil(t, coreModel.OptionalField.Value)
	require.Equal(t, "synced optional", *coreModel.OptionalField.Value)

	// Verify arrays
	requiredArrayValue := coreModel.RequiredArray.Value
	require.Len(t, requiredArrayValue, 2)
	require.Equal(t, "sync1", requiredArrayValue[0])
	require.Equal(t, "sync2", requiredArrayValue[1])

	optionalArrayValue := coreModel.OptionalArray.Value
	require.Len(t, optionalArrayValue, 2)
	require.Equal(t, "opt1", optionalArrayValue[0])
	require.Equal(t, "opt2", optionalArrayValue[1])

	// Verify nested structs
	require.NotNil(t, coreModel.RequiredStruct.Value)
	requiredStructCore := coreModel.RequiredStruct.Value
	require.Equal(t, "synced required nested", requiredStructCore.StringField.Value)
	require.True(t, requiredStructCore.BoolField.Value)
	require.Equal(t, 600, requiredStructCore.IntField.Value)
	require.InDelta(t, 6.66, requiredStructCore.Float64Field.Value, 0.001)

	require.NotNil(t, coreModel.OptionalStruct.Value)
	optionalStructCore := coreModel.OptionalStruct.Value
	require.Equal(t, "synced optional nested", optionalStructCore.StringField.Value)
	require.False(t, optionalStructCore.BoolField.Value)
	require.Equal(t, 700, optionalStructCore.IntField.Value)
	require.InDelta(t, 7.77, optionalStructCore.Float64Field.Value, 0.001)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `requiredField: synced required
optionalField: synced optional
requiredArray:
    - sync1
    - sync2
optionalArray:
    - opt1
    - opt2
requiredStruct:
    stringField: synced required nested
    boolField: true
    intField: 600
    float64Field: 6.66
optionalStruct:
    stringField: synced optional nested
    boolField: false
    intField: 700
    float64Field: 7.77
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_PrimitiveTypes_WithExtensions_Success(t *testing.T) {
	t.Parallel()

	// Create a high-level model with extensions
	highModel := tests.TestPrimitiveHighModel{
		StringField:     "synced string",
		StringPtrField:  pointer.From("synced ptr string"),
		BoolField:       true,
		BoolPtrField:    pointer.From(false),
		IntField:        99,
		IntPtrField:     pointer.From(88),
		Float64Field:    9.99,
		Float64PtrField: pointer.From(8.88),
	}

	// Initialize extensions
	highModel.Extensions = &extensions.Extensions{}
	highModel.Extensions.Init()
	highModel.Extensions.Set("x-custom", testutils.CreateStringYamlNode("extension value", 1, 1))
	highModel.Extensions.Set("x-another", testutils.CreateStringYamlNode("another extension", 1, 1))

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify all fields were synced correctly
	require.Equal(t, "synced string", coreModel.StringField.Value)
	require.NotNil(t, coreModel.StringPtrField.Value)
	require.Equal(t, "synced ptr string", *coreModel.StringPtrField.Value)
	require.True(t, coreModel.BoolField.Value)
	require.NotNil(t, coreModel.BoolPtrField.Value)
	require.False(t, *coreModel.BoolPtrField.Value)
	require.Equal(t, 99, coreModel.IntField.Value)
	require.NotNil(t, coreModel.IntPtrField.Value)
	require.Equal(t, 88, *coreModel.IntPtrField.Value)
	require.InDelta(t, 9.99, coreModel.Float64Field.Value, 0.001)
	require.NotNil(t, coreModel.Float64PtrField.Value)
	require.InDelta(t, 8.88, *coreModel.Float64PtrField.Value, 0.001)

	// Verify extensions were synced
	require.NotNil(t, coreModel.Extensions)
	customExt, ok := coreModel.Extensions.Get("x-custom")
	require.True(t, ok)
	require.NotNil(t, customExt.Value)

	anotherExt, ok := coreModel.Extensions.Get("x-another")
	require.True(t, ok)
	require.NotNil(t, anotherExt.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `stringField: synced string
stringPtrField: synced ptr string
boolField: true
boolPtrField: false
intField: 99
intPtrField: 88
float64Field: 9.99
float64PtrField: 8.88
x-custom: extension value
x-another: another extension
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_EitherValueModel_Success(t *testing.T) {
	t.Parallel()

	// Create either values
	stringOrInt := &values.EitherValue[string, string, int, int]{}
	stringValue := "either string value"
	stringOrInt.Left = &stringValue

	arrayOrString := &values.EitherValue[[]string, []string, string, string]{}
	arrayValue := []string{"either1", "either2"}
	arrayOrString.Left = &arrayValue

	structOrString := &values.EitherValue[tests.TestPrimitiveHighModel, core.TestPrimitiveModel, string, string]{}
	structValue := tests.TestPrimitiveHighModel{
		StringField:  "either struct",
		BoolField:    true,
		IntField:     123,
		Float64Field: 1.23,
	}
	structOrString.Left = &structValue

	// Create a high-level model with either values
	highModel := tests.TestEitherValueHighModel{
		StringOrInt:    stringOrInt,
		ArrayOrString:  arrayOrString,
		StructOrString: structOrString,
	}

	// Initialize extensions
	highModel.Extensions = &extensions.Extensions{}
	highModel.Extensions.Init()
	highModel.Extensions.Set("x-either", testutils.CreateStringYamlNode("either extension", 1, 1))

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify either values were synced
	require.NotNil(t, coreModel.StringOrInt.Value)
	require.True(t, coreModel.StringOrInt.Value.IsLeft)

	require.NotNil(t, coreModel.ArrayOrString.Value)
	require.True(t, coreModel.ArrayOrString.Value.IsLeft)

	require.NotNil(t, coreModel.StructOrString.Value)
	require.True(t, coreModel.StructOrString.Value.IsLeft)

	// Verify extensions were synced
	require.NotNil(t, coreModel.Extensions)
	eitherExt, ok := coreModel.Extensions.Get("x-either")
	require.True(t, ok)
	require.NotNil(t, eitherExt.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `stringOrInt: either string value
arrayOrString:
    - either1
    - either2
structOrString:
    stringField: either struct
    boolField: true
    intField: 123
    float64Field: 1.23
x-either: either extension
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_TypeConversionModel_Success(t *testing.T) {
	t.Parallel()

	// Create operations for the embedded map with HTTPMethod keys
	postOp := &tests.TestPrimitiveHighModel{
		StringField:  "Synced POST operation",
		BoolField:    true,
		IntField:     42,
		Float64Field: 3.14,
	}

	getOp := &tests.TestPrimitiveHighModel{
		StringField:  "Synced GET operation",
		BoolField:    false,
		IntField:     100,
		Float64Field: 1.23,
	}

	putOp := &tests.TestPrimitiveHighModel{
		StringField:  "Synced PUT operation",
		BoolField:    true,
		IntField:     200,
		Float64Field: 2.34,
	}

	// Create a high-level model with HTTPMethod keys
	httpMethodValue := tests.HTTPMethodPost
	highModel := tests.TestTypeConversionHighModel{
		HTTPMethodField: &httpMethodValue,
	}

	// Initialize the embedded map with HTTPMethod keys
	highModel.Map = *sequencedmap.New[tests.HTTPMethod, *tests.TestPrimitiveHighModel]()
	highModel.Set(tests.HTTPMethodPost, postOp)
	highModel.Set(tests.HTTPMethodGet, getOp)
	highModel.Set(tests.HTTPMethodPut, putOp)

	// Initialize extensions
	highModel.Extensions = &extensions.Extensions{}
	highModel.Extensions.Init()
	highModel.Extensions.Set("x-sync", testutils.CreateStringYamlNode("sync extension", 1, 1))

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify HTTPMethod field was synced (converted back to string)
	require.True(t, coreModel.HTTPMethodField.Present)
	require.NotNil(t, coreModel.HTTPMethodField.Value)
	require.Equal(t, "post", *coreModel.HTTPMethodField.Value)

	// Verify embedded map was synced (HTTPMethod keys converted to string keys)
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 3, coreModel.Len())
	require.True(t, coreModel.Has("post"))
	require.True(t, coreModel.Has("get"))
	require.True(t, coreModel.Has("put"))

	// Verify operation values
	syncedPostOp, ok1 := coreModel.Get("post")
	require.True(t, ok1)
	require.NotNil(t, syncedPostOp)
	syncedPostCore := syncedPostOp.Value
	require.Equal(t, "Synced POST operation", syncedPostCore.StringField.Value)
	require.True(t, syncedPostCore.BoolField.Value)
	require.Equal(t, 42, syncedPostCore.IntField.Value)
	require.InDelta(t, 3.14, syncedPostCore.Float64Field.Value, 0.001)

	syncedGetOp, ok2 := coreModel.Get("get")
	require.True(t, ok2)
	require.NotNil(t, syncedGetOp)
	syncedGetCore := syncedGetOp.Value
	require.Equal(t, "Synced GET operation", syncedGetCore.StringField.Value)
	require.False(t, syncedGetCore.BoolField.Value)
	require.Equal(t, 100, syncedGetCore.IntField.Value)
	require.InDelta(t, 1.23, syncedGetCore.Float64Field.Value, 0.001)

	syncedPutOp, ok3 := coreModel.Get("put")
	require.True(t, ok3)
	require.NotNil(t, syncedPutOp)
	syncedPutCore := syncedPutOp.Value
	require.Equal(t, "Synced PUT operation", syncedPutCore.StringField.Value)
	require.True(t, syncedPutCore.BoolField.Value)
	require.Equal(t, 200, syncedPutCore.IntField.Value)
	require.InDelta(t, 2.34, syncedPutCore.Float64Field.Value, 0.001)

	// Verify extensions were synced
	require.NotNil(t, coreModel.Extensions)
	syncExt, ok := coreModel.Extensions.Get("x-sync")
	require.True(t, ok)
	require.NotNil(t, syncExt.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `post:
    stringField: Synced POST operation
    boolField: true
    intField: 42
    float64Field: 3.14
get:
    stringField: Synced GET operation
    boolField: false
    intField: 100
    float64Field: 1.23
put:
    stringField: Synced PUT operation
    boolField: true
    intField: 200
    float64Field: 2.34
httpMethodField: post
x-sync: sync extension
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_ExtensionModification_Success(t *testing.T) {
	t.Parallel()

	// Create a model with initial extensions
	highModel := tests.TestPrimitiveHighModel{
		StringField:  "model with extensions",
		BoolField:    true,
		IntField:     42,
		Float64Field: 3.14,
	}

	// Initialize extensions
	highModel.Extensions = &extensions.Extensions{}
	highModel.Extensions.Init()
	highModel.Extensions.Set("x-version", testutils.CreateStringYamlNode("1.0", 1, 1))
	highModel.Extensions.Set("x-author", testutils.CreateStringYamlNode("developer", 1, 1))

	// Perform initial sync
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify initial extensions were synced
	coreModel := highModel.GetCore()
	require.NotNil(t, coreModel.Extensions)

	versionExt, ok := coreModel.Extensions.Get("x-version")
	require.True(t, ok)
	require.Equal(t, "1.0", versionExt.Value.Value)

	authorExt, ok := coreModel.Extensions.Get("x-author")
	require.True(t, ok)
	require.Equal(t, "developer", authorExt.Value.Value)

	// Modify extensions: update existing, add new, remove one
	highModel.Extensions.Set("x-version", testutils.CreateStringYamlNode("2.0", 1, 1))   // Update
	highModel.Extensions.Set("x-status", testutils.CreateStringYamlNode("active", 1, 1)) // Add new
	highModel.Extensions.Delete("x-author")                                              // Remove

	// Sync the changes
	resultNode, err = marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify extensions were updated correctly
	updatedVersionExt, ok := coreModel.Extensions.Get("x-version")
	require.True(t, ok)
	require.Equal(t, "2.0", updatedVersionExt.Value.Value)

	statusExt, ok := coreModel.Extensions.Get("x-status")
	require.True(t, ok)
	require.Equal(t, "active", statusExt.Value.Value)

	// Verify removed extension is gone
	_, ok = coreModel.Extensions.Get("x-author")
	require.False(t, ok)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `stringField: model with extensions
boolField: true
intField: 42
float64Field: 3.14
x-version: "2.0"
x-status: active
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_ExtensionReplacement_Success(t *testing.T) {
	t.Parallel()

	// Create a model with extensions that will be completely replaced
	highModel := tests.TestPrimitiveHighModel{
		StringField:  "model for replacement",
		BoolField:    false,
		IntField:     100,
		Float64Field: 1.5,
	}

	// Initialize with original extensions
	highModel.Extensions = &extensions.Extensions{}
	highModel.Extensions.Init()
	highModel.Extensions.Set("x-original-id", testutils.CreateStringYamlNode("orig-123", 1, 1))
	highModel.Extensions.Set("x-legacy-flag", testutils.CreateStringYamlNode("true", 1, 1))
	highModel.Extensions.Set("x-deprecated", testutils.CreateStringYamlNode("soon", 1, 1))

	// Perform initial sync
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify initial state
	coreModel := highModel.GetCore()
	require.NotNil(t, coreModel.Extensions)
	require.Equal(t, 3, coreModel.Extensions.Len())

	// Replace with completely new extensions (simulating workflow replacement scenario)
	newExtensions := &extensions.Extensions{}
	newExtensions.Init()
	newExtensions.Set("x-new-id", testutils.CreateStringYamlNode("new-456", 1, 1))
	newExtensions.Set("x-modern-flag", testutils.CreateStringYamlNode("enabled", 1, 1))

	// Replace the extensions entirely
	highModel.Extensions = newExtensions

	// Sync the replacement
	resultNode, err = marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify all old extensions are gone and new ones are present
	require.Equal(t, 2, coreModel.Extensions.Len())

	newIdExt, ok := coreModel.Extensions.Get("x-new-id")
	require.True(t, ok)
	require.Equal(t, "new-456", newIdExt.Value.Value)

	modernFlagExt, ok := coreModel.Extensions.Get("x-modern-flag")
	require.True(t, ok)
	require.Equal(t, "enabled", modernFlagExt.Value.Value)

	// Verify old extensions are completely removed
	_, ok = coreModel.Extensions.Get("x-original-id")
	require.False(t, ok)
	_, ok = coreModel.Extensions.Get("x-legacy-flag")
	require.False(t, ok)
	_, ok = coreModel.Extensions.Get("x-deprecated")
	require.False(t, ok)

	// Verify the core model's RootNode contains only new extensions
	expectedYAML := `stringField: model for replacement
boolField: false
intField: 100
float64Field: 1.5
x-new-id: new-456
x-modern-flag: enabled
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMapPointer_Success(t *testing.T) {
	t.Parallel()

	// Create a high-level model with pointer embedded map (legacy pattern)
	highModel := tests.TestEmbeddedMapPointerHighModel{}

	// Initialize the pointer embedded map
	highModel.Map = sequencedmap.New[string, string]()
	highModel.Set("ptrKey1", "pointer value1")
	highModel.Set("ptrKey2", "pointer value2")
	highModel.Set("ptrKey3", "pointer value3")

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify embedded map was synced
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 3, coreModel.Len())
	require.True(t, coreModel.Has("ptrKey1"))
	require.True(t, coreModel.Has("ptrKey2"))
	require.True(t, coreModel.Has("ptrKey3"))

	// Verify values
	val1, ok1 := coreModel.Get("ptrKey1")
	require.True(t, ok1)
	require.Equal(t, "pointer value1", val1.Value)

	val2, ok2 := coreModel.Get("ptrKey2")
	require.True(t, ok2)
	require.Equal(t, "pointer value2", val2.Value)

	val3, ok3 := coreModel.Get("ptrKey3")
	require.True(t, ok3)
	require.Equal(t, "pointer value3", val3.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `ptrKey1: pointer value1
ptrKey2: pointer value2
ptrKey3: pointer value3
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMapWithFieldsPointer_Success(t *testing.T) {
	t.Parallel()

	// Create dynamic values for the pointer embedded map
	dynamicVal1 := &tests.TestPrimitiveHighModel{
		StringField:  "synced pointer dynamic 1",
		BoolField:    true,
		IntField:     111,
		Float64Field: 1.11,
	}

	dynamicVal2 := &tests.TestPrimitiveHighModel{
		StringField:  "synced pointer dynamic 2",
		BoolField:    false,
		IntField:     222,
		Float64Field: 2.22,
	}

	// Create a high-level model with pointer embedded map and fields
	highModel := tests.TestEmbeddedMapWithFieldsPointerHighModel{
		NameField: "synced pointer name",
	}

	// Initialize the pointer embedded map
	highModel.Map = sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
	highModel.Set("ptrSyncKey1", dynamicVal1)
	highModel.Set("ptrSyncKey2", dynamicVal2)

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify regular field
	require.Equal(t, "synced pointer name", coreModel.NameField.Value)

	// Verify pointer embedded map was synced
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 2, coreModel.Len())
	require.True(t, coreModel.Has("ptrSyncKey1"))
	require.True(t, coreModel.Has("ptrSyncKey2"))

	// Verify dynamic field values
	syncedVal1, ok1 := coreModel.Get("ptrSyncKey1")
	require.True(t, ok1)
	require.NotNil(t, syncedVal1)
	syncedCore1 := syncedVal1.Value
	require.Equal(t, "synced pointer dynamic 1", syncedCore1.StringField.Value)
	require.True(t, syncedCore1.BoolField.Value)

	syncedVal2, ok2 := coreModel.Get("ptrSyncKey2")
	require.True(t, ok2)
	require.NotNil(t, syncedVal2)
	syncedCore2 := syncedVal2.Value
	require.Equal(t, "synced pointer dynamic 2", syncedCore2.StringField.Value)
	require.False(t, syncedCore2.BoolField.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `ptrSyncKey1:
    stringField: synced pointer dynamic 1
    boolField: true
    intField: 111
    float64Field: 1.11
ptrSyncKey2:
    stringField: synced pointer dynamic 2
    boolField: false
    intField: 222
    float64Field: 2.22
name: synced pointer name
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMapComparison_PointerVsValue_Success(t *testing.T) {
	t.Parallel()

	t.Run("PointerEmbedBehavior", func(t *testing.T) {
		t.Parallel()
		// Test pointer embedded map
		ptrModel := tests.TestEmbeddedMapPointerHighModel{}
		ptrModel.Map = sequencedmap.New[string, string]()
		ptrModel.Set("key1", "ptr_value1")
		ptrModel.Set("key2", "ptr_value2")

		ptrResultNode, err := marshaller.SyncValue(t.Context(), &ptrModel, ptrModel.GetCore(), ptrModel.GetRootNode(), false)
		require.NoError(t, err)
		require.NotNil(t, ptrResultNode)

		ptrCoreModel := ptrModel.GetCore()
		require.NotNil(t, ptrCoreModel.Map)
		require.Equal(t, 2, ptrCoreModel.Len())

		ptrVal1, ok := ptrCoreModel.Get("key1")
		require.True(t, ok)
		require.Equal(t, "ptr_value1", ptrVal1.Value)
	})

	t.Run("ValueEmbedBehavior", func(t *testing.T) {
		t.Parallel()
		// Test value embedded map
		valueModel := tests.TestEmbeddedMapHighModel{}
		valueModel.Map = *sequencedmap.New[string, string]()
		valueModel.Set("key1", "val_value1")
		valueModel.Set("key2", "val_value2")

		valueResultNode, err := marshaller.SyncValue(t.Context(), &valueModel, valueModel.GetCore(), valueModel.GetRootNode(), false)
		require.NoError(t, err)
		require.NotNil(t, valueResultNode)

		valueCoreModel := valueModel.GetCore()
		require.NotNil(t, valueCoreModel.Map)
		require.Equal(t, 2, valueCoreModel.Len())

		valueVal1, ok := valueCoreModel.Get("key1")
		require.True(t, ok)
		require.Equal(t, "val_value1", valueVal1.Value)
	})

	t.Run("BothProduceSameResult", func(t *testing.T) {
		t.Parallel()
		// Verify both pointer and value embeds produce equivalent results
		ptrModel := tests.TestEmbeddedMapPointerHighModel{}
		ptrModel.Map = sequencedmap.New[string, string]()
		ptrModel.Set("shared_key", "shared_value")

		valueModel := tests.TestEmbeddedMapHighModel{}
		valueModel.Map = *sequencedmap.New[string, string]()
		valueModel.Set("shared_key", "shared_value")

		// Sync both models
		_, err := marshaller.SyncValue(t.Context(), &ptrModel, ptrModel.GetCore(), ptrModel.GetRootNode(), false)
		require.NoError(t, err)

		_, err = marshaller.SyncValue(t.Context(), &valueModel, valueModel.GetCore(), valueModel.GetRootNode(), false)
		require.NoError(t, err)

		// Both should produce the same YAML output
		ptrYAML, err := yaml.Marshal(ptrModel.GetCore().GetRootNode())
		require.NoError(t, err)

		valueYAML, err := yaml.Marshal(valueModel.GetCore().GetRootNode())
		require.NoError(t, err)

		require.YAMLEq(t, string(ptrYAML), string(valueYAML))
		require.YAMLEq(t, "shared_key: shared_value\n", string(ptrYAML))
	})
}
