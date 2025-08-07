package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestLink_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
operationId: getUserById
parameters:
  id: '$response.body#/id'
  format: json
  limit: 10
requestBody: '$response.body#/user'
description: Link to get user by ID with parameters and request body
server:
  url: https://api.example.com/v2
  description: Version 2 API server
  variables:
    version:
      default: v2
      description: API version
x-custom: value
x-timeout: 30
x-retry-count: 3
`

	var link openapi.Link

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &link)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify basic fields
	require.Equal(t, "getUserById", link.GetOperationID())
	require.Equal(t, "", link.GetOperationRef()) // Should be empty since we used operationId
	require.Equal(t, "Link to get user by ID with parameters and request body", link.GetDescription())

	// Verify parameters
	require.NotNil(t, link.Parameters)
	require.Equal(t, 3, link.Parameters.Len())

	// Check parameter existence
	require.True(t, link.Parameters.Has("id"))
	require.True(t, link.Parameters.Has("format"))
	require.True(t, link.Parameters.Has("limit"))

	// Verify request body
	require.NotNil(t, link.RequestBody)

	// Verify server
	require.NotNil(t, link.Server)
	require.Equal(t, "https://api.example.com/v2", link.Server.GetURL())
	require.Equal(t, "Version 2 API server", link.Server.GetDescription())
	require.NotNil(t, link.Server.Variables)
	require.True(t, link.Server.Variables.Has("version"))

	// Verify extensions
	require.NotNil(t, link.Extensions)
	require.True(t, link.Extensions.Has("x-custom"))
	require.True(t, link.Extensions.Has("x-timeout"))
	require.True(t, link.Extensions.Has("x-retry-count"))
}

func TestLink_Unmarshal_OperationRef(t *testing.T) {
	t.Parallel()

	yml := `
operationRef: '#/paths/~1users~1{id}/get'
description: Reference to get user operation
parameters:
  userId: '$response.body#/id'
`

	var link openapi.Link

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &link)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify operationRef is used instead of operationId
	require.Equal(t, "", link.GetOperationID()) // Should be empty since we used operationRef
	require.Equal(t, "#/paths/~1users~1{id}/get", link.GetOperationRef())
	require.Equal(t, "Reference to get user operation", link.GetDescription())

	// Verify parameters
	require.NotNil(t, link.Parameters)
	require.Equal(t, 1, link.Parameters.Len())
	require.True(t, link.Parameters.Has("userId"))
}

func TestLink_Unmarshal_Minimal(t *testing.T) {
	t.Parallel()

	yml := `
operationId: simpleOperation
`

	var link openapi.Link

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &link)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify minimal link
	require.Equal(t, "simpleOperation", link.GetOperationID())
	require.Equal(t, "", link.GetOperationRef())
	require.Equal(t, "", link.GetDescription())
	require.Nil(t, link.Parameters)
	require.Nil(t, link.RequestBody)
	require.Nil(t, link.Server)
	require.Nil(t, link.Extensions)
}
