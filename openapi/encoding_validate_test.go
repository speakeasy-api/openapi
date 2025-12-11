package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
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

func TestEncoding_Getters_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		yml               string
		wantContentType   string
		wantStyle         openapi.SerializationStyle
		wantExplode       bool
		wantAllowReserved bool
	}{
		{
			name: "all fields set",
			yml: `
contentType: application/json
style: form
explode: true
allowReserved: true
`,
			wantContentType:   "application/json",
			wantStyle:         openapi.SerializationStyleForm,
			wantExplode:       true,
			wantAllowReserved: true,
		},
		{
			name:              "defaults - no style",
			yml:               `{}`,
			wantContentType:   "application/octet-stream",
			wantStyle:         openapi.SerializationStyleForm,
			wantExplode:       true, // form style defaults to explode=true
			wantAllowReserved: false,
		},
		{
			name: "non-form style defaults explode to false",
			yml: `
style: pipeDelimited
`,
			wantContentType:   "application/octet-stream",
			wantStyle:         openapi.SerializationStylePipeDelimited,
			wantExplode:       false,
			wantAllowReserved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var encoding openapi.Encoding
			_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &encoding)
			require.NoError(t, err)

			assert.Equal(t, tt.wantContentType, encoding.GetContentType(nil), "GetContentType mismatch")
			assert.Equal(t, tt.wantStyle, encoding.GetStyle(), "GetStyle mismatch")
			assert.Equal(t, tt.wantExplode, encoding.GetExplode(), "GetExplode mismatch")
			assert.Equal(t, tt.wantAllowReserved, encoding.GetAllowReserved(), "GetAllowReserved mismatch")
			assert.NotNil(t, encoding.GetExtensions(), "GetExtensions should never be nil")
		})
	}
}

func TestEncoding_Getters_Nil(t *testing.T) {
	t.Parallel()

	var encoding *openapi.Encoding = nil

	assert.Equal(t, "application/octet-stream", encoding.GetContentType(nil), "nil encoding GetContentType should return default")
	assert.Empty(t, encoding.GetContentTypeValue(), "nil encoding GetContentTypeValue should return empty")
	assert.Equal(t, openapi.SerializationStyleForm, encoding.GetStyle(), "nil encoding GetStyle should return form")
	assert.True(t, encoding.GetExplode(), "nil encoding GetExplode should return true (form default)")
	assert.False(t, encoding.GetAllowReserved(), "nil encoding GetAllowReserved should return false")
	assert.Nil(t, encoding.GetHeaders(), "nil encoding GetHeaders should return nil")
	assert.NotNil(t, encoding.GetExtensions(), "nil encoding GetExtensions should return empty")
}

func TestEncoding_GetHeaders_Success(t *testing.T) {
	t.Parallel()

	yml := `
contentType: application/json
headers:
  X-Rate-Limit:
    schema:
      type: integer
  X-Custom:
    schema:
      type: string
`

	var encoding openapi.Encoding
	_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &encoding)
	require.NoError(t, err)

	headers := encoding.GetHeaders()
	require.NotNil(t, headers, "GetHeaders should not be nil")
	assert.Equal(t, 2, headers.Len(), "headers should have two entries")
}
