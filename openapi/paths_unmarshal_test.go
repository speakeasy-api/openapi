package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestPaths_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
/users:
  get:
    summary: List users
    responses:
      '200':
        description: Successful response
  post:
    summary: Create user
    responses:
      '201':
        description: User created
/users/{id}:
  get:
    summary: Get user by ID
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: integer
    responses:
      '200':
        description: Successful response
x-custom: value
x-rate-limit: 100
`

	var paths openapi.Paths

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &paths)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify paths structure
	require.Equal(t, 2, paths.Len())

	// Verify /users path
	usersPath, exists := paths.Get("/users")
	require.True(t, exists)
	require.NotNil(t, usersPath.Object)
	require.Equal(t, 2, usersPath.Object.Len())

	// Verify GET operation
	getOp := usersPath.Object.Get()
	require.NotNil(t, getOp)
	require.Equal(t, "List users", getOp.GetSummary())
	require.NotNil(t, getOp.Responses)

	// Verify POST operation
	postOp := usersPath.Object.Post()
	require.NotNil(t, postOp)
	require.Equal(t, "Create user", postOp.GetSummary())
	require.NotNil(t, postOp.Responses)

	// Verify /users/{id} path
	userByIdPath, exists := paths.Get("/users/{id}")
	require.True(t, exists)
	require.NotNil(t, userByIdPath.Object)
	require.Equal(t, 1, userByIdPath.Object.Len())

	// Verify GET operation with parameters
	getUserOp := userByIdPath.Object.Get()
	require.NotNil(t, getUserOp)
	require.Equal(t, "Get user by ID", getUserOp.GetSummary())
	require.Len(t, getUserOp.Parameters, 1)
	require.Equal(t, "id", getUserOp.Parameters[0].Object.GetName())

	// Verify extensions
	require.NotNil(t, paths.Extensions)
	require.True(t, paths.Extensions.Has("x-custom"))
	require.True(t, paths.Extensions.Has("x-rate-limit"))
}

func TestPathItem_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
summary: User operations
description: Operations for managing users
servers:
  - url: https://api.example.com/v1
    description: Production server
  - url: https://staging-api.example.com/v1
    description: Staging server
parameters:
  - name: version
    in: header
    schema:
      type: string
  - name: format
    in: query
    schema:
      type: string
      enum: [json, xml]
get:
  summary: Get user
  responses:
    '200':
      description: Successful response
post:
  summary: Create user
  requestBody:
    content:
      application/json:
        schema:
          type: object
  responses:
    '201':
      description: User created
put:
  summary: Update user
  responses:
    '200':
      description: User updated
delete:
  summary: Delete user
  responses:
    '204':
      description: User deleted
options:
  summary: Get options
  responses:
    '200':
      description: Options response
head:
  summary: Get headers
  responses:
    '200':
      description: Headers response
patch:
  summary: Patch user
  responses:
    '200':
      description: User patched
trace:
  summary: Trace request
  responses:
    '200':
      description: Trace response
x-custom: value
x-rate-limit: 100
`

	var pathItem openapi.PathItem

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &pathItem)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify basic fields
	require.Equal(t, "User operations", pathItem.GetSummary())
	require.Equal(t, "Operations for managing users", pathItem.GetDescription())

	// Verify servers
	require.Len(t, pathItem.Servers, 2)
	require.Equal(t, "https://api.example.com/v1", pathItem.Servers[0].GetURL())
	require.Equal(t, "Production server", pathItem.Servers[0].GetDescription())
	require.Equal(t, "https://staging-api.example.com/v1", pathItem.Servers[1].GetURL())
	require.Equal(t, "Staging server", pathItem.Servers[1].GetDescription())

	// Verify parameters
	require.Len(t, pathItem.Parameters, 2)
	require.Equal(t, "version", pathItem.Parameters[0].Object.GetName())
	require.Equal(t, openapi.ParameterInHeader, pathItem.Parameters[0].Object.GetIn())
	require.Equal(t, "format", pathItem.Parameters[1].Object.GetName())
	require.Equal(t, openapi.ParameterInQuery, pathItem.Parameters[1].Object.GetIn())

	// Verify all HTTP methods
	require.Equal(t, 8, pathItem.Len())

	// Verify GET operation
	getOp := pathItem.Get()
	require.NotNil(t, getOp)
	require.Equal(t, "Get user", getOp.GetSummary())

	// Verify POST operation
	postOp := pathItem.Post()
	require.NotNil(t, postOp)
	require.Equal(t, "Create user", postOp.GetSummary())
	require.NotNil(t, postOp.RequestBody)

	// Verify PUT operation
	putOp := pathItem.Put()
	require.NotNil(t, putOp)
	require.Equal(t, "Update user", putOp.GetSummary())

	// Verify DELETE operation
	deleteOp := pathItem.Delete()
	require.NotNil(t, deleteOp)
	require.Equal(t, "Delete user", deleteOp.GetSummary())

	// Verify OPTIONS operation
	optionsOp := pathItem.Options()
	require.NotNil(t, optionsOp)
	require.Equal(t, "Get options", optionsOp.GetSummary())

	// Verify HEAD operation
	headOp := pathItem.Head()
	require.NotNil(t, headOp)
	require.Equal(t, "Get headers", headOp.GetSummary())

	// Verify PATCH operation
	patchOp := pathItem.Patch()
	require.NotNil(t, patchOp)
	require.Equal(t, "Patch user", patchOp.GetSummary())

	// Verify TRACE operation
	traceOp := pathItem.Trace()
	require.NotNil(t, traceOp)
	require.Equal(t, "Trace request", traceOp.GetSummary())

	// Verify extensions
	require.NotNil(t, pathItem.Extensions)
	require.True(t, pathItem.Extensions.Has("x-custom"))
	require.True(t, pathItem.Extensions.Has("x-rate-limit"))
}
