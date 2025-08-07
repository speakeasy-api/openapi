package openapi_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
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
		OperationID: stringPtr("getUserById"),
	}
	pathItem.Set("get", operation1)

	// Add PUT operation with updateUser to the same path
	operation2 := &openapi.Operation{
		OperationID: stringPtr("updateUser"),
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

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &link)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
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
		OperationID: stringPtr("getUserById"),
	}
	pathItem.Set("get", operation1)

	// Add PUT operation with updateUser to the same path
	operation2 := &openapi.Operation{
		OperationID: stringPtr("updateUser"),
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
			wantErrs: []string{"field url is missing"},
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
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &link)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
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

func TestLink_Validate_OperationID_NotFound(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add a path with an operation
	pathItem := openapi.NewPathItem()
	operation := &openapi.Operation{
		OperationID: stringPtr("existingOperation"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	link := &openapi.Link{
		OperationID: stringPtr("nonExistentOperation"),
	}

	errs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
	require.NotEmpty(t, errs, "Expected validation error for non-existent operationId")
	require.Contains(t, errs[0].Error(), "operationId nonExistentOperation does not exist in document")
}

func TestLink_Validate_OperationID_Found(t *testing.T) {
	t.Parallel()

	// Create a minimal OpenAPI document with operations
	openAPIDoc := &openapi.OpenAPI{
		Paths: openapi.NewPaths(),
	}

	// Add a path with an operation
	pathItem := openapi.NewPathItem()
	operation := &openapi.Operation{
		OperationID: stringPtr("getUserById"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	link := &openapi.Link{
		OperationID: stringPtr("getUserById"),
	}

	errs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
	require.Empty(t, errs, "Expected no validation errors for existing operationId")
}

func TestLink_Validate_OperationID_WithoutOpenAPIContext_Panics(t *testing.T) {
	t.Parallel()

	link := &openapi.Link{
		OperationID: stringPtr("getUserById"),
	}

	require.Panics(t, func() {
		link.Validate(context.Background())
	}, "Expected panic when validating operationId without OpenAPI context")
}

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
		OperationID: stringPtr("getUserById"),
	}
	pathItem.Set("get", operation1)

	// Add PUT operation with updateUser to the same path
	operation2 := &openapi.Operation{
		OperationID: stringPtr("updateUser"),
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

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &link)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
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
		OperationID: stringPtr("getUserById"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	link := &openapi.Link{
		OperationID: stringPtr("getUserById"),
		Parameters:  nil, // Explicitly nil
		RequestBody: nil, // Explicitly nil
		Server:      nil, // Explicitly nil
	}

	errs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
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
		OperationID: stringPtr("getUserById"),
	}
	pathItem.Set("get", operation)
	openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

	yml := `
operationId: getUserById
parameters: {}
description: Empty parameters map
`
	var link openapi.Link

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &link)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	errs := link.Validate(context.Background(), validation.WithContextObject(openAPIDoc))
	require.Empty(t, errs, "Expected no validation errors for empty parameters")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
