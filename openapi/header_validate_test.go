package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeader_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid header with schema",
			yml: `
schema:
  type: string
description: API version header
`,
		},
		{
			name: "valid required header",
			yml: `
required: true
schema:
  type: string
  pattern: "^v[0-9]+$"
description: Version header
`,
		},
		{
			name: "valid header with content",
			yml: `
content:
  application/json:
    schema:
      type: object
      properties:
        version:
          type: string
description: Complex header content
`,
		},
		{
			name: "valid header with examples",
			yml: `
schema:
  type: string
examples:
  v1:
    value: "v1.0"
    summary: Version 1
  v2:
    value: "v2.0"
    summary: Version 2
description: Version header with examples
`,
		},
		{
			name: "valid header with style and explode",
			yml: `
schema:
  type: array
  items:
    type: string
style: simple
explode: false
description: Array header
`,
		},
		{
			name: "valid deprecated header",
			yml: `
deprecated: true
schema:
  type: string
description: Deprecated header
`,
		},
		{
			name: "valid header with extensions",
			yml: `
schema:
  type: string
description: Header with extensions
x-test: some-value
x-custom: custom-data
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header openapi.Header
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := header.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, header.Valid, "expected header to be valid")
		})
	}
}

func TestHeader_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid schema type",
			yml: `
schema:
  type: invalid-type
description: Header with invalid schema
`,
			wantErrs: []string{
				"[3:9] schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
				"[3:9] schema.type expected array, got string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header openapi.Header
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := header.Validate(t.Context())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, header.Valid, "expected header to be invalid")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range errs {
				errMessages = append(errMessages, err.Error())
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}
