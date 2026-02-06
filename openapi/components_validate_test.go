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

func TestComponents_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_empty_components",
			yml:  `{}`,
		},
		{
			name: "valid_components_with_schemas",
			yml: `
schemas:
  User:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
  Error:
    type: object
    properties:
      code:
        type: integer
      message:
        type: string
`,
		},
		{
			name: "valid_components_with_responses",
			yml: `
responses:
  NotFound:
    description: The specified resource was not found
  Unauthorized:
    description: Unauthorized
`,
		},
		{
			name: "valid_components_with_parameters",
			yml: `
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
`,
		},
		{
			name: "valid_components_with_examples",
			yml: `
examples:
  user-example:
    summary: User Example
    value:
      id: 1
      name: John Doe
`,
		},
		{
			name: "valid_components_with_request_bodies",
			yml: `
requestBodies:
  UserArray:
    description: user to add to the system
    content:
      application/json:
        schema:
          type: array
          items:
            type: object
`,
		},
		{
			name: "valid_components_with_headers",
			yml: `
headers:
  X-Rate-Limit-Limit:
    description: The number of allowed requests in the current period
    schema:
      type: integer
`,
		},
		{
			name: "valid_components_with_security_schemes",
			yml: `
securitySchemes:
  ApiKeyAuth:
    type: apiKey
    in: header
    name: X-API-Key
  BearerAuth:
    type: http
    scheme: bearer
`,
		},
		{
			name: "valid_components_with_links",
			yml: `
links:
  UserRepositories:
    operationId: getRepositoriesByOwner
    parameters:
      username: $response.body#/login
`,
		},
		{
			name: "valid_components_with_callbacks",
			yml: `
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
        responses:
          '200':
            description: webhook successfully processed
`,
		},
		{
			name: "valid_components_with_path_items",
			yml: `
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
`,
		},
		{
			name: "valid_components_with_extensions",
			yml: `
schemas:
  User:
    type: object
    properties:
      id:
        type: integer
x-custom: value
x-another: 123
`,
		},
		{
			name: "valid_components_with_multiple_sections",
			yml: `
schemas:
  User:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
responses:
  NotFound:
    description: The specified resource was not found
parameters:
  limitParam:
    name: limit
    in: query
    schema:
      type: integer
securitySchemes:
  ApiKeyAuth:
    type: apiKey
    in: header
    name: X-API-Key
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var components openapi.Components

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &components)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			openAPIDoc := openapi.NewOpenAPI()
			if tt.name == "valid_components_with_links" {
				// Create OpenAPI document with the required operationId for link validation
				openAPIDoc.Paths = openapi.NewPaths()

				// Add path with operation that matches the operationId in the test
				pathItem := openapi.NewPathItem()
				operation := &openapi.Operation{
					OperationID: pointer.From("getRepositoriesByOwner"),
				}
				pathItem.Set("get", operation)
				openAPIDoc.Paths.Set("/users/{username}/repos", &openapi.ReferencedPathItem{Object: pathItem})
			}

			errs := components.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestComponents_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_security_scheme",
			yml: `
securitySchemes:
  InvalidScheme:
    description: Some scheme
`,
			wantErrs: []string{"[4:5] error validation-required-field `securityScheme.type` is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var components openapi.Components

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &components)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := components.Validate(t.Context())
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
