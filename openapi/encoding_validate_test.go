package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestEncoding_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_minimal",
			yml:  `{}`,
		},
		{
			name: "valid_with_content_type",
			yml: `
contentType: application/json
`,
		},
		{
			name: "valid_with_wildcard_content_type",
			yml: `
contentType: image/*
`,
		},
		{
			name: "valid_with_multiple_content_types",
			yml: `
contentType: application/json,application/xml
`,
		},
		{
			name: "valid_with_style_form",
			yml: `
style: form
explode: true
`,
		},
		{
			name: "valid_with_style_space_delimited",
			yml: `
style: spaceDelimited
explode: false
`,
		},
		{
			name: "valid_with_style_pipe_delimited",
			yml: `
style: pipeDelimited
explode: false
`,
		},
		{
			name: "valid_with_style_deep_object",
			yml: `
style: deepObject
explode: true
`,
		},
		{
			name: "valid_with_headers",
			yml: `
contentType: application/json
headers:
  X-Rate-Limit:
    description: Rate limit header
    schema:
      type: integer
  X-Custom-Header:
    description: Custom header
    schema:
      type: string
`,
		},
		{
			name: "valid_with_allow_reserved",
			yml: `
contentType: application/x-www-form-urlencoded
style: form
explode: true
allowReserved: true
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
contentType: application/json
style: form
x-custom: value
x-encoding-type: special
`,
		},
		{
			name: "valid_complete",
			yml: `
contentType: multipart/form-data
headers:
  Content-Disposition:
    description: Content disposition header
    schema:
      type: string
style: form
explode: true
allowReserved: false
x-custom: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var encoding openapi.Encoding

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &encoding)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := encoding.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestEncoding_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yml         string
		expectedErr string
	}{
		{
			name: "invalid_style",
			yml: `
style: invalidStyle
`,
			expectedErr: "style must be one of [form, spaceDelimited, pipeDelimited, deepObject]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var encoding openapi.Encoding

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &encoding)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := encoding.Validate(t.Context())
			require.NotEmpty(t, errs, "Expected validation errors")
			require.Contains(t, errs[0].Error(), tt.expectedErr)
		})
	}
}
