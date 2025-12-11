package swagger_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "minimal valid swagger document",
			yaml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`,
		},
		{
			name: "swagger with host and basePath",
			yaml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
host: api.example.com
basePath: /v1
paths: {}`,
		},
		{
			name: "swagger with schemes and consumes/produces",
			yaml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
schemes:
  - https
  - http
consumes:
  - application/json
produces:
  - application/json
paths: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, validationErrs, err := swagger.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should have no validation errors")
			require.NotNil(t, doc, "document should not be nil")
			assert.Equal(t, "2.0", doc.Swagger, "swagger version should be 2.0")
		})
	}
}

func TestUnmarshal_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "missing swagger field",
			yaml: `info:
  title: Test API
  version: 1.0.0
paths: {}`,
			expectedError: "swagger is missing",
		},
		{
			name: "missing info field",
			yaml: `swagger: "2.0"
paths: {}`,
			expectedError: "info is missing",
		},
		{
			name: "missing paths field",
			yaml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0`,
			expectedError: "paths is missing",
		},
		{
			name: "missing info.title",
			yaml: `swagger: "2.0"
info:
  version: 1.0.0
paths: {}`,
			expectedError: "info.title is missing",
		},
		{
			name: "missing info.version",
			yaml: `swagger: "2.0"
info:
  title: Test API
paths: {}`,
			expectedError: "info.version is missing",
		},
		{
			name: "invalid swagger version",
			yaml: `swagger: "3.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`,
			expectedError: "swagger must be '2.0'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, validationErrs, err := swagger.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should not return error")
			require.NotNil(t, doc, "document should not be nil")
			require.NotEmpty(t, validationErrs, "should have validation errors")

			found := false
			var allErrors []string
			for _, verr := range validationErrs {
				allErrors = append(allErrors, verr.Error())
				if strings.Contains(verr.Error(), tt.expectedError) {
					found = true
					break
				}
			}
			assert.True(t, found, "should contain expected error: %s\nGot errors: %v", tt.expectedError, allErrors)
		})
	}
}

func TestMarshal_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := &swagger.Swagger{
		Swagger: swagger.Version,
		Info: swagger.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: swagger.NewPaths(),
	}

	var buf strings.Builder
	err := swagger.Marshal(ctx, doc, &buf)
	require.NoError(t, err, "marshal should succeed")

	expected := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	assert.Equal(t, expected, buf.String(), "marshaled output should match expected YAML")
}

func TestSwagger_Getters_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
host: api.example.com
basePath: /v1
schemes:
  - https
consumes:
  - application/json
produces:
  - application/xml
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
definitions:
  User:
    type: object
parameters:
  limitParam:
    name: limit
    in: query
    type: integer
responses:
  NotFound:
    description: Not found
securityDefinitions:
  api_key:
    type: apiKey
    name: X-API-Key
    in: header
security:
  - api_key: []
tags:
  - name: users
    description: User operations
externalDocs:
  description: External docs
  url: https://example.com/docs
x-custom: value
`
	doc, validationErrs, err := swagger.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	assert.Equal(t, "2.0", doc.GetSwagger(), "GetSwagger should return 2.0")
	assert.NotNil(t, doc.GetInfo(), "GetInfo should return non-nil Info")
	assert.Equal(t, "Test API", doc.GetInfo().Title, "GetInfo should return correct Info")
	assert.Equal(t, "api.example.com", doc.GetHost(), "GetHost should return correct value")
	assert.Equal(t, "/v1", doc.GetBasePath(), "GetBasePath should return correct value")
	assert.Equal(t, []string{"https"}, doc.GetSchemes(), "GetSchemes should return correct value")
	assert.Equal(t, []string{"application/json"}, doc.GetConsumes(), "GetConsumes should return correct value")
	assert.Equal(t, []string{"application/xml"}, doc.GetProduces(), "GetProduces should return correct value")
	assert.NotNil(t, doc.GetPaths(), "GetPaths should return non-nil")
	assert.NotNil(t, doc.GetDefinitions(), "GetDefinitions should return non-nil")
	assert.NotNil(t, doc.GetParameters(), "GetParameters should return non-nil")
	assert.NotNil(t, doc.GetResponses(), "GetResponses should return non-nil")
	assert.NotNil(t, doc.GetSecurityDefinitions(), "GetSecurityDefinitions should return non-nil")
	assert.NotNil(t, doc.GetSecurity(), "GetSecurity should return non-nil")
	assert.NotNil(t, doc.GetTags(), "GetTags should return non-nil")
	assert.NotNil(t, doc.GetExternalDocs(), "GetExternalDocs should return non-nil")
	assert.NotNil(t, doc.GetExtensions(), "GetExtensions should return non-nil")
}

func TestSwagger_Getters_Nil(t *testing.T) {
	t.Parallel()

	var doc *swagger.Swagger

	assert.Empty(t, doc.GetSwagger(), "GetSwagger should return empty string for nil")
	assert.Nil(t, doc.GetInfo(), "GetInfo should return nil for nil doc")
	assert.Empty(t, doc.GetHost(), "GetHost should return empty string for nil")
	assert.Empty(t, doc.GetBasePath(), "GetBasePath should return empty string for nil")
	assert.Nil(t, doc.GetSchemes(), "GetSchemes should return nil for nil doc")
	assert.Nil(t, doc.GetConsumes(), "GetConsumes should return nil for nil doc")
	assert.Nil(t, doc.GetProduces(), "GetProduces should return nil for nil doc")
	assert.Nil(t, doc.GetPaths(), "GetPaths should return nil for nil doc")
	assert.Nil(t, doc.GetDefinitions(), "GetDefinitions should return nil for nil doc")
	assert.Nil(t, doc.GetParameters(), "GetParameters should return nil for nil doc")
	assert.Nil(t, doc.GetResponses(), "GetResponses should return nil for nil doc")
	assert.Nil(t, doc.GetSecurityDefinitions(), "GetSecurityDefinitions should return nil for nil doc")
	assert.Nil(t, doc.GetSecurity(), "GetSecurity should return nil for nil doc")
	assert.Nil(t, doc.GetTags(), "GetTags should return nil for nil doc")
	assert.Nil(t, doc.GetExternalDocs(), "GetExternalDocs should return nil for nil doc")
	assert.NotNil(t, doc.GetExtensions(), "GetExtensions should return empty extensions for nil doc")
}
