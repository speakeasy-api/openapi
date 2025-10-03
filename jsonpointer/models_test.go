package jsonpointer

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigateModel_BasicFields(t *testing.T) {
	t.Parallel()

	// Create a test model with some data
	model := &tests.TestPrimitiveHighModel{
		StringField:  "test value",
		BoolField:    true,
		IntField:     42,
		Float64Field: 3.14,
	}

	// Test navigating to stringField
	target, err := GetTarget(model, "/stringField")
	require.NoError(t, err)
	assert.Equal(t, "test value", target)

	// Test navigating to boolField
	target, err = GetTarget(model, "/boolField")
	require.NoError(t, err)
	assert.Equal(t, true, target)

	// Test navigating to intField
	target, err = GetTarget(model, "/intField")
	require.NoError(t, err)
	assert.Equal(t, 42, target)

	// Test navigating to float64Field
	target, err = GetTarget(model, "/float64Field")
	require.NoError(t, err)
	assert.InDelta(t, 3.14, target, 0.001)
}

func TestNavigateModel_PointerFields(t *testing.T) {
	t.Parallel()

	stringPtr := "pointer value"
	boolPtr := false
	intPtr := 24
	floatPtr := 2.71

	model := &tests.TestPrimitiveHighModel{
		StringField:     "test",
		StringPtrField:  &stringPtr,
		BoolField:       true,
		BoolPtrField:    &boolPtr,
		IntField:        42,
		IntPtrField:     &intPtr,
		Float64Field:    3.14,
		Float64PtrField: &floatPtr,
	}

	// Test navigating to pointer fields
	target, err := GetTarget(model, "/stringPtrField")
	require.NoError(t, err)
	assert.Equal(t, &stringPtr, target)

	target, err = GetTarget(model, "/boolPtrField")
	require.NoError(t, err)
	assert.Equal(t, &boolPtr, target)

	target, err = GetTarget(model, "/intPtrField")
	require.NoError(t, err)
	assert.Equal(t, &intPtr, target)

	target, err = GetTarget(model, "/float64PtrField")
	require.NoError(t, err)
	assert.Equal(t, &floatPtr, target)
}

func TestNavigateModel_NestedModel(t *testing.T) {
	t.Parallel()

	nestedModel := &tests.TestPrimitiveHighModel{
		StringField:  "nested value",
		BoolField:    false,
		IntField:     100,
		Float64Field: 1.23,
	}

	model := &tests.TestComplexHighModel{
		NestedModel: nestedModel,
		NestedModelValue: tests.TestPrimitiveHighModel{
			StringField:  "value model",
			BoolField:    true,
			IntField:     200,
			Float64Field: 4.56,
		},
	}

	// Test navigating to nested model field
	target, err := GetTarget(model, "/nestedModel/stringField")
	require.NoError(t, err)
	assert.Equal(t, "nested value", target)

	// Test navigating to nested model value field
	target, err = GetTarget(model, "/nestedModelValue/intField")
	require.NoError(t, err)
	assert.Equal(t, 200, target)
}

func TestNavigateModel_ArrayField(t *testing.T) {
	t.Parallel()

	model := &tests.TestComplexHighModel{
		ArrayField:     []string{"item1", "item2", "item3"},
		NodeArrayField: []string{"node1", "node2"},
	}

	// Test navigating to array elements
	target, err := GetTarget(model, "/arrayField/0")
	require.NoError(t, err)
	assert.Equal(t, "item1", target)

	target, err = GetTarget(model, "/arrayField/2")
	require.NoError(t, err)
	assert.Equal(t, "item3", target)

	target, err = GetTarget(model, "/nodeArrayField/1")
	require.NoError(t, err)
	assert.Equal(t, "node2", target)
}

func TestNavigateModel_NotFound(t *testing.T) {
	t.Parallel()

	model := &tests.TestPrimitiveHighModel{
		StringField: "test",
	}

	// Test navigating to non-existent field
	_, err := GetTarget(model, "/nonExistentField")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test navigating to field that doesn't exist in core model
	_, err = GetTarget(model, "/invalidField")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestNavigateModel_IndexNavigationError(t *testing.T) {
	t.Parallel()

	model := &tests.TestPrimitiveHighModel{
		StringField: "test",
	}

	// Test that index navigation on models returns an error when key "0" doesn't exist
	_, err := GetTarget(model, "/0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found -- key 0 not found in core model")
}

func TestNavigateModel_EmbeddedMap(t *testing.T) {
	t.Parallel()

	t.Run("SimpleEmbeddedMap", func(t *testing.T) {
		t.Parallel()
		// Create a simple embedded map model
		embeddedMap := &tests.TestEmbeddedMapHighModel{}
		embeddedMap.Map = sequencedmap.New[string, string]()
		embeddedMap.Set("key1", "value1")
		embeddedMap.Set("key2", "value2")

		// Test navigating to embedded map keys
		target, err := GetTarget(embeddedMap, "/key1")
		require.NoError(t, err)
		assert.Equal(t, "value1", target)

		target, err = GetTarget(embeddedMap, "/key2")
		require.NoError(t, err)
		assert.Equal(t, "value2", target)
	})

	t.Run("EmbeddedMapWithFields", func(t *testing.T) {
		t.Parallel()
		// Create nested models for the embedded map
		nestedModel1 := &tests.TestPrimitiveHighModel{
			StringField: "nested1",
			IntField:    100,
		}
		nestedModel2 := &tests.TestPrimitiveHighModel{
			StringField: "nested2",
			IntField:    200,
		}

		// Create embedded map with fields model
		embeddedMapWithFields := &tests.TestEmbeddedMapWithFieldsHighModel{
			NameField: "test name",
		}
		embeddedMapWithFields.Map = sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
		embeddedMapWithFields.Set("model1", nestedModel1)
		embeddedMapWithFields.Set("model2", nestedModel2)

		// Test navigating to regular fields
		target, err := GetTarget(embeddedMapWithFields, "/name")
		require.NoError(t, err)
		assert.Equal(t, "test name", target)

		// Test navigating to embedded map keys
		target, err = GetTarget(embeddedMapWithFields, "/model1")
		require.NoError(t, err)
		assert.Equal(t, nestedModel1, target)

		target, err = GetTarget(embeddedMapWithFields, "/model2")
		require.NoError(t, err)
		assert.Equal(t, nestedModel2, target)

		// Test navigating through embedded map to nested model fields
		target, err = GetTarget(embeddedMapWithFields, "/model1/stringField")
		require.NoError(t, err)
		assert.Equal(t, "nested1", target)

		target, err = GetTarget(embeddedMapWithFields, "/model2/intField")
		require.NoError(t, err)
		assert.Equal(t, 200, target)
	})

	t.Run("EmbeddedMapNotFound", func(t *testing.T) {
		t.Parallel()
		embeddedMap := &tests.TestEmbeddedMapHighModel{}
		embeddedMap.Map = sequencedmap.New[string, string]()
		embeddedMap.Set("existing", "value")

		// Test navigating to non-existent key in embedded map
		_, err := GetTarget(embeddedMap, "/nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestNavigateModel_EmbeddedMapEscapedKeys(t *testing.T) {
	t.Parallel()

	t.Run("EmbeddedMapWithEscapedKeys", func(t *testing.T) {
		t.Parallel()
		// Create a test that mimics OpenAPI paths structure
		// This reproduces the issue with escaped JSON pointer paths like /paths/~1users~1{userId}
		embeddedMap := &tests.TestEmbeddedMapHighModel{}
		embeddedMap.Map = sequencedmap.New[string, string]()

		// Set keys that contain special characters like OpenAPI paths
		embeddedMap.Set("/users/{userId}", "path-item-1")
		embeddedMap.Set("/users", "path-item-2")
		embeddedMap.Set("/api/v1/data", "path-item-3")

		// Test navigating using escaped JSON pointer syntax
		// This should work but currently fails
		target, err := GetTarget(embeddedMap, "/~1users~1{userId}")
		require.NoError(t, err, "Should be able to navigate to escaped path key")
		assert.Equal(t, "path-item-1", target)

		// Test navigating to simpler escaped path
		target, err = GetTarget(embeddedMap, "/~1users")
		require.NoError(t, err, "Should be able to navigate to escaped path key")
		assert.Equal(t, "path-item-2", target)

		// Test navigating to path with multiple slashes
		target, err = GetTarget(embeddedMap, "/~1api~1v1~1data")
		require.NoError(t, err, "Should be able to navigate to complex escaped path key")
		assert.Equal(t, "path-item-3", target)
	})
}

func TestNavigateModel_EmbeddedMapComparison_PointerVsValue(t *testing.T) {
	t.Parallel()

	t.Run("PointerEmbeddedMapNavigation", func(t *testing.T) {
		t.Parallel()
		// Create a model with pointer embedded map
		model := &tests.TestEmbeddedMapPointerHighModel{}
		model.Map = sequencedmap.New[string, string]()
		model.Set("ptrKey1", "pointer value1")
		model.Set("ptrKey2", "pointer value2")

		// Test navigating to embedded map keys
		target, err := GetTarget(model, "/ptrKey1")
		require.NoError(t, err)
		assert.Equal(t, "pointer value1", target)

		target, err = GetTarget(model, "/ptrKey2")
		require.NoError(t, err)
		assert.Equal(t, "pointer value2", target)
	})

	t.Run("ValueEmbeddedMapNavigation", func(t *testing.T) {
		t.Parallel()
		// Create a model with value embedded map
		model := &tests.TestEmbeddedMapHighModel{}
		model.Map = sequencedmap.New[string, string]()
		model.Set("valKey1", "value value1")
		model.Set("valKey2", "value value2")

		// Test navigating to embedded map keys
		target, err := GetTarget(model, "/valKey1")
		require.NoError(t, err)
		assert.Equal(t, "value value1", target)

		target, err = GetTarget(model, "/valKey2")
		require.NoError(t, err)
		assert.Equal(t, "value value2", target)
	})

	t.Run("BothTypesWorkIdentically", func(t *testing.T) {
		t.Parallel()
		// Create models with same data but different embed types
		ptrModel := &tests.TestEmbeddedMapPointerHighModel{}
		ptrModel.Map = sequencedmap.New[string, string]()
		ptrModel.Set("sharedKey", "shared value")

		valueModel := &tests.TestEmbeddedMapHighModel{}
		valueModel.Map = sequencedmap.New[string, string]()
		valueModel.Set("sharedKey", "shared value")

		// Both should navigate to the same result
		ptrTarget, err := GetTarget(ptrModel, "/sharedKey")
		require.NoError(t, err)

		valueTarget, err := GetTarget(valueModel, "/sharedKey")
		require.NoError(t, err)

		assert.Equal(t, ptrTarget, valueTarget, "Both pointer and value embedded maps should navigate to same result")
		assert.Equal(t, "shared value", ptrTarget)
	})

	t.Run("EmbeddedMapWithFieldsComparison", func(t *testing.T) {
		t.Parallel()
		// Create nested models for the embedded maps
		nestedModel := &tests.TestPrimitiveHighModel{
			StringField: "nested test",
			IntField:    42,
		}

		// Test pointer embedded map with fields
		ptrModel := &tests.TestEmbeddedMapWithFieldsPointerHighModel{
			NameField: "pointer test name",
		}
		ptrModel.Map = sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
		ptrModel.Set("nested", nestedModel)

		// Test value embedded map with fields
		valueModel := &tests.TestEmbeddedMapWithFieldsHighModel{
			NameField: "value test name",
		}
		valueModel.Map = sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
		valueModel.Set("nested", nestedModel)

		// Test navigating to regular fields
		ptrName, err := GetTarget(ptrModel, "/name")
		require.NoError(t, err)
		assert.Equal(t, "pointer test name", ptrName)

		valueName, err := GetTarget(valueModel, "/name")
		require.NoError(t, err)
		assert.Equal(t, "value test name", valueName)

		// Test navigating to embedded map keys
		ptrNested, err := GetTarget(ptrModel, "/nested")
		require.NoError(t, err)
		assert.Equal(t, nestedModel, ptrNested)

		valueNested, err := GetTarget(valueModel, "/nested")
		require.NoError(t, err)
		assert.Equal(t, nestedModel, valueNested)

		// Test navigating through embedded map to nested model fields
		ptrNestedField, err := GetTarget(ptrModel, "/nested/stringField")
		require.NoError(t, err)
		assert.Equal(t, "nested test", ptrNestedField)

		valueNestedField, err := GetTarget(valueModel, "/nested/stringField")
		require.NoError(t, err)
		assert.Equal(t, "nested test", valueNestedField)
	})
}
