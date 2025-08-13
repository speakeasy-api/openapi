package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestExample_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
summary: Example of a pet
description: A pet object example
value:
  id: 1
  name: doggie
  status: available
externalValue: https://example.com/examples/pet.json
x-test: some-value
`

	var example openapi.Example

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &example)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "Example of a pet", example.GetSummary())
	require.Equal(t, "A pet object example", example.GetDescription())
	require.Equal(t, "https://example.com/examples/pet.json", example.GetExternalValue())

	value := example.GetValue()
	require.NotNil(t, value)

	ext, ok := example.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
