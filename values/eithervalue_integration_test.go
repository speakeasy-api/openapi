package values

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEitherValue_JSONPointer_Integration(t *testing.T) {
	// Create a complex structure with EitherValue that supports navigation
	leftValue := &MockBothNavigable{
		MapData: map[string]interface{}{
			"users": &MockIndexNavigable{
				Data: []interface{}{
					map[string]interface{}{"name": "Alice", "id": 1},
					map[string]interface{}{"name": "Bob", "id": 2},
				},
			},
			"config": map[string]interface{}{
				"version": "1.0",
				"debug":   true,
			},
		},
		SliceData: []interface{}{
			"item0",
			map[string]interface{}{"nested": "value"},
		},
	}

	eitherValue := &EitherValue[MockBothNavigable, MockBothNavigable, string, string]{
		Left: leftValue,
	}

	// Test JSON pointer navigation through the EitherValue
	tests := []struct {
		name     string
		pointer  string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "navigate to map key",
			pointer:  "/config",
			expected: map[string]interface{}{"version": "1.0", "debug": true},
			wantErr:  false,
		},
		{
			name:     "navigate to nested map value",
			pointer:  "/config/version", // This works because Go's built-in map navigation is supported
			expected: "1.0",
			wantErr:  false,
		},
		{
			name:     "navigate to array index",
			pointer:  "/users",
			expected: &MockIndexNavigable{Data: []interface{}{map[string]interface{}{"name": "Alice", "id": 1}, map[string]interface{}{"name": "Bob", "id": 2}}},
			wantErr:  false,
		},
		{
			name:     "navigate through array index",
			pointer:  "/users/0",
			expected: map[string]interface{}{"name": "Alice", "id": 1},
			wantErr:  false,
		},
		{
			name:     "navigate to slice index",
			pointer:  "/1", // Using index navigation on the EitherValue itself
			expected: map[string]interface{}{"nested": "value"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer(tt.pointer))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEitherValue_JSONPointer_RightValue(t *testing.T) {
	// Test with Right value set
	rightValue := &MockKeyNavigable{
		Data: map[string]interface{}{
			"status": "active",
			"count":  42,
		},
	}

	eitherValue := &EitherValue[string, string, MockKeyNavigable, MockKeyNavigable]{
		Right: rightValue,
	}

	// Test JSON pointer navigation
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/status"))
	require.NoError(t, err)
	assert.Equal(t, "active", result)

	result, err = jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/count"))
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestEitherValue_JSONPointer_UnsupportedNavigation(t *testing.T) {
	// Test with value that doesn't support the requested navigation type
	eitherValue := &EitherValue[string, string, string, string]{
		Left: stringPtr("simple string"),
	}

	// Try to navigate with key (should fail because string doesn't support navigation)
	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/somekey"))
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "expected map, slice, or struct, got string")
}

func TestEitherValue_JSONPointer_RootPointer(t *testing.T) {
	// Test with root pointer "/" - this actually returns the EitherValue itself since "/" means empty path
	leftValue := &MockKeyNavigable{
		Data: map[string]interface{}{"test": "value"},
	}

	eitherValue := &EitherValue[MockKeyNavigable, MockKeyNavigable, string, string]{
		Left: leftValue,
	}

	result, err := jsonpointer.GetTarget(eitherValue, jsonpointer.JSONPointer("/"))
	require.NoError(t, err)
	assert.Equal(t, eitherValue, result)
}
