package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestTag_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: pets
description: Everything about your pets
externalDocs:
  description: Find out more
  url: https://example.com/pets
x-test: some-value
`

	var tag openapi.Tag

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &tag)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "pets", tag.GetName())
	require.Equal(t, "Everything about your pets", tag.GetDescription())

	extDocs := tag.GetExternalDocs()
	require.NotNil(t, extDocs)
	require.Equal(t, "Find out more", extDocs.GetDescription())
	require.Equal(t, "https://example.com/pets", extDocs.GetURL())

	ext, ok := tag.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
