package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestRequestBody_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
description: User data for creation
required: true
content:
  application/json:
    schema:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
    examples:
      user1:
        value:
          name: John
          age: 30
        summary: Example user
  application/xml:
    schema:
      type: object
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
`

	var requestBody openapi.RequestBody

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &requestBody)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "User data for creation", requestBody.GetDescription())
	require.True(t, requestBody.GetRequired())

	content := requestBody.GetContent()
	require.NotNil(t, content)

	jsonContent, ok := content.Get("application/json")
	require.True(t, ok)
	require.NotNil(t, jsonContent.GetSchema())

	examples := jsonContent.GetExamples()
	require.NotNil(t, examples)
	user1Example, ok := examples.Get("user1")
	require.True(t, ok)
	require.Equal(t, "Example user", user1Example.Object.GetSummary())

	xmlContent, ok := content.Get("application/xml")
	require.True(t, ok)
	require.NotNil(t, xmlContent.GetSchema())

	formContent, ok := content.Get("multipart/form-data")
	require.True(t, ok)
	require.NotNil(t, formContent.GetSchema())

	encoding := formContent.GetEncoding()
	require.NotNil(t, encoding)
	fileEncoding, ok := encoding.Get("file")
	require.True(t, ok)
	require.Equal(t, "image/png", fileEncoding.GetContentTypeValue())
}
