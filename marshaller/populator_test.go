package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/stretchr/testify/require"
)

func TestPopulation_PrimitiveTypes_Success(t *testing.T) {
	t.Parallel()

	yml := `
stringField: "test string"
stringPtrField: "test ptr string"
boolField: true
boolPtrField: false
intField: 42
intPtrField: 24
float64Field: 3.14
float64PtrField: 2.71
x-custom: "extension value"
`

	// First unmarshal to core model
	var coreModel core.TestPrimitiveModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestPrimitiveHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify all fields were populated correctly
	require.Equal(t, "test string", highModel.StringField)
	require.NotNil(t, highModel.StringPtrField)
	require.Equal(t, "test ptr string", *highModel.StringPtrField)
	require.True(t, highModel.BoolField)
	require.NotNil(t, highModel.BoolPtrField)
	require.False(t, *highModel.BoolPtrField)
	require.Equal(t, 42, highModel.IntField)
	require.NotNil(t, highModel.IntPtrField)
	require.Equal(t, 24, *highModel.IntPtrField)
	require.InDelta(t, 3.14, highModel.Float64Field, 0.001)
	require.NotNil(t, highModel.Float64PtrField)
	require.InDelta(t, 2.71, *highModel.Float64PtrField, 0.001)

	// Verify extensions were populated
	require.NotNil(t, highModel.Extensions)
}

func TestPopulation_PrimitiveTypes_PartialData(t *testing.T) {
	t.Parallel()

	yml := `
stringField: "required only"
boolField: true
intField: 42
float64Field: 3.14
`

	// First unmarshal to core model
	var coreModel core.TestPrimitiveModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestPrimitiveHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify required fields were populated
	require.Equal(t, "required only", highModel.StringField)
	require.True(t, highModel.BoolField)
	require.Equal(t, 42, highModel.IntField)
	require.InDelta(t, 3.14, highModel.Float64Field, 0.001)

	// Verify optional fields are nil/zero
	require.Nil(t, highModel.StringPtrField)
	require.Nil(t, highModel.BoolPtrField)
	require.Nil(t, highModel.IntPtrField)
	require.Nil(t, highModel.Float64PtrField)
}

func TestPopulation_ComplexTypes_Success(t *testing.T) {
	t.Parallel()

	yml := `
nestedModelValue:
  stringField: "value model"
  boolField: false
  intField: 200
  float64Field: 4.56
nestedModel:
  stringField: "nested value"
  boolField: true
  intField: 100
  float64Field: 1.23
arrayField:
  - "item1"
  - "item2"
  - "item3"
nodeArrayField:
  - "node1"
  - "node2"
mapField:
  key1: "value1"
  key2: "value2"
eitherModelOrPrimitive: 789
valueField: "some value"
valuesField:
  - "some value"
  - "some other value"
  - "yet another value"
x-extension: "ext value"
`

	// First unmarshal to core model
	var coreModel core.TestComplexModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestComplexHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify nested model population
	require.NotNil(t, highModel.NestedModel)
	require.Equal(t, "nested value", highModel.NestedModel.StringField)
	require.True(t, highModel.NestedModel.BoolField)
	require.Equal(t, 100, highModel.NestedModel.IntField)
	require.InDelta(t, 1.23, highModel.NestedModel.Float64Field, 0.001)

	// Verify nested model value population
	require.Equal(t, "value model", highModel.NestedModelValue.StringField)
	require.False(t, highModel.NestedModelValue.BoolField)
	require.Equal(t, 200, highModel.NestedModelValue.IntField)
	require.InDelta(t, 4.56, highModel.NestedModelValue.Float64Field, 0.001)

	// Verify array field population
	require.Len(t, highModel.ArrayField, 3)
	require.Equal(t, "item1", highModel.ArrayField[0])
	require.Equal(t, "item2", highModel.ArrayField[1])
	require.Equal(t, "item3", highModel.ArrayField[2])

	// Verify node array field population
	require.Len(t, highModel.NodeArrayField, 2)
	require.Equal(t, "node1", highModel.NodeArrayField[0])
	require.Equal(t, "node2", highModel.NodeArrayField[1])

	// Verify map field population
	require.NotNil(t, highModel.MapPrimitiveField)
	val1, ok1 := highModel.MapPrimitiveField.Get("key1")
	require.True(t, ok1)
	require.Equal(t, "value1", val1)
	val2, ok2 := highModel.MapPrimitiveField.Get("key2")
	require.True(t, ok2)
	require.Equal(t, "value2", val2)

	// Verify value field population
	require.Equal(t, "some value", highModel.ValueField.Value)

	// Verify values field population
	require.Len(t, highModel.ValuesField, 3)
	require.Equal(t, "some value", highModel.ValuesField[0].Value)
	require.Equal(t, "some other value", highModel.ValuesField[1].Value)
	require.Equal(t, "yet another value", highModel.ValuesField[2].Value)

	// Verify extensions were populated
	require.NotNil(t, highModel.Extensions)
}

func TestPopulation_RequiredNilableTypes_Success(t *testing.T) {
	t.Parallel()

	yml := `
requiredPtr: "required pointer value"
requiredSlice: ["item1", "item2"]
requiredMap:
  key1: "value1"
  key2: "value2"
requiredStruct:
  stringField: "nested required"
  boolField: true
  intField: 42
  float64Field: 3.14
requiredEither: "either string value"
requiredRawNode: "raw node value"
`

	// First unmarshal to core model
	var coreModel core.TestRequiredNilableModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestRequiredNilableHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify required fields were populated
	require.NotNil(t, highModel.RequiredPtr)
	require.Equal(t, "required pointer value", *highModel.RequiredPtr)

	require.Len(t, highModel.RequiredSlice, 2)
	require.Equal(t, "item1", highModel.RequiredSlice[0])
	require.Equal(t, "item2", highModel.RequiredSlice[1])

	require.NotNil(t, highModel.RequiredMap)
	val1, ok1 := highModel.RequiredMap.Get("key1")
	require.True(t, ok1)
	require.Equal(t, "value1", val1)
	val2, ok2 := highModel.RequiredMap.Get("key2")
	require.True(t, ok2)
	require.Equal(t, "value2", val2)

	require.NotNil(t, highModel.RequiredStruct)
	require.Equal(t, "nested required", highModel.RequiredStruct.StringField)
	require.True(t, highModel.RequiredStruct.BoolField)
	require.Equal(t, 42, highModel.RequiredStruct.IntField)
	require.InDelta(t, 3.14, highModel.RequiredStruct.Float64Field, 0.001)

	// Verify either field was populated
	require.NotNil(t, highModel.RequiredEither)

	// Verify optional fields are nil
	require.Nil(t, highModel.OptionalPtr)
	require.Nil(t, highModel.OptionalSlice)
	require.Nil(t, highModel.OptionalMap)
	require.Nil(t, highModel.OptionalStruct)
}

func TestPopulation_RequiredPointer_Success(t *testing.T) {
	t.Parallel()

	yml := `
requiredPtr: "required pointer value"
optionalPtr: "optional pointer value"
`

	// First unmarshal to core model
	var coreModel core.TestRequiredPointerModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestRequiredPointerHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify required pointer field
	require.NotNil(t, highModel.RequiredPtr)
	require.Equal(t, "required pointer value", *highModel.RequiredPtr)

	// Verify optional pointer field
	require.NotNil(t, highModel.OptionalPtr)
	require.Equal(t, "optional pointer value", *highModel.OptionalPtr)
}

func TestPopulation_NullPointerFields_Success(t *testing.T) {
	t.Parallel()

	yml := `
stringField: "test"
boolField: true
intField: 42
float64Field: 3.14
stringPtrField: null
boolPtrField: null
intPtrField: null
float64PtrField: null
`

	// First unmarshal to core model
	var coreModel core.TestPrimitiveModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestPrimitiveHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify required fields are populated
	require.Equal(t, "test", highModel.StringField)
	require.True(t, highModel.BoolField)
	require.Equal(t, 42, highModel.IntField)
	require.InDelta(t, 3.14, highModel.Float64Field, 0.001)

	// Verify null pointer fields are still nil in high model
	require.Nil(t, highModel.StringPtrField)
	require.Nil(t, highModel.BoolPtrField)
	require.Nil(t, highModel.IntPtrField)
	require.Nil(t, highModel.Float64PtrField)
}

func TestPopulation_EmbeddedMapWithFields_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: "test name"
dynamicKey1:
  stringField: "dynamic value 1"
  boolField: true
  intField: 100
  float64Field: 1.23
dynamicKey2:
  stringField: "dynamic value 2"
  boolField: false
  intField: 42
  float64Field: 4.56
x-extension: "ext value"
`

	// First unmarshal to core model
	var coreModel core.TestEmbeddedMapWithFieldsModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestEmbeddedMapWithFieldsHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Debug: Check if core model has embedded map populated
	t.Logf("Core model embedded map is initialized: %v", coreModel.IsInitialized())
	if coreModel.IsInitialized() {
		t.Logf("Core model embedded map length: %d", coreModel.Len())
	}

	// Verify regular field
	require.Equal(t, "test name", highModel.NameField)

	// Verify dynamic fields were populated
	require.NotNil(t, highModel.Map)
	require.True(t, highModel.Has("dynamicKey1"))
	require.True(t, highModel.Has("dynamicKey2"))

	// Verify dynamic field values
	dynamicVal1, ok1 := highModel.Get("dynamicKey1")
	require.True(t, ok1)
	require.NotNil(t, dynamicVal1)
	require.Equal(t, "dynamic value 1", dynamicVal1.StringField)
	require.True(t, dynamicVal1.BoolField)

	dynamicVal2, ok2 := highModel.Get("dynamicKey2")
	require.True(t, ok2)
	require.NotNil(t, dynamicVal2)
	require.Equal(t, "dynamic value 2", dynamicVal2.StringField)
	require.False(t, dynamicVal2.BoolField)

	// Verify extensions were populated
	require.NotNil(t, highModel.Extensions)
}

func TestPopulation_EmbeddedMap_Success(t *testing.T) {
	t.Parallel()

	yml := `
dynamicKey1: "value1"
dynamicKey2: "value2"
dynamicKey3: "value3"
`

	// First unmarshal to core model
	var coreModel core.TestEmbeddedMapModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestEmbeddedMapHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify embedded map was populated
	require.NotNil(t, highModel.Map)
	require.Equal(t, 3, highModel.Len())
	require.True(t, highModel.Has("dynamicKey1"))
	require.True(t, highModel.Has("dynamicKey2"))
	require.True(t, highModel.Has("dynamicKey3"))

	// Verify values
	val1, ok1 := highModel.Get("dynamicKey1")
	require.True(t, ok1)
	require.Equal(t, "value1", val1)

	val2, ok2 := highModel.Get("dynamicKey2")
	require.True(t, ok2)
	require.Equal(t, "value2", val2)

	val3, ok3 := highModel.Get("dynamicKey3")
	require.True(t, ok3)
	require.Equal(t, "value3", val3)
}

func TestPopulation_Validation_Success(t *testing.T) {
	t.Parallel()

	yml := `
requiredField: "required value"
optionalField: "optional value"
requiredArray: ["item1", "item2"]
optionalArray: ["opt1", "opt2"]
requiredStruct:
  stringField: "nested required"
  boolField: true
  intField: 42
  float64Field: 3.14
optionalStruct:
  stringField: "nested optional"
  boolField: false
  intField: 24
  float64Field: 2.71
x-extension: "ext value"
`

	// First unmarshal to core model
	var coreModel core.TestValidationModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population to high-level model
	var highModel tests.TestValidationHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify required fields
	require.Equal(t, "required value", highModel.RequiredField)
	require.NotNil(t, highModel.OptionalField)
	require.Equal(t, "optional value", *highModel.OptionalField)

	// Verify arrays
	require.Len(t, highModel.RequiredArray, 2)
	require.Equal(t, "item1", highModel.RequiredArray[0])
	require.Equal(t, "item2", highModel.RequiredArray[1])

	require.Len(t, highModel.OptionalArray, 2)
	require.Equal(t, "opt1", highModel.OptionalArray[0])
	require.Equal(t, "opt2", highModel.OptionalArray[1])

	// Verify nested structs
	require.NotNil(t, highModel.RequiredStruct)
	require.Equal(t, "nested required", highModel.RequiredStruct.StringField)
	require.True(t, highModel.RequiredStruct.BoolField)
	require.Equal(t, 42, highModel.RequiredStruct.IntField)
	require.InDelta(t, 3.14, highModel.RequiredStruct.Float64Field, 0.001)

	require.NotNil(t, highModel.OptionalStruct)
	require.Equal(t, "nested optional", highModel.OptionalStruct.StringField)
	require.False(t, highModel.OptionalStruct.BoolField)
	require.Equal(t, 24, highModel.OptionalStruct.IntField)
	require.InDelta(t, 2.71, highModel.OptionalStruct.Float64Field, 0.001)

	// Verify extensions
	require.NotNil(t, highModel.Extensions)
}

func TestPopulation_TypeConversion_Error(t *testing.T) {
	t.Parallel()

	// This test reproduces the issue from openapi.Callback where:
	// - Core model uses string keys (like "post", "get")
	// - High-level model expects HTTPMethod keys
	// - Population should fail with type conversion error

	yml := `
httpMethodField: "post"
post:
  stringField: "POST operation"
  boolField: true
  intField: 42
  float64Field: 3.14
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
put:
  stringField: "PUT operation"
  boolField: true
  intField: 200
  float64Field: 2.34
`

	// First unmarshal to core model (this should work fine)
	var coreModel core.TestTypeConversionCoreModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Verify core model populated correctly with string keys
	require.NotNil(t, coreModel.Map)
	require.Equal(t, 3, coreModel.Len())

	// Verify HTTPMethod field was populated
	require.True(t, coreModel.HTTPMethodField.Present)
	require.NotNil(t, coreModel.HTTPMethodField.Value)
	require.Equal(t, "post", *coreModel.HTTPMethodField.Value)

	postOp, exists := coreModel.Get("post")
	require.True(t, exists)
	require.Equal(t, "POST operation", postOp.Value.StringField.Value)

	getOp, exists := coreModel.Get("get")
	require.True(t, exists)
	require.Equal(t, "GET operation", getOp.Value.StringField.Value)

	putOp, exists := coreModel.Get("put")
	require.True(t, exists)
	require.Equal(t, "PUT operation", putOp.Value.StringField.Value)

	// Now try to populate high-level model with HTTPMethod keys
	// This should now succeed with our fix!
	var highModel tests.TestTypeConversionHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)

	// This should now succeed with key type conversion
	require.NoError(t, err, "Population should succeed with key type conversion")

	// Verify the HTTPMethod field conversion worked
	require.NotNil(t, highModel.HTTPMethodField)
	require.Equal(t, tests.HTTPMethodPost, *highModel.HTTPMethodField)

	// Verify the embedded map was populated correctly with converted keys
	require.NotNil(t, highModel.Map)
	require.Equal(t, 3, highModel.Len())

	// Verify POST operation with HTTPMethod key
	postOpHigh, exists := highModel.Get(tests.HTTPMethodPost)
	require.True(t, exists, "POST operation should exist with HTTPMethod key")
	require.NotNil(t, postOpHigh)
	require.Equal(t, "POST operation", postOpHigh.StringField)

	// Verify GET operation with HTTPMethod key
	getOpHigh, exists := highModel.Get(tests.HTTPMethodGet)
	require.True(t, exists, "GET operation should exist with HTTPMethod key")
	require.NotNil(t, getOpHigh)
	require.Equal(t, "GET operation", getOpHigh.StringField)

	// Verify PUT operation with HTTPMethod key
	putOpHigh, exists := highModel.Get(tests.HTTPMethodPut)
	require.True(t, exists, "PUT operation should exist with HTTPMethod key")
	require.NotNil(t, putOpHigh)
	require.Equal(t, "PUT operation", putOpHigh.StringField)
}

func TestPopulation_HTTPMethodField_Success(t *testing.T) {
	t.Parallel()

	// Test if individual field conversion from string to HTTPMethod works
	yml := `
httpMethodField: "post"
`

	// First unmarshal to core model (string field)
	var coreModel core.TestTypeConversionCoreModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Verify core model has string value
	require.True(t, coreModel.HTTPMethodField.Present)
	require.NotNil(t, coreModel.HTTPMethodField.Value)
	require.Equal(t, "post", *coreModel.HTTPMethodField.Value)

	// Now try to populate high-level model (HTTPMethod field)
	var highModel tests.TestTypeConversionHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)

	if err != nil {
		t.Logf("Field conversion error: %v", err)
		// If this fails, it means the marshaller doesn't handle string -> HTTPMethod conversion
		// even for individual fields, so we need to implement that first
		require.NoError(t, err, "Field-level type conversion should work")
	} else {
		// If this succeeds, verify the conversion worked
		require.NotNil(t, highModel.HTTPMethodField)
		require.Equal(t, tests.HTTPMethodPost, *highModel.HTTPMethodField)
		t.Logf("Field conversion successful: %v -> %v", "post", *highModel.HTTPMethodField)
	}
}

// TestPopulation_WithContext_Success tests the new PopulateWithContext function
// that supports passing owning document context through the population chain.
func TestPopulation_WithContext_Success(t *testing.T) {
	t.Parallel()

	yml := `
stringField: "test string"
boolField: true
intField: 42
float64Field: 3.14
`

	// First unmarshal to core model
	var coreModel core.TestPrimitiveModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Create a context with an owning document
	owningDoc := "test-owning-document"
	ctx := &marshaller.PopulationContext{
		Parent:         nil,
		OwningDocument: owningDoc,
	}

	// Test population with context
	var highModel tests.TestPrimitiveHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, ctx)
	require.NoError(t, err)

	// Verify fields were populated correctly
	require.Equal(t, "test string", highModel.StringField)
	require.True(t, highModel.BoolField)
	require.Equal(t, 42, highModel.IntField)
	require.InDelta(t, 3.14, highModel.Float64Field, 0.001)
}

// TestPopulation_WithContext_NilContext_Success tests that PopulateWithContext
// works correctly when context is nil (backward compatible).
func TestPopulation_WithContext_NilContext_Success(t *testing.T) {
	t.Parallel()

	yml := `
stringField: "test string"
boolField: true
intField: 42
float64Field: 3.14
`

	// First unmarshal to core model
	var coreModel core.TestPrimitiveModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Test population with nil context
	var highModel tests.TestPrimitiveHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, nil)
	require.NoError(t, err)

	// Verify fields were populated correctly
	require.Equal(t, "test string", highModel.StringField)
	require.True(t, highModel.BoolField)
	require.Equal(t, 42, highModel.IntField)
	require.InDelta(t, 3.14, highModel.Float64Field, 0.001)
}

// TestPopulation_WithContext_ComplexTypes_Success tests PopulateWithContext
// with complex nested types to ensure context is passed through correctly.
func TestPopulation_WithContext_ComplexTypes_Success(t *testing.T) {
	t.Parallel()

	yml := `
nestedModelValue:
  stringField: "value model"
  boolField: false
  intField: 200
  float64Field: 4.56
nestedModel:
  stringField: "nested value"
  boolField: true
  intField: 100
  float64Field: 1.23
arrayField:
  - "item1"
  - "item2"
eitherModelOrPrimitive: 789
`

	// First unmarshal to core model
	var coreModel core.TestComplexModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &coreModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, coreModel.Valid)

	// Create a context with an owning document
	owningDoc := map[string]string{"type": "test-document"}
	ctx := &marshaller.PopulationContext{
		Parent:         nil,
		OwningDocument: owningDoc,
	}

	// Test population with context
	var highModel tests.TestComplexHighModel
	err = marshaller.PopulateWithContext(coreModel, &highModel, ctx)
	require.NoError(t, err)

	// Verify nested model population
	require.NotNil(t, highModel.NestedModel)
	require.Equal(t, "nested value", highModel.NestedModel.StringField)
	require.True(t, highModel.NestedModel.BoolField)

	// Verify array field population
	require.Len(t, highModel.ArrayField, 2)
	require.Equal(t, "item1", highModel.ArrayField[0])
	require.Equal(t, "item2", highModel.ArrayField[1])
}

// TestPopulationContext_Struct tests that PopulationContext struct works correctly.
func TestPopulationContext_Struct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		parent         any
		owningDocument any
	}{
		{
			name:           "both nil",
			parent:         nil,
			owningDocument: nil,
		},
		{
			name:           "parent only",
			parent:         "parent-value",
			owningDocument: nil,
		},
		{
			name:           "owning document only",
			parent:         nil,
			owningDocument: "owning-doc-value",
		},
		{
			name:           "both set",
			parent:         "parent-value",
			owningDocument: "owning-doc-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &marshaller.PopulationContext{
				Parent:         tt.parent,
				OwningDocument: tt.owningDocument,
			}

			require.Equal(t, tt.parent, ctx.Parent)
			require.Equal(t, tt.owningDocument, ctx.OwningDocument)
		})
	}
}
