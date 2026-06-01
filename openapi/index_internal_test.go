package openapi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/stretchr/testify/assert"
)

func TestSchemaResolutionError_SanitizesReferenceErrors_Success(t *testing.T) {
	t.Parallel()

	schema := oas3.NewJSONSchemaFromReference("#/components/schemas/User")

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "not found error",
			err:      fmt.Errorf("resolve schema: %w", jsonpointer.ErrNotFound),
			expected: "reference not found: #/components/schemas/User",
		},
		{
			name:     "invalid path error",
			err:      fmt.Errorf("resolve schema: %w", jsonpointer.ErrInvalidPath),
			expected: "invalid reference path: #/components/schemas/User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := schemaResolutionError(schema, tt.err)
			assert.EqualError(t, actual, tt.expected, "should return sanitized reference error")
		})
	}
}

func TestSchemaResolutionError_ReturnsOriginalError_Success(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original resolution error")
	refSchema := oas3.NewJSONSchemaFromReference("#/components/schemas/User")
	nonRefSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{})

	tests := []struct {
		name   string
		schema *oas3.JSONSchemaReferenceable
		err    error
	}{
		{
			name:   "nil schema",
			schema: nil,
			err:    originalErr,
		},
		{
			name:   "schema without reference",
			schema: nonRefSchema,
			err:    originalErr,
		},
		{
			name:   "unmatched error",
			schema: refSchema,
			err:    originalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := schemaResolutionError(tt.schema, tt.err)
			assert.ErrorIs(t, actual, originalErr, "should return original error")
		})
	}
}

func TestSchemaResolutionError_NilError_Success(t *testing.T) {
	t.Parallel()

	schema := oas3.NewJSONSchemaFromReference("#/components/schemas/User")
	actual := schemaResolutionError(schema, nil)

	assert.NoError(t, actual, "nil error should remain nil")
}
