package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/require"
)

func TestLink_Validate_Success(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations for operationId validation
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add paths with operations that match the operationIds used in tests
	pathItem := openapi.NewPathItem()

	// Add GET operation with getUserById
	operation1 := &openapi.Operation{
		OperationID: pointer.From("getUserById"),
	}
	pathItem.Set("get", operation1)

	// Add PUT operation with updateUser to the same path
	operation2 := &openapi.Operation{
		OperationID: pointer.From("updateUser"),
	}
	pathItem.Set("put", operation2)

	// Set the path item with both operations
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_with_operation_id",
			yml: `
operationId: getUserById
description: Get user by ID
`,
		},
		{
			name: "valid_with_operation_ref",
			yml: `
operationRef: '#/paths/~1users~1{id}/get'
description: Reference to get user operation
`,
		},
		{
			name: "valid_with_parameters",
			yml: `
operationId: getUserById
parameters:
  id: '$response.body#/id'
  format: json
description: Get user with parameters
`,
		},
		{
			name: "valid_with_request_body",
			yml: `
operationId: updateUser
requestBody: '$response.body#/user'
description: Update user with request body
`,
		},
		{
			name: "valid_with_server",
			yml: `
operationId: getUserById
server:
  url: https://api.example.com/v2
  description: Version 2 API
description: Get user from v2 API
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
operationId: getUserById
description: Get user by ID
x-custom: value
x-timeout: 30
`,
		},
		{
			name: "valid_minimal_with_operation_id",
			yml: `
operationId: getUserById
`,
		},
		{
			name: "valid_minimal_with_operation_ref",
			yml: `
operationRef: '#/paths/~1users~1{id}/get'
`,
		},
		{
			name: "valid_no_operation_reference",
			yml: `
description: Link without operation reference
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var link openapi.Link

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &link)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := link.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestLink_Validate_Error(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations for operationId validation
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add paths with operations that match the operationIds used in tests
	pathItem := openapi.NewPathItem()

	// Add GET operation with getUserById
	operation1 := &openapi.Operation{
		OperationID: pointer.From("getUserById"),
	}
	pathItem.Set("get", operation1)

	// Add PUT operation with updateUser to the same path
	operation2 := &openapi.Operation{
		OperationID: pointer.From("updateUser"),
	}
	pathItem.Set("put", operation2)

	// Set the path item with both operations
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_both_operation_id_and_ref",
			yml: `
operationId: getUserById
operationRef: '#/paths/~1users~1{id}/get'
description: Invalid - both operationId and operationRef
`,
			wantErrs: []string{"operationID and operationRef are mutually exclusive"},
		},
		{
			name: "invalid_server",
			yml: `
operationId: getUserById
server:
  description: Invalid server without URL
description: Link with invalid server
`,
			wantErrs: []string{"[4:3] error validation-required-field `server.url` is required"},
		},
		{
			name: "invalid_operation_ref_uri",
			yml: `
operationRef: "http://[::1:bad"
description: Invalid operationRef URI
`,
			wantErrs: []string{"operationRef is not a valid uri: parse"},
		},
		{
			name: "invalid_parameter_expression_syntax",
			yml: `
operationId: getUserById
parameters:
  id: "$request.header."
description: Invalid parameter expression syntax - empty header name
`,
			wantErrs: []string{"header reference must be a valid token"},
		},
		{
			name: "invalid_request_body_expression_syntax",
			yml: `
operationId: updateUser
requestBody: "$request.query."
description: Invalid request body expression syntax - empty query name
`,
			wantErrs: []string{"query reference must be a valid name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var link openapi.Link

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &link)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := link.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			// Check that all expected errors are present
			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}

// Note: TestLink_Validate_OperationID_NotFound has been removed because operationId validation
// has been moved to the linter rule "semantic-link-operation" (see link_operation.go in linter/rules).
// This allows validation to occur after the index is built, enabling checks against operations
// in external documents that may be referenced later.

func TestLink_Validate_OperationID_Found(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add a path with an operation
	pathItem := openapi.NewPathItem()
	operation := &openapi.Operation{
		OperationID: pointer.From("getUserById"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	link := &openapi.Link{
		OperationID: pointer.From("getUserById"),
	}

	errs := link.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
	require.Empty(t, errs, "Expected no validation errors for existing operationId")
}

// Note: TestLink_Validate_OperationID_WithoutOpenAPIContext_Panics has been removed because
// operationId validation has been moved to the linter rule "semantic-link-operation".
// Link.Validate() no longer requires OpenAPI context for operationId validation.

func TestLink_Validate_ComplexExpressions(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add paths with operations that match the operationIds used in tests
	pathItem := openapi.NewPathItem()

	// Add GET operation with getUserById
	operation1 := &openapi.Operation{
		OperationID: pointer.From("getUserById"),
	}
	pathItem.Set("get", operation1)

	// Add PUT operation with updateUser to the same path
	operation2 := &openapi.Operation{
		OperationID: pointer.From("updateUser"),
	}
	pathItem.Set("put", operation2)

	// Set the path item with both operations
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_complex_parameter_expressions",
			yml: `
operationId: getUserById
parameters:
  id: '$response.body#/user/id'
  token: '$request.header.Authorization'
  query: '$request.query.filter'
  path: '$request.path.version'
description: Complex parameter expressions
`,
		},
		{
			name: "valid_complex_request_body_expression",
			yml: `
operationId: updateUser
requestBody: '$response.body#/user'
description: Complex request body expression
`,
		},
		{
			name: "valid_runtime_expressions",
			yml: `
operationId: getUserById
parameters:
  url: '$url'
  method: '$method'
  statusCode: '$statusCode'
description: Runtime expressions
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var link openapi.Link

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &link)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := link.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors for valid expressions")
		})
	}
}

func TestLink_Validate_NilParameters(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add a path with an operation
	pathItem := openapi.NewPathItem()
	operation := &openapi.Operation{
		OperationID: pointer.From("getUserById"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	link := &openapi.Link{
		OperationID: pointer.From("getUserById"),
		Parameters:  nil, // Explicitly nil
		RequestBody: nil, // Explicitly nil
		Server:      nil, // Explicitly nil
	}

	errs := link.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
	require.Empty(t, errs, "Expected no validation errors for nil parameters")
}

func TestLink_Validate_EmptyParameters(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add a path with an operation
	pathItem := openapi.NewPathItem()
	operation := &openapi.Operation{
		OperationID: pointer.From("getUserById"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	yml := `
operationId: getUserById
parameters: {}
description: Empty parameters map
`
	var link openapi.Link

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &link)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	errs := link.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
	require.Empty(t, errs, "Expected no validation errors for empty parameters")
}

func TestLink_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `
operationId: getUserById
operationRef: '#/paths/~1users~1{id}/get'
description: Get user by ID
parameters:
  id: '$response.body#/id'
requestBody: '$response.body#/user'
server:
  url: https://api.example.com
x-custom: value
`
	var link openapi.Link

	_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &link)
	require.NoError(t, err)

	require.Equal(t, "getUserById", link.GetOperationID(), "GetOperationID should return correct value")
	require.Equal(t, "#/paths/~1users~1{id}/get", link.GetOperationRef(), "GetOperationRef should return correct value")
	require.Equal(t, "Get user by ID", link.GetDescription(), "GetDescription should return correct value")
	require.NotNil(t, link.GetParameters(), "GetParameters should return non-nil")
	require.NotNil(t, link.GetRequestBody(), "GetRequestBody should return non-nil")
	require.NotNil(t, link.GetServer(), "GetServer should return non-nil")
	require.NotNil(t, link.GetExtensions(), "GetExtensions should return non-nil")
	require.Equal(t, "https://api.example.com", link.GetServer().GetURL(), "GetServer should return correct URL")
}

func TestLink_Getters_NilLink(t *testing.T) {
	t.Parallel()

	var link *openapi.Link

	require.Empty(t, link.GetOperationID(), "GetOperationID should return empty string for nil")
	require.Empty(t, link.GetOperationRef(), "GetOperationRef should return empty string for nil")
	require.Empty(t, link.GetDescription(), "GetDescription should return empty string for nil")
	require.Nil(t, link.GetParameters(), "GetParameters should return nil for nil link")
	require.Nil(t, link.GetRequestBody(), "GetRequestBody should return nil for nil link")
	require.Nil(t, link.GetServer(), "GetServer should return nil for nil link")
	require.NotNil(t, link.GetExtensions(), "GetExtensions should return empty extensions for nil link")
}

func TestLink_ResolveOperation(t *testing.T) {
	t.Parallel()

	link := &openapi.Link{
		OperationID: pointer.From("getUserById"),
	}

	op, err := link.ResolveOperation(t.Context())
	require.NoError(t, err, "ResolveOperation should not error")
	require.Nil(t, op, "ResolveOperation returns nil for now (TODO)")
}
