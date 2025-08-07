package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestComponents_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
schemas:
  User:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
      email:
        type: string
        format: email
  Error:
    type: object
    properties:
      code:
        type: integer
      message:
        type: string
responses:
  NotFound:
    description: The specified resource was not found
    content:
      application/json:
        schema:
          $ref: '#/components/schemas/Error'
  Unauthorized:
    description: Unauthorized
    content:
      application/json:
        schema:
          $ref: '#/components/schemas/Error'
parameters:
  skipParam:
    name: skip
    in: query
    description: number of items to skip
    schema:
      type: integer
      format: int32
  limitParam:
    name: limit
    in: query
    description: max records to return
    schema:
      type: integer
      format: int32
examples:
  user-example:
    summary: User Example
    description: Example of a user object
    value:
      id: 1
      name: John Doe
      email: john@example.com
requestBodies:
  UserArray:
    description: user to add to the system
    content:
      application/json:
        schema:
          type: array
          items:
            $ref: '#/components/schemas/User'
      application/xml:
        schema:
          type: array
          items:
            $ref: '#/components/schemas/User'
headers:
  X-Rate-Limit-Limit:
    description: The number of allowed requests in the current period
    schema:
      type: integer
  X-Rate-Limit-Remaining:
    description: The number of requests left for the time window
    schema:
      type: integer
securitySchemes:
  ApiKeyAuth:
    type: apiKey
    in: header
    name: X-API-Key
  BearerAuth:
    type: http
    scheme: bearer
    bearerFormat: JWT
  OAuth2:
    type: oauth2
    flows:
      authorizationCode:
        authorizationUrl: https://example.com/oauth/authorize
        tokenUrl: https://example.com/oauth/token
        scopes:
          read: Grants read access
          write: Grants write access
links:
  UserRepositories:
    operationId: getRepositoriesByOwner
    parameters:
      username: $response.body#/login
  UserGists:
    operationId: getGistsByOwner
    parameters:
      username: $response.body#/login
callbacks:
  myWebhook:
    '{$request.body#/callbackUrl}':
      post:
        requestBody:
          description: Callback payload
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: string
        responses:
          '200':
            description: webhook successfully processed
pathItems:
  Pet:
    get:
      description: Returns a pet by ID
      operationId: getPetById
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: pet response
          content:
            application/json:
              schema:
                type: object
        '404':
          $ref: '#/components/responses/NotFound'
x-custom: value
x-another: 123
`

	var components openapi.Components

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &components)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Test schemas
	schemas := components.GetSchemas()
	require.NotNil(t, schemas)
	require.True(t, schemas.Has("User"))
	require.True(t, schemas.Has("Error"))

	// Test responses
	responses := components.GetResponses()
	require.NotNil(t, responses)
	notFoundResponse, ok := responses.Get("NotFound")
	require.True(t, ok)
	require.Equal(t, "The specified resource was not found", notFoundResponse.Object.GetDescription())

	// Test parameters
	parameters := components.GetParameters()
	require.NotNil(t, parameters)
	skipParam, ok := parameters.Get("skipParam")
	require.True(t, ok)
	require.Equal(t, "skip", skipParam.Object.GetName())

	// Test examples
	examples := components.GetExamples()
	require.NotNil(t, examples)
	userExample, ok := examples.Get("user-example")
	require.True(t, ok)
	require.Equal(t, "User Example", userExample.Object.GetSummary())

	// Test request bodies
	requestBodies := components.GetRequestBodies()
	require.NotNil(t, requestBodies)
	userArrayBody, ok := requestBodies.Get("UserArray")
	require.True(t, ok)
	require.Equal(t, "user to add to the system", userArrayBody.Object.GetDescription())

	// Test headers
	headers := components.GetHeaders()
	require.NotNil(t, headers)
	rateLimitHeader, ok := headers.Get("X-Rate-Limit-Limit")
	require.True(t, ok)
	require.Equal(t, "The number of allowed requests in the current period", rateLimitHeader.Object.GetDescription())

	// Test security schemes
	securitySchemes := components.GetSecuritySchemes()
	require.NotNil(t, securitySchemes)
	apiKeyAuth, ok := securitySchemes.Get("ApiKeyAuth")
	require.True(t, ok)
	require.Equal(t, openapi.SecuritySchemeTypeAPIKey, apiKeyAuth.Object.GetType())

	// Test links
	links := components.GetLinks()
	require.NotNil(t, links)
	userReposLink, ok := links.Get("UserRepositories")
	require.True(t, ok)
	require.NotNil(t, userReposLink)

	// Test callbacks
	callbacks := components.GetCallbacks()
	require.NotNil(t, callbacks)
	webhookCallback, ok := callbacks.Get("myWebhook")
	require.True(t, ok)
	require.NotNil(t, webhookCallback)

	// Test path items
	pathItems := components.GetPathItems()
	require.NotNil(t, pathItems)
	petPathItem, ok := pathItems.Get("Pet")
	require.True(t, ok)
	require.NotNil(t, petPathItem)

	// Test extensions
	extensions := components.GetExtensions()
	require.NotNil(t, extensions)
	customExt, ok := extensions.Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "value", customExt.Value)
}
