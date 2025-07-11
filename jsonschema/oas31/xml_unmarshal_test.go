package oas31_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestXML_Unmarshal_Success(t *testing.T) {
	yml := `
name: user
namespace: https://example.com/schema
prefix: ex
attribute: true
wrapped: false
x-test: some-value
x-custom: custom-value
`

	var xml oas31.XML

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &xml)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "user", xml.GetName())
	require.Equal(t, "https://example.com/schema", xml.GetNamespace())
	require.Equal(t, "ex", xml.GetPrefix())
	require.True(t, xml.GetAttribute())
	require.False(t, xml.GetWrapped())

	extensions := xml.GetExtensions()
	require.NotNil(t, extensions)

	ext, ok := extensions.Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)

	ext, ok = extensions.Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "custom-value", ext.Value)
}
