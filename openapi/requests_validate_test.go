package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestBody_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid request body with content",
			yml: `
content:
  application/json:
    schema:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
description: User data
`,
		},
		{
			name: "valid required request body",
			yml: `
required: true
content:
  application/json:
    schema:
      type: object
  application/xml:
    schema:
      type: object
description: Required user data
`,
		},
		{
			name: "valid request body with multiple content types",
			yml: `
content:
  application/json:
    schema:
      type: object
    examples:
      user:
        value:
          name: John
          age: 30
  application/xml:
    schema:
      type: object
  text/plain:
    schema:
      type: string
description: Multi-format request body
`,
		},
		{
			name: "valid request body with encoding",
			yml: `
content:
  multipart/form-data:
    schema:
      type: object
      properties:
        file:
          type: string
          format: binary
        metadata:
          type: object
    encoding:
      file:
        contentType: image/png
        style: form
      metadata:
        contentType: application/json
description: File upload request
`,
		},
		{
			name: "valid minimal request body",
			yml: `
content:
  application/json:
    schema:
      type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var requestBody openapi.RequestBody
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &requestBody)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := requestBody.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, requestBody.Valid, "expected request body to be valid")
		})
	}
}

func TestRequestBody_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing content",
			yml: `
description: Request body without content
required: true
`,
			wantErrs: []string{"[2:1] error validation-required-field requestBody.content is required"},
		},
		{
			name: "empty content",
			yml: `
content: {}
description: Request body with empty content
`,
			wantErrs: []string{"[2:10] error validation-required-field requestBody.content is required"},
		},
		{
			name: "invalid schema in content",
			yml: `
content:
  application/json:
    schema:
      type: invalid-type
description: Request body with invalid schema
`,
			wantErrs: []string{
				"[5:13] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
				"[5:13] error validation-type-mismatch schema.type expected array, got string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var requestBody openapi.RequestBody

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &requestBody)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := requestBody.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

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
