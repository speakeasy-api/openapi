package openapi_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPI_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "minimal OpenAPI document",
			yaml: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}`,
		},
		{
			name: "OpenAPI document with servers",
			yaml: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
    description: Production server
paths: {}`,
		},
		{
			name: "OpenAPI document with tags",
			yaml: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
paths: {}`,
		},
		{
			name: "OpenAPI document with security",
			yaml: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
security:
  - ApiKeyAuth: []
paths: {}
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key`,
		},
		{
			name: "OpenAPI document with external docs",
			yaml: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
externalDocs:
  url: https://example.com/docs
  description: API Documentation
paths: {}`,
		},
		{
			name: "OpenAPI document with extensions",
			yaml: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
x-custom-extension: custom-value
paths: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)
			require.Empty(t, validationErrs)
			require.NotNil(t, doc)

			// Basic structure validation
			assert.Equal(t, "3.1.0", doc.OpenAPI)
			assert.Equal(t, "Test API", doc.Info.Title)
			assert.Equal(t, "1.0.0", doc.Info.Version)
			assert.NotNil(t, doc.Paths)
		})
	}
}

func TestOpenAPI_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "missing openapi field",
			yaml: `info:
  title: Test API
  version: 1.0.0
paths: {}`,
			expectedError: "field openapi is missing",
		},
		{
			name: "missing info field",
			yaml: `openapi: 3.1.0
paths: {}`,
			expectedError: "field info is missing",
		},
		{
			name: "invalid openapi version",
			yaml: `openapi: 2.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`,
			expectedError: fmt.Sprintf("openapi field only OpenAPI versions between %s and %s are supported", openapi.MinimumSupportedVersion(), openapi.MaximumSupportedVersion()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))

			if tt.expectedError != "" {
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					require.NotEmpty(t, validationErrs, "Expected validation errors but got none")
					found := false
					for _, validationErr := range validationErrs {
						if assert.Contains(t, validationErr.Error(), tt.expectedError) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error message not found in validation errors")
				}
			}

			// Document might still be created even with validation errors
			if doc != nil {
				assert.NotNil(t, doc)
			}
		})
	}
}
