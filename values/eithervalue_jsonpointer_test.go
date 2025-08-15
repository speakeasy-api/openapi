package values

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKeyNavigable is a test type that implements KeyNavigable
type MockKeyNavigable struct {
	Data map[string]interface{}
}

func (m *MockKeyNavigable) NavigateWithKey(key string) (any, error) {
	if value, exists := m.Data[key]; exists {
		return value, nil
	}
	return nil, jsonpointer.ErrNotFound
}

// MockIndexNavigable is a test type that implements IndexNavigable
type MockIndexNavigable struct {
	Data []interface{}
}

func (m *MockIndexNavigable) NavigateWithIndex(index int) (any, error) {
	if index < 0 || index >= len(m.Data) {
		return nil, jsonpointer.ErrNotFound
	}
	return m.Data[index], nil
}

// MockBothNavigable implements both KeyNavigable and IndexNavigable
type MockBothNavigable struct {
	MapData   map[string]interface{}
	SliceData []interface{}
}

func (m *MockBothNavigable) NavigateWithKey(key string) (any, error) {
	if value, exists := m.MapData[key]; exists {
		return value, nil
	}
	return nil, jsonpointer.ErrNotFound
}

func (m *MockBothNavigable) NavigateWithIndex(index int) (any, error) {
	if index < 0 || index >= len(m.SliceData) {
		return nil, jsonpointer.ErrNotFound
	}
	return m.SliceData[index], nil
}

func TestEitherValue_JSONPointer_LeftValue_KeyNavigation(t *testing.T) {
	t.Parallel()

	// Test with Left value that supports key navigation
	leftValue := &MockKeyNavigable{
		Data: map[string]interface{}{
			"test": "value1",
			"foo":  "bar",
		},
	}

	eitherValue := &EitherValue[MockKeyNavigable, MockKeyNavigable, string, string]{
		Left: leftValue,
	}

	// Test successful navigation using JSON pointer
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/test"))
	require.NoError(t, err)
	assert.Equal(t, "value1", result)

	// Test key not found
	result, err = jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/nonexistent"))
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestEitherValue_JSONPointer_RightValue_KeyNavigation(t *testing.T) {
	t.Parallel()

	// Test with Right value that supports key navigation
	rightValue := &MockKeyNavigable{
		Data: map[string]interface{}{
			"right": "value2",
		},
	}

	eitherValue := &EitherValue[string, string, MockKeyNavigable, MockKeyNavigable]{
		Right: rightValue,
	}

	// Test successful navigation using JSON pointer
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/right"))
	require.NoError(t, err)
	assert.Equal(t, "value2", result)
}

func TestEitherValue_JSONPointer_UnsupportedType(t *testing.T) {
	t.Parallel()

	// Test with Left value that doesn't support key navigation
	eitherValue := &EitherValue[string, string, string, string]{
		Left: stringPtr("simple string"),
	}

	// Try to navigate with key (should fail because string doesn't support navigation)
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/somekey"))
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestEitherValue_JSONPointer_LeftValue_IndexNavigation(t *testing.T) {
	t.Parallel()

	// Test with Left value that supports index navigation
	leftValue := &MockIndexNavigable{
		Data: []interface{}{"item0", "item1", "item2"},
	}

	eitherValue := &EitherValue[MockIndexNavigable, MockIndexNavigable, string, string]{
		Left: leftValue,
	}

	// Test successful navigation using JSON pointer
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/1"))
	require.NoError(t, err)
	assert.Equal(t, "item1", result)

	// Test index out of range
	result, err = jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/10"))
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestEitherValue_JSONPointer_RightValue_IndexNavigation(t *testing.T) {
	t.Parallel()

	// Test with Right value that supports index navigation
	rightValue := &MockIndexNavigable{
		Data: []interface{}{"right0", "right1"},
	}

	eitherValue := &EitherValue[string, string, MockIndexNavigable, MockIndexNavigable]{
		Right: rightValue,
	}

	// Test successful navigation using JSON pointer
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/0"))
	require.NoError(t, err)
	assert.Equal(t, "right0", result)
}

func TestEitherValue_GetNavigableNode_NoValueSet(t *testing.T) {
	t.Parallel()

	// Test with neither Left nor Right set
	eitherValue := &EitherValue[string, string, string, string]{}

	// Test GetNavigableNode directly
	result, err := eitherValue.GetNavigableNode()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no value set")
	assert.Nil(t, result)
}

func TestEitherValue_GetNavigableNode_LeftValue(t *testing.T) {
	t.Parallel()

	// Test GetNavigableNode with Left value
	leftValue := &MockKeyNavigable{
		Data: map[string]interface{}{"test": "value"},
	}

	eitherValue := &EitherValue[MockKeyNavigable, MockKeyNavigable, string, string]{
		Left: leftValue,
	}

	result, err := eitherValue.GetNavigableNode()
	require.NoError(t, err)
	assert.Equal(t, leftValue, result)
}

func TestEitherValue_GetNavigableNode_RightValue(t *testing.T) {
	t.Parallel()

	// Test GetNavigableNode with Right value
	rightValue := &MockIndexNavigable{
		Data: []interface{}{"item"},
	}

	eitherValue := &EitherValue[string, string, MockIndexNavigable, MockIndexNavigable]{
		Right: rightValue,
	}

	result, err := eitherValue.GetNavigableNode()
	require.NoError(t, err)
	assert.Equal(t, rightValue, result)
}

func TestEitherValue_JSONPointer_BothNavigationTypes(t *testing.T) {
	t.Parallel()

	// Test with value that supports both key and index navigation
	bothValue := &MockBothNavigable{
		MapData:   map[string]interface{}{"key1": "mapvalue"},
		SliceData: []interface{}{"slicevalue"},
	}

	eitherValue := &EitherValue[MockBothNavigable, MockBothNavigable, string, string]{
		Left: bothValue,
	}

	// Test key navigation using JSON pointer
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/key1"))
	require.NoError(t, err)
	assert.Equal(t, "mapvalue", result)

	// Test index navigation using JSON pointer
	result, err = jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/0"))
	require.NoError(t, err)
	assert.Equal(t, "slicevalue", result)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
