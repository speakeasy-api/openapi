package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid response with description only",
			yml: `
description: Success response
`,
		},
		{
			name: "valid response with content",
			yml: `
description: User data response
content:
  application/json:
    schema:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
  application/xml:
    schema:
      type: object
`,
		},
		{
			name: "valid response with headers",
			yml: `
description: Response with headers
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
`,
		},
		{
			name: "valid response with links",
			yml: `
description: Response with links
content:
  application/json:
    schema:
      type: object
links:
  GetUserByUserId:
    operationId: getUserById
    parameters:
      userId: $response.body#/id
  GetUserAddresses:
    operationRef: "#/paths/~1users~1{userId}~1addresses/get"
    parameters:
      userId: $response.body#/id
`,
		},
		{
			name: "valid response with extensions",
			yml: `
description: Response with extensions
content:
  application/json:
    schema:
      type: string
x-test: some-value
x-custom: custom-data
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var response openapi.Response
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &response)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create a minimal OpenAPI document for operationId validation
			var opts []validation.Option
			if tt.name == "valid response with links" {
				// Create OpenAPI document with the required operationId for link validation
				openAPIDoc := &openapi.OpenAPI{
					Paths: openapi.NewPaths(),
				}

				// Add path with operation that matches the operationId in the test
				pathItem := openapi.NewPathItem()
				operation := &openapi.Operation{
					OperationID: pointer.From("getUserById"),
				}
				pathItem.Set("get", operation)
				openAPIDoc.Paths.Set("/users/{id}", &openapi.ReferencedPathItem{Object: pathItem})

				opts = append(opts, validation.WithContextObject(openAPIDoc))
			}

			errs := response.Validate(t.Context(), opts...)
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, response.Valid, "expected response to be valid")
		})
	}
}

func TestResponse_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing description",
			yml: `
content:
  application/json:
    schema:
      type: object
`,
			wantErrs: []string{"[2:1] error validation-required-field response.description is required"},
		},
		{
			name: "empty description",
			yml: `
description: ""
content:
  application/json:
    schema:
      type: object
`,
			wantErrs: []string{"[2:14] error validation-required-field response.description is required"},
		},
		{
			name: "invalid schema in content",
			yml: `
description: Response with invalid schema
content:
  application/json:
    schema:
      type: invalid-type
`,
			wantErrs: []string{
				"[6:13] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
				"[6:13] error validation-type-mismatch schema.type expected array, got string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var response openapi.Response

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &response)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := response.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}

func TestResponses_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid responses with status codes",
			yml: `
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
`,
		},
		{
			name: "valid responses with default",
			yml: `
"200":
  description: Success
default:
  description: Default response
  content:
    application/json:
      schema:
        type: object
`,
		},
		{
			name: "valid responses with extensions",
			yml: `
"200":
  description: Success
x-test: some-value
`,
		},
		{
			name: "valid responses with only default",
			yml: `
default:
  description: Default response for all status codes
  content:
    application/json:
      schema:
        type: object
        properties:
          message:
            type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var responses openapi.Responses
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &responses)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := responses.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, responses.Valid, "expected responses to be valid")
		})
	}
}

func TestResponses_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid response in responses",
			yml: `
"200":
  description: ""
"404":
  description: Not found
`,
			wantErrs: []string{"error validation-required-field response.description is required"},
		},
		{
			name: "no response codes",
			yml: `
x-test: some-value
`,
			wantErrs: []string{"error validation-allowed-values responses must have at least one response code"},
		},
		{
			name:     "empty responses object",
			yml:      `{}`,
			wantErrs: []string{"error validation-allowed-values responses must have at least one response code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var responses openapi.Responses
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &responses)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := responses.Validate(t.Context())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, responses.Valid, "expected responses to be invalid")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range errs {
				errMessages = append(errMessages, err.Error())
			}

			for _, expectedErr := range tt.wantErrs {
				found := false
				for _, errMsg := range errMessages {
					if strings.Contains(errMsg, expectedErr) {
						found = true
						break
					}
				}
				require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
			}
		})
	}
}
