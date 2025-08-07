package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestEncoding_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
contentType: multipart/form-data
headers:
  Content-Disposition:
    description: Content disposition header for file uploads
    schema:
      type: string
      example: 'form-data; name="file"; filename="example.txt"'
  X-Upload-ID:
    description: Unique upload identifier
    schema:
      type: string
      format: uuid
  X-File-Size:
    description: Size of the uploaded file
    schema:
      type: integer
      minimum: 0
style: form
explode: true
allowReserved: false
x-custom: value
x-encoding-version: 1.0
x-max-file-size: 10485760
`

	var encoding openapi.Encoding

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &encoding)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify basic fields
	require.Equal(t, "multipart/form-data", encoding.GetContentTypeValue())
	require.Equal(t, openapi.SerializationStyleForm, encoding.GetStyle())
	require.True(t, encoding.GetExplode())
	require.False(t, encoding.GetAllowReserved())

	// Verify headers
	require.NotNil(t, encoding.Headers)
	require.Equal(t, 3, encoding.Headers.Len())

	// Check Content-Disposition header
	contentDisposition, exists := encoding.Headers.Get("Content-Disposition")
	require.True(t, exists)
	require.NotNil(t, contentDisposition.Object)
	require.Equal(t, "Content disposition header for file uploads", contentDisposition.Object.GetDescription())

	// Check X-Upload-ID header
	uploadID, exists := encoding.Headers.Get("X-Upload-ID")
	require.True(t, exists)
	require.NotNil(t, uploadID.Object)
	require.Equal(t, "Unique upload identifier", uploadID.Object.GetDescription())

	// Check X-File-Size header
	fileSize, exists := encoding.Headers.Get("X-File-Size")
	require.True(t, exists)
	require.NotNil(t, fileSize.Object)
	require.Equal(t, "Size of the uploaded file", fileSize.Object.GetDescription())

	// Verify extensions
	require.NotNil(t, encoding.Extensions)
	require.True(t, encoding.Extensions.Has("x-custom"))
	require.True(t, encoding.Extensions.Has("x-encoding-version"))
	require.True(t, encoding.Extensions.Has("x-max-file-size"))
}

func TestEncoding_Unmarshal_Minimal(t *testing.T) {
	t.Parallel()

	yml := `{}`

	var encoding openapi.Encoding

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &encoding)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify defaults
	require.Equal(t, "", encoding.GetContentTypeValue())
	require.Equal(t, openapi.SerializationStyleForm, encoding.GetStyle()) // Default style
	require.True(t, encoding.GetExplode())                                // Default explode for form style
	require.False(t, encoding.GetAllowReserved())                         // Default allowReserved
	require.Nil(t, encoding.Headers)
	require.Nil(t, encoding.Extensions)
}

func TestEncoding_Unmarshal_StyleVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		yml             string
		expectedStyle   openapi.SerializationStyle
		expectedExplode bool
	}{
		{
			name: "form_style_explicit_explode",
			yml: `
style: form
explode: false
`,
			expectedStyle:   openapi.SerializationStyleForm,
			expectedExplode: false,
		},
		{
			name: "space_delimited_style",
			yml: `
style: spaceDelimited
`,
			expectedStyle:   openapi.SerializationStyleSpaceDelimited,
			expectedExplode: false, // Default for non-form styles
		},
		{
			name: "pipe_delimited_style",
			yml: `
style: pipeDelimited
explode: true
`,
			expectedStyle:   openapi.SerializationStylePipeDelimited,
			expectedExplode: true,
		},
		{
			name: "deep_object_style",
			yml: `
style: deepObject
`,
			expectedStyle:   openapi.SerializationStyleDeepObject,
			expectedExplode: false, // Default for non-form styles
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var encoding openapi.Encoding

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &encoding)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			require.Equal(t, tt.expectedStyle, encoding.GetStyle())
			require.Equal(t, tt.expectedExplode, encoding.GetExplode())
		})
	}
}

func TestEncoding_Unmarshal_ContentTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		yml        string
		expectedCT string
	}{
		{
			name: "json_content_type",
			yml: `
contentType: application/json
`,
			expectedCT: "application/json",
		},
		{
			name: "wildcard_content_type",
			yml: `
contentType: image/*
`,
			expectedCT: "image/*",
		},
		{
			name: "multiple_content_types",
			yml: `
contentType: application/json,application/xml,text/plain
`,
			expectedCT: "application/json,application/xml,text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var encoding openapi.Encoding

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &encoding)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			require.Equal(t, tt.expectedCT, encoding.GetContentTypeValue())
		})
	}
}
