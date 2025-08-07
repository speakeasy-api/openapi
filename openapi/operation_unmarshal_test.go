package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestOperation_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
operationId: getUserById
summary: Get user by ID
description: Retrieves a user by their unique identifier
tags:
  - users
  - accounts
deprecated: false
servers:
  - url: https://api.example.com/v1
    description: Production server
security:
  - ApiKeyAuth: []
parameters:
  - name: userId
    in: path
    required: true
    schema:
      type: string
requestBody:
  description: User data
  required: true
  content:
    application/json:
      schema:
        type: object
responses:
  "200":
    description: User found
    content:
      application/json:
        schema:
          type: object
  "404":
    description: User not found
callbacks:
  userCreated:
    "{$request.body#/callbackUrl}":
      post:
        responses:
          "200":
            description: Callback received
externalDocs:
  description: More info
  url: https://example.com/docs
x-test: some-value
`

	var operation openapi.Operation

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &operation)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "getUserById", operation.GetOperationID())
	require.Equal(t, "Get user by ID", operation.GetSummary())
	require.Equal(t, "Retrieves a user by their unique identifier", operation.GetDescription())
	require.False(t, operation.GetDeprecated())

	tags := operation.GetTags()
	require.Equal(t, []string{"users", "accounts"}, tags)

	servers := operation.GetServers()
	require.Len(t, servers, 1)
	require.Equal(t, "https://api.example.com/v1", servers[0].GetURL())

	security := operation.GetSecurity()
	require.Len(t, security, 1)

	parameters := operation.GetParameters()
	require.Len(t, parameters, 1)
	require.Equal(t, "userId", parameters[0].Object.GetName())

	requestBody := operation.GetRequestBody()
	require.NotNil(t, requestBody)
	require.Equal(t, "User data", requestBody.Object.GetDescription())

	responses := operation.GetResponses()
	require.NotNil(t, responses)

	callbacks := operation.GetCallbacks()
	require.NotNil(t, callbacks)
	userCreatedCallback, ok := callbacks.Get("userCreated")
	require.True(t, ok)
	require.NotNil(t, userCreatedCallback)

	externalDocs := operation.GetExternalDocs()
	require.NotNil(t, externalDocs)
	require.Equal(t, "More info", externalDocs.GetDescription())
	require.Equal(t, "https://example.com/docs", externalDocs.GetURL())

	ext, ok := operation.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
