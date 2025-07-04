package oas31_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestDiscriminator_Unmarshal_Success(t *testing.T) {
	yml := `
propertyName: petType
mapping:
  dog: "#/components/schemas/Dog"
  cat: "#/components/schemas/Cat"
  bird: "#/components/schemas/Bird"
x-test: some-value
x-custom: custom-value
`

	var discriminator oas31.Discriminator

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &discriminator)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "petType", discriminator.GetPropertyName())

	mapping := discriminator.GetMapping()
	require.NotNil(t, mapping)
	require.Equal(t, 3, mapping.Len())

	dogRef, ok := mapping.Get("dog")
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/Dog", dogRef)

	catRef, ok := mapping.Get("cat")
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/Cat", catRef)

	birdRef, ok := mapping.Get("bird")
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/Bird", birdRef)

	extensions := discriminator.GetExtensions()
	require.NotNil(t, extensions)

	ext, ok := extensions.Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)

	ext, ok = extensions.Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "custom-value", ext.Value)
}
