package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestExternalDocumentation_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
description: Find more info here
url: https://example.com/docs
x-test: some-value
x-custom: custom-value
`

	var extDocs oas3.ExternalDocumentation

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &extDocs)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "Find more info here", extDocs.GetDescription())
	require.Equal(t, "https://example.com/docs", extDocs.GetURL())

	extensions := extDocs.GetExtensions()
	require.NotNil(t, extensions)

	ext, ok := extensions.Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)

	ext, ok = extensions.Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "custom-value", ext.Value)
}
