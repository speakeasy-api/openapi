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
			wantErrs: []string{"openapi field openapi invalid OpenAPI version invalid-version"},
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
			wantErrs: []string{"only OpenAPI version 3.1.1 and below is supported"},
		},
		{
			name: "invalid_info_missing_title",
			yml: `
openapi: 3.1.0
info:
  version: 1.0.0
paths: {}
`,
			wantErrs: []string{"field title is missing"},
		},
		{
			name: "invalid_info_missing_version",
			yml: `
openapi: 3.1.0
info:
  title: Test API
paths: {}
`,
			wantErrs: []string{"field version is missing"},
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
			wantErrs: []string{"field url is missing"},
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
			wantErrs: []string{"field name is missing"},
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
			wantErrs: []string{"field url is missing"},
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
