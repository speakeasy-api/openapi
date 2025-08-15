package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestHeader_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
schema:
  type: string
description: API version header
x-test: some-value
`

	var header openapi.Header

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &header)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "API version header", header.GetDescription())

	schema := header.GetSchema()
	require.NotNil(t, schema)

	ext, ok := header.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
