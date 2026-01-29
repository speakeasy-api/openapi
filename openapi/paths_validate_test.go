package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/require"
)

func TestPaths_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yml            string
		openApiVersion string
	}{
		{
			name: "valid_empty_paths",
			yml:  `{}`,
		},
		{
			name: "valid_single_path",
			yml: `
/users:
  get:
    summary: List users
    responses:
      '200':
        description: Successful response
`,
		},
		{
			name: "valid_multiple_paths",
			yml: `
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
`,
		},
		{
			name: "valid_paths_with_extensions",
			yml: `
/users:
  get:
    summary: List users
    responses:
      '200':
        description: Successful response
x-custom: value
x-another: 123
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var paths openapi.Paths

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &paths)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			openAPIDoc := openapi.NewOpenAPI()
			if tt.openApiVersion != "" {
				openAPIDoc.OpenAPI = tt.openApiVersion
			}

			errs := paths.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestPathItem_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yml            string
		openApiVersion string
	}{
		{
			name: "valid_get_operation",
			yml: `
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
`,
		},
		{
			name: "valid_multiple_operations",
			yml: `
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
post:
  summary: Create resource
  requestBody:
    content:
      application/json:
        schema:
          type: object
  responses:
    '201':
      description: Resource created
put:
  summary: Update resource
  responses:
    '200':
      description: Resource updated
delete:
  summary: Delete resource
  responses:
    '204':
      description: Resource deleted
`,
		},
		{
			name: "valid_with_summary_and_description",
			yml: `
summary: User operations
description: Operations for managing users
get:
  summary: Get user
  responses:
    '200':
      description: Successful response
`,
		},
		{
			name: "valid_with_servers",
			yml: `
servers:
  - url: https://api.example.com/v1
    description: Production server
  - url: https://staging-api.example.com/v1
    description: Staging server
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
`,
		},
		{
			name: "valid_with_parameters",
			yml: `
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
  summary: Get resource
  responses:
    '200':
      description: Successful response
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
x-custom: value
x-rate-limit: 100
`,
		},
		{
			name: "valid_all_http_methods",
			yml: `
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
put:
  summary: Update resource
  responses:
    '200':
      description: Resource updated
post:
  summary: Create resource
  responses:
    '201':
      description: Resource created
delete:
  summary: Delete resource
  responses:
    '204':
      description: Resource deleted
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
  summary: Patch resource
  responses:
    '200':
      description: Resource patched
trace:
  summary: Trace request
  responses:
    '200':
      description: Trace response
`,
		},
		{
			name: "valid_additional_operation",
			yml: `
additionalOperations:
  COPY:
    summary: Copy operation
    description: Custom COPY operation for the test endpoint
    operationId: copyTest
    tags:
      - test
    responses:
      201:
        description: Created
    x-test: some-value
        `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var pathItem openapi.PathItem

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &pathItem)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			openAPIDoc := openapi.NewOpenAPI()
			if tt.openApiVersion != "" {
				openAPIDoc.OpenAPI = tt.openApiVersion
			}

			errs := pathItem.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestPathItem_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yml            string
		openApiVersion string
		wantErrs       []string
	}{
		{
			name: "invalid_server",
			yml: `
servers:
  - description: Invalid server
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
`,
			wantErrs: []string{"[3:5] error validation-required-field server.url is required"},
		},
		{
			name: "invalid_parameter",
			yml: `
parameters:
  - in: query
    schema:
      type: string
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
`,
			wantErrs: []string{"[3:5] error validation-required-field parameter.name is required"},
		},
		{
			name:           "unexpected_additional_operations",
			openApiVersion: "3.1.2",
			yml: `
additionalOperations:
  COPY:
    summary: Copy operation
    description: Custom COPY operation for the test endpoint
    operationId: copyTest
    tags:
      - test
    responses:
      201:
        description: Created
    x-test: some-value
        `,
			wantErrs: []string{"additionalOperations is not supported in OpenAPI version 3.1.2"},
		},
		{
			name:           "standard_method_in_additional_operations",
			openApiVersion: "3.2.0",
			yml: `
additionalOperations:
  GET:
    summary: Get operation
    description: Custom GET operation for the test endpoint
    operationId: getTest
    tags:
      - test
    responses:
      200:
        description: Successful response
    x-test: some-value
        `,
			wantErrs: []string{"method [GET] is a standardized HTTP method and must be defined in its own field"},
		},
		{
			name:           "invalid_openapi_version",
			openApiVersion: "invalid-version",
			yml: `
get:
  summary: Get resource
  responses:
    '200':
      description: Successful response
      `,
			wantErrs: []string{"invalid version invalid-version"},
		},
		{
			name:           "not_using_additional_operations_for_non_standard_method",
			openApiVersion: "3.2.0",
			yml: `
copy:
  summary: Copy resource
  responses:
      '201':
        description: Resource copied
      `,
			wantErrs: []string{"method [copy] is not a standardized HTTP method and must be listed under additionalOperations"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var pathItem openapi.PathItem

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &pathItem)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			openAPIDoc := openapi.NewOpenAPI()
			if tt.openApiVersion != "" {
				openAPIDoc.OpenAPI = tt.openApiVersion
			}

			validateErrs := pathItem.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
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
