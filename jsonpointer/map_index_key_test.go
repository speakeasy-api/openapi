package jsonpointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

func TestMapIntegerKeys_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		data        interface{}
		jsonPointer string
		expected    interface{}
	}{
		{
			name: "integer key in map[string]interface{}",
			data: map[string]interface{}{
				"200": "success response",
				"404": "not found",
				"500": "server error",
			},
			jsonPointer: "/200",
			expected:    "success response",
		},
		{
			name: "integer key with nested path in map[string]interface{}",
			data: map[string]interface{}{
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "OK",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
								},
							},
						},
					},
				},
			},
			jsonPointer: "/responses/200/description",
			expected:    "OK",
		},
		{
			name: "integer key mixed with string keys",
			data: map[string]interface{}{
				"paths": map[string]interface{}{
					"/users": map[string]interface{}{
						"get": map[string]interface{}{
							"responses": map[string]interface{}{
								"200": map[string]interface{}{
									"description": "List of users",
								},
								"400": map[string]interface{}{
									"description": "Bad request",
								},
							},
						},
					},
				},
			},
			jsonPointer: "/paths/~1users/get/responses/200/description",
			expected:    "List of users",
		},
		{
			name: "integer key in map[int]interface{}",
			data: map[int]interface{}{
				200: "success",
				404: "not found",
				500: "server error",
			},
			jsonPointer: "/200",
			expected:    "success",
		},
		{
			name: "integer key in nested map[int]interface{}",
			data: map[string]interface{}{
				"statusCodes": map[int]interface{}{
					200: "OK",
					404: "Not Found",
					500: "Internal Server Error",
				},
			},
			jsonPointer: "/statusCodes/200",
			expected:    "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetTarget(tt.data, JSONPointer(tt.jsonPointer))
			require.NoError(t, err, "should find the target successfully")
			assert.Equal(t, tt.expected, result, "should return the expected value")
		})
	}
}

// TestStruct implements KeyNavigable and IndexNavigable interfaces
type TestStruct struct {
	data map[string]interface{}
}

// NavigateWithKey implements KeyNavigable interface
func (ts *TestStruct) NavigateWithKey(key string) (interface{}, error) {
	val, exists := ts.data[key]
	if !exists {
		return nil, ErrNotFound
	}
	return val, nil
}

// NavigateWithIndex implements IndexNavigable interface
func (ts *TestStruct) NavigateWithIndex(index int) (interface{}, error) {
	if index < 0 || index >= len(ts.data) {
		return nil, ErrNotFound
	}
	// For test purposes, convert index to string key
	val, exists := ts.data[string(rune('0'+index))]
	if !exists {
		return nil, ErrNotFound
	}
	return val, nil
}

func TestStructKeyNavigableIndexNavigable_Success(t *testing.T) {
	t.Parallel()

	testStruct := &TestStruct{
		data: map[string]interface{}{
			"200": "success response",
			"404": "not found",
			"0":   "first item",
			"1":   "second item",
		},
	}

	tests := []struct {
		name        string
		data        interface{}
		jsonPointer string
		expected    interface{}
	}{
		{
			name:        "integer key should try key navigation first",
			data:        testStruct,
			jsonPointer: "/200",
			expected:    "success response",
		},
		{
			name:        "integer key should fallback to index navigation if key fails",
			data:        testStruct,
			jsonPointer: "/0",
			expected:    "first item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetTarget(tt.data, JSONPointer(tt.jsonPointer))
			require.NoError(t, err, "should find the target successfully")
			assert.Equal(t, tt.expected, result, "should return the expected value")
		})
	}
}

func TestEmbeddedMapModel_Success(t *testing.T) {
	t.Parallel()

	// Test with the existing TestEmbeddedMapHighModel
	embeddedMap := sequencedmap.New[string, string]()
	embeddedMap.Set("200", "success status")
	embeddedMap.Set("404", "not found status")
	embeddedMap.Set("data", "some data")

	model := &tests.TestEmbeddedMapHighModel{
		Map: *embeddedMap,
	}

	tests := []struct {
		name        string
		data        interface{}
		jsonPointer string
		expected    interface{}
	}{
		{
			name:        "integer key in embedded map",
			data:        model,
			jsonPointer: "/200",
			expected:    "success status",
		},
		{
			name:        "integer key in nested embedded map",
			data:        model,
			jsonPointer: "/404",
			expected:    "not found status",
		},
		{
			name:        "regular string key in embedded map",
			data:        model,
			jsonPointer: "/data",
			expected:    "some data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetTarget(tt.data, JSONPointer(tt.jsonPointer))
			require.NoError(t, err, "should find the target successfully")
			assert.Equal(t, tt.expected, result, "should return the expected value")
		})
	}
}

func TestEdgeCases_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		data        interface{}
		jsonPointer string
		expected    interface{}
	}{
		{
			name: "zero integer key",
			data: map[string]interface{}{
				"0": "zero value",
			},
			jsonPointer: "/0",
			expected:    "zero value",
		},
		{
			name: "negative integer key as string",
			data: map[string]interface{}{
				"-1": "negative value",
			},
			jsonPointer: "/-1",
			expected:    "negative value",
		},
		{
			name: "large integer key",
			data: map[string]interface{}{
				"999999": "large number",
			},
			jsonPointer: "/999999",
			expected:    "large number",
		},
		{
			name: "integer key that looks like array index but is a map key",
			data: map[string]interface{}{
				"items": map[string]interface{}{
					"0": "map zero",
					"1": "map one",
				},
			},
			jsonPointer: "/items/0",
			expected:    "map zero",
		},
		{
			name: "both array and map with same integer key",
			data: map[string]interface{}{
				"mixed": map[string]interface{}{
					"0":     "map zero",
					"array": []string{"array zero", "array one"},
				},
			},
			jsonPointer: "/mixed/0",
			expected:    "map zero", // Should prefer key over index
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetTarget(tt.data, JSONPointer(tt.jsonPointer))
			require.NoError(t, err, "should find the target successfully")
			assert.Equal(t, tt.expected, result, "should return the expected value")
		})
	}
}

func TestMapIntegerKeys_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		data        interface{}
		jsonPointer string
		expectError bool
	}{
		{
			name: "non-existent integer key",
			data: map[string]interface{}{
				"200": "success",
				"404": "not found",
			},
			jsonPointer: "/500",
			expectError: true,
		},
		{
			name: "invalid path with integer key",
			data: map[string]interface{}{
				"200": "success",
			},
			jsonPointer: "/200/invalid",
			expectError: true,
		},
		{
			name:        "integer key on non-navigable type",
			data:        "string value",
			jsonPointer: "/200",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := GetTarget(tt.data, JSONPointer(tt.jsonPointer))
			if tt.expectError {
				require.Error(t, err, "should return an error for invalid path")
			} else {
				require.NoError(t, err, "should not return an error")
			}
		})
	}
}
