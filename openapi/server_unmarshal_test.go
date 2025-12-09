package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestServer_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
url: https://{environment}.example.com/{version}
description: Server with variables
variables:
  environment:
    default: api
    enum:
      - api
      - staging
    description: Environment name
  version:
    default: v1
    description: API version
x-test: some-value
`

	var server openapi.Server

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &server)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "https://{environment}.example.com/{version}", server.GetURL())
	require.Equal(t, "Server with variables", server.GetDescription())

	variables := server.GetVariables()
	require.NotNil(t, variables)

	envVar, ok := variables.Get("environment")
	require.True(t, ok)
	require.Equal(t, "api", envVar.GetDefault())
	require.Equal(t, "Environment name", envVar.GetDescription())
	require.Equal(t, []string{"api", "staging"}, envVar.GetEnum())

	versionVar, ok := variables.Get("version")
	require.True(t, ok)
	require.Equal(t, "v1", versionVar.GetDefault())
	require.Equal(t, "API version", versionVar.GetDescription())

	ext, ok := server.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}

func TestServer_Unmarshal_WithName_Success(t *testing.T) {
	t.Parallel()

	yml := `
url: https://api.example.com/v1
description: Production server
name: prod
`

	var server openapi.Server

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &server)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "https://api.example.com/v1", server.GetURL())
	require.Equal(t, "Production server", server.GetDescription())
	require.Equal(t, "prod", server.GetName())
}

func TestServerVariable_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
default: production
enum:
  - production
  - staging
  - development
description: Environment name
`

	var variable openapi.ServerVariable

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &variable)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "production", variable.GetDefault())
	require.Equal(t, []string{"production", "staging", "development"}, variable.GetEnum())
	require.Equal(t, "Environment name", variable.GetDescription())
}
