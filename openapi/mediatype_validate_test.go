package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
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
			wantErrs: []string{"schema field type value must be one of"},
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

			for _, expectedErr := range tt.wantErrs {
				found := false
				for _, errMsg := range errMessages {
					if strings.Contains(errMsg, expectedErr) {
						found = true
						break
					}
				}
				require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
			}
		})
	}
}
