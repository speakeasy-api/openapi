package oas31_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestExternalDocumentation_Unmarshal_Success(t *testing.T) {
	yml := `
description: Find more info here
url: https://example.com/docs
x-test: some-value
x-custom: custom-value
`

	var extDocs oas31.ExternalDocumentation

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &extDocs)
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
