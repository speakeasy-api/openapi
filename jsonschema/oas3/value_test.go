package oas3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExclusiveMaximumFromBool_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    bool
		expected bool
	}{
		{
			name:     "true value",
			value:    true,
			expected: true,
		},
		{
			name:     "false value",
			value:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NewExclusiveMaximumFromBool(tt.value)
			require.NotNil(t, result, "result should not be nil")
			assert.True(t, result.IsLeft(), "should be left value (bool)")
			assert.Equal(t, tt.expected, *result.Left, "left value should match input")
		})
	}
}

func TestNewExclusiveMaximumFromFloat64_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{
			name:     "zero value",
			value:    0.0,
			expected: 0.0,
		},
		{
			name:     "positive value",
			value:    100.5,
			expected: 100.5,
		},
		{
			name:     "negative value",
			value:    -50.25,
			expected: -50.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NewExclusiveMaximumFromFloat64(tt.value)
			require.NotNil(t, result, "result should not be nil")
			assert.True(t, result.IsRight(), "should be right value (float64)")
			assert.InDelta(t, tt.expected, *result.Right, 0.0001, "right value should match input")
		})
	}
}

func TestNewExclusiveMinimumFromBool_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    bool
		expected bool
	}{
		{
			name:     "true value",
			value:    true,
			expected: true,
		},
		{
			name:     "false value",
			value:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NewExclusiveMinimumFromBool(tt.value)
			require.NotNil(t, result, "result should not be nil")
			assert.True(t, result.IsLeft(), "should be left value (bool)")
			assert.Equal(t, tt.expected, *result.Left, "left value should match input")
		})
	}
}

func TestNewExclusiveMinimumFromFloat64_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{
			name:     "zero value",
			value:    0.0,
			expected: 0.0,
		},
		{
			name:     "positive value",
			value:    100.5,
			expected: 100.5,
		},
		{
			name:     "negative value",
			value:    -50.25,
			expected: -50.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NewExclusiveMinimumFromFloat64(tt.value)
			require.NotNil(t, result, "result should not be nil")
			assert.True(t, result.IsRight(), "should be right value (float64)")
			assert.InDelta(t, tt.expected, *result.Right, 0.0001, "right value should match input")
		})
	}
}

func TestNewTypeFromArray_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    []SchemaType
		expected []SchemaType
	}{
		{
			name:     "empty array",
			value:    []SchemaType{},
			expected: []SchemaType{},
		},
		{
			name:     "single type",
			value:    []SchemaType{SchemaTypeString},
			expected: []SchemaType{SchemaTypeString},
		},
		{
			name:     "multiple types",
			value:    []SchemaType{SchemaTypeString, SchemaTypeNumber},
			expected: []SchemaType{SchemaTypeString, SchemaTypeNumber},
		},
		{
			name:     "all types",
			value:    []SchemaType{SchemaTypeObject, SchemaTypeArray, SchemaTypeString, SchemaTypeNumber, SchemaTypeInteger, SchemaTypeBoolean, SchemaTypeNull},
			expected: []SchemaType{SchemaTypeObject, SchemaTypeArray, SchemaTypeString, SchemaTypeNumber, SchemaTypeInteger, SchemaTypeBoolean, SchemaTypeNull},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NewTypeFromArray(tt.value)
			require.NotNil(t, result, "result should not be nil")
			assert.True(t, result.IsLeft(), "should be left value (array)")
			assert.Equal(t, tt.expected, *result.Left, "left value should match input")
		})
	}
}

func TestNewTypeFromString_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    SchemaType
		expected SchemaType
	}{
		{
			name:     "string type",
			value:    SchemaTypeString,
			expected: SchemaTypeString,
		},
		{
			name:     "number type",
			value:    SchemaTypeNumber,
			expected: SchemaTypeNumber,
		},
		{
			name:     "integer type",
			value:    SchemaTypeInteger,
			expected: SchemaTypeInteger,
		},
		{
			name:     "boolean type",
			value:    SchemaTypeBoolean,
			expected: SchemaTypeBoolean,
		},
		{
			name:     "object type",
			value:    SchemaTypeObject,
			expected: SchemaTypeObject,
		},
		{
			name:     "array type",
			value:    SchemaTypeArray,
			expected: SchemaTypeArray,
		},
		{
			name:     "null type",
			value:    SchemaTypeNull,
			expected: SchemaTypeNull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NewTypeFromString(tt.value)
			require.NotNil(t, result, "result should not be nil")
			assert.True(t, result.IsRight(), "should be right value (string)")
			assert.Equal(t, tt.expected, *result.Right, "right value should match input")
		})
	}
}
