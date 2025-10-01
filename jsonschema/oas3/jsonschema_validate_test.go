package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchema_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "schema fails direct validation",
			yml: `
"test"`,
			wantErrs: []string{"[2:1] failed to validate either Schema [expected object, got `test`] or bool [line 2: cannot unmarshal !!str `test` into bool]"},
		},
		{
			name: "child schema fails validation",
			yml: `
type:
    - array
    - "null"
items:
    $ref: "#/components/schemas/ffmpeg-profile"
default:
    $ref: "#/components/schemas/stream/properties/profiles/default"
description:
    $ref: "#/components/schemas/stream/properties/profiles/description"
`,
			wantErrs: []string{
				"[2:1] schema.description expected string, got object",
				"[10:5] schema.description expected string, got object",
			},
		},
		{
			name: "incorrect type fails validation",
			yml: `
type: invalid_type
`,
			wantErrs: []string{
				"[2:7] schema.type expected array, got string",
				"[2:7] schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.JSONSchema[oas3.Referenceable]

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := schema.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)
			validation.SortValidationErrors(allErrors)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}
