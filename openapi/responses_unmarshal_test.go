package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestResponse_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
description: User data response
headers:
  X-Rate-Limit:
    description: Rate limit remaining
    schema:
      type: integer
  X-Expires-After:
    description: Expiration time
    schema:
      type: string
      format: date-time
content:
  application/json:
    schema:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
    examples:
      user1:
        value:
          id: 1
          name: John
        summary: Example user
  application/xml:
    schema:
      type: object
links:
  GetUserByUserId:
    operationId: getUserById
    parameters:
      userId: $response.body#/id
x-test: some-value
`

	var response openapi.Response

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &response)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "User data response", response.GetDescription())

	headers := response.GetHeaders()
	require.NotNil(t, headers)
	rateLimitHeader, ok := headers.Get("X-Rate-Limit")
	require.True(t, ok)
	require.Equal(t, "Rate limit remaining", rateLimitHeader.Object.GetDescription())

	content := response.GetContent()
	require.NotNil(t, content)
	jsonContent, ok := content.Get("application/json")
	require.True(t, ok)
	require.NotNil(t, jsonContent.GetSchema())

	examples := jsonContent.GetExamples()
	require.NotNil(t, examples)
	user1Example, ok := examples.Get("user1")
	require.True(t, ok)
	require.Equal(t, "Example user", user1Example.Object.GetSummary())

	links := response.GetLinks()
	require.NotNil(t, links)
	getUserLink, ok := links.Get("GetUserByUserId")
	require.True(t, ok)
	require.NotNil(t, getUserLink)

	ext, ok := response.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}

func TestResponses_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
"200":
  description: Success
  content:
    application/json:
      schema:
        type: object
"404":
  description: Not found
"500":
  description: Internal server error
default:
  description: Default response
  content:
    application/json:
      schema:
        type: object
x-test: some-value
`

	var responses openapi.Responses

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &responses)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	defaultResponse := responses.GetDefault()
	require.NotNil(t, defaultResponse)
	require.Equal(t, "Default response", defaultResponse.Object.GetDescription())

	ext, ok := responses.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
