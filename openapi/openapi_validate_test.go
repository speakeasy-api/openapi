package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestOpenAPI_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_minimal_3_1_0",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
		},
		{
			name: "valid_minimal_3_0_3",
			yml: `
openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
		},
		{
			name: "valid_with_servers",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
    description: Production server
  - url: https://staging-api.example.com/v1
    description: Staging server
paths: {}
`,
		},
		{
			name: "valid_with_tags",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
  - name: orders
    description: Order operations
paths: {}
`,
		},
		{
			name: "valid_with_external_docs",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
externalDocs:
  description: Find more info here
  url: https://example.com/docs
paths: {}
`,
		},
		{
			name: "valid_with_json_schema_dialect",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
jsonSchemaDialect: https://json-schema.org/draft/2020-12/schema
paths: {}
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
x-custom: value
x-api-version: 2.0
`,
		},
		{
			name: "valid_with_self_absolute_uri",
			yml: `
openapi: 3.2.0
$self: https://example.com/api/openapi.yaml
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
		},
		{
			name: "valid_with_self_relative_uri",
			yml: `
openapi: 3.2.0
$self: /api/openapi.yaml
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
		},
		{
			name: "valid_complete",
			yml: `
openapi: 3.1.0
info:
  title: Complete Test API
  version: 1.0.0
  description: A complete API example
externalDocs:
  description: API Documentation
  url: https://example.com/docs
servers:
  - url: https://api.example.com/v1
    description: Production server
tags:
  - name: users
    description: User operations
security:
  - ApiKeyAuth: []
paths:
  /users:
    get:
      summary: List users
      responses:
        '200':
          description: Successful response
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
jsonSchemaDialect: https://json-schema.org/draft/2020-12/schema
x-custom: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc openapi.OpenAPI

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := doc.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestOpenAPI_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_openapi_version_format",
			yml: `
openapi: invalid-version
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
			wantErrs: []string{"error validation-supported-version openapi.openapi invalid OpenAPI version invalid-version"},
		},
		{
			name: "unsupported_openapi_version",
			yml: `
openapi: 4.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
			wantErrs: []string{"error validation-supported-version openapi.openapi only OpenAPI versions between"},
		},
		{
			name: "invalid_info_missing_title",
			yml: `
openapi: 3.1.0
info:
  version: 1.0.0
paths: {}
`,
			wantErrs: []string{"[4:3] error validation-required-field info.title is required"},
		},
		{
			name: "invalid_info_missing_version",
			yml: `
openapi: 3.1.0
info:
  title: Test API
paths: {}
`,
			wantErrs: []string{"[4:3] error validation-required-field info.version is required"},
		},
		{
			name: "invalid_server",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - description: Invalid server without URL
paths: {}
`,
			wantErrs: []string{"[7:5] error validation-required-field server.url is required"},
		},
		{
			name: "invalid_tag",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
tags:
  - description: Tag without name
paths: {}
`,
			wantErrs: []string{"[7:5] error validation-required-field tag.name is required"},
		},
		{
			name: "invalid_external_docs",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
externalDocs:
  description: External docs without URL
paths: {}
`,
			wantErrs: []string{"[7:3] error validation-required-field externalDocumentation.url is required"},
		},
		{
			name: "invalid_self_not_uri",
			yml: `
openapi: 3.2.0
$self: "ht!tp://invalid-scheme"
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
			wantErrs: []string{"error validation-invalid-format openapi.$self is not a valid uri reference"},
		},
		{
			name: "duplicate_operation_id",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /melody:
    post:
      operationId: littleSong
      responses:
        '200':
          description: ok
  /ember:
    get:
      operationId: littleSong
      responses:
        '200':
          description: ok
`,
			wantErrs: []string{"error validation-operation-id-unique the 'get' operation at path '/ember' contains a duplicate operationId 'littleSong'"},
		},
		{
			name: "duplicate_operation_parameter",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema:
            type: string
        - in: path
          name: id
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
			wantErrs: []string{"error validation-operation-parameters parameter \"id\" is duplicated in GET operation at path \"/users/{id}\""},
		},
		{
			name: "duplicate_pathitem_parameter",
			yml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{id}:
    parameters:
      - in: path
        name: id
        required: true
        schema:
          type: string
      - in: path
        name: id
        required: true
        schema:
          type: string
    get:
      responses:
        '200':
          description: ok
`,
			wantErrs: []string{"error validation-operation-parameters parameter \"id\" is duplicated in path \"/users/{id}\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc openapi.OpenAPI

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := doc.Validate(t.Context())
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
