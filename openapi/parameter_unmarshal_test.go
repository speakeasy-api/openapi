package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestParameter_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: userId
in: path
required: true
schema:
  type: string
  pattern: "^[0-9]+$"
description: The user ID
deprecated: false
allowEmptyValue: false
style: simple
explode: false
allowReserved: false
example: "123"
examples:
  valid:
    value: "456"
    summary: Valid user ID
x-test: some-value
`

	var param openapi.Parameter

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &param)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "userId", param.GetName())
	require.Equal(t, openapi.ParameterInPath, param.GetIn())
	require.True(t, param.GetRequired())
	require.Equal(t, "The user ID", param.GetDescription())
	require.False(t, param.GetDeprecated())
	require.False(t, param.GetAllowEmptyValue())
	require.Equal(t, openapi.SerializationStyleSimple, param.GetStyle())
	require.False(t, param.GetExplode())
	require.False(t, param.GetAllowReserved())

	schema := param.GetSchema()
	require.NotNil(t, schema)

	example := param.GetExample()
	require.NotNil(t, example)

	examples := param.GetExamples()
	require.NotNil(t, examples)
	validExample, ok := examples.Get("valid")
	require.True(t, ok)
	require.Equal(t, "Valid user ID", validExample.Object.GetSummary())

	ext, ok := param.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
