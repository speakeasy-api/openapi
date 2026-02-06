package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaType_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid media type with schema only",
			yml: `
schema:
  type: string
`,
		},
		{
			name: "valid media type with schema and example",
			yml: `
schema:
  type: object
  properties:
    name:
      type: string
    age:
      type: integer
example:
  name: John
  age: 30
`,
		},
		{
			name: "valid media type with examples",
			yml: `
schema:
  type: string
examples:
  simple:
    value: "hello"
    summary: Simple string
  complex:
    value: "world"
    description: Another example
`,
		},
		{
			name: "valid media type with encoding",
			yml: `
schema:
  type: object
  properties:
    file:
      type: string
      format: binary
encoding:
  file:
    contentType: image/png
    headers:
      X-Rate-Limit:
        schema:
          type: integer
`,
		},
		{
			name: "valid media type with complex encoding",
			yml: `
schema:
  type: object
  properties:
    profileImage:
      type: string
      format: binary
    metadata:
      type: object
encoding:
  profileImage:
    contentType: image/jpeg
    style: form
    explode: true
    allowReserved: false
  metadata:
    contentType: application/json
`,
		},
		{
			name: "valid media type with extensions",
			yml: `
schema:
  type: string
x-test: some-value
x-custom: custom-data
`,
		},
		{
			name: "valid media type with itemSchema only",
			yml: `
itemSchema:
  type: object
  properties:
    id:
      type: integer
    name:
      type: string
`,
		},
		{
			name: "valid media type with itemSchema and example",
			yml: `
itemSchema:
  type: string
example: "hello world"
`,
		},
		{
			name: "valid media type with itemSchema and examples",
			yml: `
itemSchema:
  $ref: "#/components/schemas/User"
examples:
  user1:
    value:
      id: 1
      name: John
  user2:
    value:
      id: 2
      name: Jane
`,
		},
		{
			name: "valid media type with prefixEncoding",
			yml: `
schema:
  type: array
  prefixItems:
    - type: object
    - type: string
prefixEncoding:
  - contentType: application/json
  - contentType: text/plain
`,
		},
		{
			name: "valid media type with itemEncoding",
			yml: `
itemSchema:
  type: object
  properties:
    id:
      type: integer
itemEncoding:
  contentType: application/json
`,
		},
		{
			name: "valid media type with both prefixEncoding and itemEncoding",
			yml: `
schema:
  type: array
  prefixItems:
    - type: object
  items:
    type: string
prefixEncoding:
  - contentType: application/json
itemEncoding:
  contentType: text/plain
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mediaType openapi.MediaType
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &mediaType)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := mediaType.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, mediaType.Valid, "expected media type to be valid")
		})
	}
}

func TestMediaType_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid encoding header",
			yml: `
schema:
  type: object
  properties:
    file:
      type: string
      format: binary
encoding:
  file:
    headers:
      Invalid-Header:
        schema:
          type: invalid-type
`,
			wantErrs: []string{
				"[13:17] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
				"[13:17] error validation-type-mismatch schema.type expected `array`, got `string`",
			},
		},
		{
			name: "encoding and prefixEncoding cannot coexist",
			yml: `
schema:
  type: object
  properties:
    file:
      type: string
encoding:
  file:
    contentType: image/png
prefixEncoding:
  - contentType: application/json
`,
			wantErrs: []string{
				"[8:3] error validation-mutually-exclusive-fields mediaType.encoding is mutually exclusive with mediaType.prefixEncoding and mediaType.itemEncoding",
			},
		},
		{
			name: "encoding and itemEncoding cannot coexist",
			yml: `
schema:
  type: object
  properties:
    file:
      type: string
encoding:
  file:
    contentType: image/png
itemEncoding:
  contentType: application/json
`,
			wantErrs: []string{
				"[8:3] error validation-mutually-exclusive-fields mediaType.encoding is mutually exclusive with mediaType.prefixEncoding and mediaType.itemEncoding",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mediaType openapi.MediaType
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &mediaType)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := mediaType.Validate(t.Context())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, mediaType.Valid, "expected media type to be invalid")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range errs {
				errMessages = append(errMessages, err.Error())
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}
