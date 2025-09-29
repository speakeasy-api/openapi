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

func TestTag_Unmarshal_WithNewFields_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: products
summary: Products
description: All product-related operations
parent: catalog
kind: nav
externalDocs:
  description: Product API documentation
  url: https://example.com/products
x-custom: custom-value
`

	var tag openapi.Tag

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &tag)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "products", tag.GetName())
	require.Equal(t, "Products", tag.GetSummary())
	require.Equal(t, "All product-related operations", tag.GetDescription())
	require.Equal(t, "catalog", tag.GetParent())
	require.Equal(t, "nav", tag.GetKind())

	extDocs := tag.GetExternalDocs()
	require.NotNil(t, extDocs)
	require.Equal(t, "Product API documentation", extDocs.GetDescription())
	require.Equal(t, "https://example.com/products", extDocs.GetURL())

	ext, ok := tag.GetExtensions().Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "custom-value", ext.Value)
}

func TestTag_Unmarshal_MinimalNewFields_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: minimal
summary: Minimal Tag
`

	var tag openapi.Tag

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &tag)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "minimal", tag.GetName())
	require.Equal(t, "Minimal Tag", tag.GetSummary())
	require.Equal(t, "", tag.GetDescription())
	require.Equal(t, "", tag.GetParent())
	require.Equal(t, "", tag.GetKind())
}

func TestTag_Unmarshal_KindValues_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     string
		expected string
	}{
		{"nav kind", "nav", "nav"},
		{"badge kind", "badge", "badge"},
		{"audience kind", "audience", "audience"},
		{"custom kind", "custom-value", "custom-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			yml := `
name: test
kind: ` + tt.kind

			var tag openapi.Tag

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &tag)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			require.Equal(t, "test", tag.GetName())
			require.Equal(t, tt.expected, tag.GetKind())
		})
	}
}
