package marshaller_test

import (
	"context"
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify all fields were synced correctly
	require.Equal(t, "synced string", coreModel.StringField.Value)
	require.NotNil(t, coreModel.StringPtrField.Value)
	require.Equal(t, "synced ptr string", *coreModel.StringPtrField.Value)
	require.Equal(t, true, coreModel.BoolField.Value)
	require.NotNil(t, coreModel.BoolPtrField.Value)
	require.Equal(t, false, *coreModel.BoolPtrField.Value)
	require.Equal(t, 99, coreModel.IntField.Value)
	require.NotNil(t, coreModel.IntPtrField.Value)
	require.Equal(t, 88, *coreModel.IntPtrField.Value)
	require.Equal(t, 9.99, coreModel.Float64Field.Value)
	require.NotNil(t, coreModel.Float64PtrField.Value)
	require.Equal(t, 8.88, *coreModel.Float64PtrField.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_PrimitiveTypes_NilPointers_Success(t *testing.T) {
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify required fields were synced
	require.Equal(t, "required string", coreModel.StringField.Value)
	require.Equal(t, true, coreModel.BoolField.Value)
	require.Equal(t, 42, coreModel.IntField.Value)
	require.Equal(t, 3.14, coreModel.Float64Field.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_ComplexTypes_Success(t *testing.T) {
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify nested model was synced
	require.NotNil(t, coreModel.NestedModel.Value)
	nestedCore := coreModel.NestedModel.Value
	require.Equal(t, "nested synced", nestedCore.StringField.Value)
	require.Equal(t, true, nestedCore.BoolField.Value)
	require.Equal(t, 200, nestedCore.IntField.Value)
	require.Equal(t, 2.22, nestedCore.Float64Field.Value)

	// Verify nested model value was synced
	nestedValueCore := coreModel.NestedModelValue.Value
	require.Equal(t, "value synced", nestedValueCore.StringField.Value)
	require.Equal(t, false, nestedValueCore.BoolField.Value)
	require.Equal(t, 300, nestedValueCore.IntField.Value)
	require.Equal(t, 3.33, nestedValueCore.Float64Field.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_RequiredNilableTypes_Success(t *testing.T) {
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
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
	require.Equal(t, true, structCore.BoolField.Value)
	require.Equal(t, 500, structCore.IntField.Value)
	require.Equal(t, 5.55, structCore.Float64Field.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_RequiredPointer_Success(t *testing.T) {
	// Create a high-level model with required pointer
	highModel := tests.TestRequiredPointerHighModel{
		RequiredPtr: pointer.From("required synced ptr"),
		OptionalPtr: pointer.From("optional synced ptr"),
	}

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMapWithFields_Success(t *testing.T) {
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
	highModel.Map = sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
	highModel.Map.Set("syncKey1", dynamicVal1)
	highModel.Map.Set("syncKey2", dynamicVal2)

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify regular field
	require.Equal(t, "synced name", coreModel.NameField.Value)

	// Verify embedded map was synced
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 2, coreModel.Map.Len())
	require.True(t, coreModel.Map.Has("syncKey1"))
	require.True(t, coreModel.Map.Has("syncKey2"))

	// Verify dynamic field values
	syncedVal1, ok1 := coreModel.Map.Get("syncKey1")
	require.True(t, ok1)
	require.NotNil(t, syncedVal1)
	syncedCore1 := syncedVal1.Value
	require.Equal(t, "synced dynamic 1", syncedCore1.StringField.Value)
	require.Equal(t, true, syncedCore1.BoolField.Value)

	syncedVal2, ok2 := coreModel.Map.Get("syncKey2")
	require.True(t, ok2)
	require.NotNil(t, syncedVal2)
	syncedCore2 := syncedVal2.Value
	require.Equal(t, "synced dynamic 2", syncedCore2.StringField.Value)
	require.Equal(t, false, syncedCore2.BoolField.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_EmbeddedMap_Success(t *testing.T) {
	// Create a high-level model with embedded map
	highModel := tests.TestEmbeddedMapHighModel{}

	// Initialize the embedded map
	highModel.Map = sequencedmap.New[string, string]()
	highModel.Map.Set("syncKey1", "synced value1")
	highModel.Map.Set("syncKey2", "synced value2")
	highModel.Map.Set("syncKey3", "synced value3")

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify embedded map was synced
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 3, coreModel.Map.Len())
	require.True(t, coreModel.Map.Has("syncKey1"))
	require.True(t, coreModel.Map.Has("syncKey2"))
	require.True(t, coreModel.Map.Has("syncKey3"))

	// Verify values
	val1, ok1 := coreModel.Map.Get("syncKey1")
	require.True(t, ok1)
	require.Equal(t, "synced value1", val1.Value)

	val2, ok2 := coreModel.Map.Get("syncKey2")
	require.True(t, ok2)
	require.Equal(t, "synced value2", val2.Value)

	val3, ok3 := coreModel.Map.Get("syncKey3")
	require.True(t, ok3)
	require.Equal(t, "synced value3", val3.Value)

	// Verify the core model's RootNode contains the correct YAML
	expectedYAML := `syncKey1: synced value1
syncKey2: synced value2
syncKey3: synced value3
`

	actualYAML, err := yaml.Marshal(coreModel.GetRootNode())
	require.NoError(t, err)
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_Validation_Success(t *testing.T) {
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
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
	require.Equal(t, true, requiredStructCore.BoolField.Value)
	require.Equal(t, 600, requiredStructCore.IntField.Value)
	require.Equal(t, 6.66, requiredStructCore.Float64Field.Value)

	require.NotNil(t, coreModel.OptionalStruct.Value)
	optionalStructCore := coreModel.OptionalStruct.Value
	require.Equal(t, "synced optional nested", optionalStructCore.StringField.Value)
	require.Equal(t, false, optionalStructCore.BoolField.Value)
	require.Equal(t, 700, optionalStructCore.IntField.Value)
	require.Equal(t, 7.77, optionalStructCore.Float64Field.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_PrimitiveTypes_WithExtensions_Success(t *testing.T) {
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Get the core model for verification
	coreModel := highModel.GetCore()

	// Verify all fields were synced correctly
	require.Equal(t, "synced string", coreModel.StringField.Value)
	require.NotNil(t, coreModel.StringPtrField.Value)
	require.Equal(t, "synced ptr string", *coreModel.StringPtrField.Value)
	require.Equal(t, true, coreModel.BoolField.Value)
	require.NotNil(t, coreModel.BoolPtrField.Value)
	require.Equal(t, false, *coreModel.BoolPtrField.Value)
	require.Equal(t, 99, coreModel.IntField.Value)
	require.NotNil(t, coreModel.IntPtrField.Value)
	require.Equal(t, 88, *coreModel.IntPtrField.Value)
	require.Equal(t, 9.99, coreModel.Float64Field.Value)
	require.NotNil(t, coreModel.Float64PtrField.Value)
	require.Equal(t, 8.88, *coreModel.Float64PtrField.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_EitherValueModel_Success(t *testing.T) {
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
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
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
	require.Equal(t, expectedYAML, string(actualYAML))
}

func TestSync_TypeConversionModel_Success(t *testing.T) {
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
	highModel.Map = sequencedmap.New[tests.HTTPMethod, *tests.TestPrimitiveHighModel]()
	highModel.Map.Set(tests.HTTPMethodPost, postOp)
	highModel.Map.Set(tests.HTTPMethodGet, getOp)
	highModel.Map.Set(tests.HTTPMethodPut, putOp)

	// Initialize extensions
	highModel.Extensions = &extensions.Extensions{}
	highModel.Extensions.Init()
	highModel.Extensions.Set("x-sync", testutils.CreateStringYamlNode("sync extension", 1, 1))

	// Sync the high model to the core model
	resultNode, err := marshaller.SyncValue(context.Background(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
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
	require.Equal(t, 3, coreModel.Map.Len())
	require.True(t, coreModel.Map.Has("post"))
	require.True(t, coreModel.Map.Has("get"))
	require.True(t, coreModel.Map.Has("put"))

	// Verify operation values
	syncedPostOp, ok1 := coreModel.Map.Get("post")
	require.True(t, ok1)
	require.NotNil(t, syncedPostOp)
	syncedPostCore := syncedPostOp.Value
	require.Equal(t, "Synced POST operation", syncedPostCore.StringField.Value)
	require.Equal(t, true, syncedPostCore.BoolField.Value)
	require.Equal(t, 42, syncedPostCore.IntField.Value)
	require.Equal(t, 3.14, syncedPostCore.Float64Field.Value)

	syncedGetOp, ok2 := coreModel.Map.Get("get")
	require.True(t, ok2)
	require.NotNil(t, syncedGetOp)
	syncedGetCore := syncedGetOp.Value
	require.Equal(t, "Synced GET operation", syncedGetCore.StringField.Value)
	require.Equal(t, false, syncedGetCore.BoolField.Value)
	require.Equal(t, 100, syncedGetCore.IntField.Value)
	require.Equal(t, 1.23, syncedGetCore.Float64Field.Value)

	syncedPutOp, ok3 := coreModel.Map.Get("put")
	require.True(t, ok3)
	require.NotNil(t, syncedPutOp)
	syncedPutCore := syncedPutOp.Value
	require.Equal(t, "Synced PUT operation", syncedPutCore.StringField.Value)
	require.Equal(t, true, syncedPutCore.BoolField.Value)
	require.Equal(t, 200, syncedPutCore.IntField.Value)
	require.Equal(t, 2.34, syncedPutCore.Float64Field.Value)

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
	require.Equal(t, expectedYAML, string(actualYAML))
}
