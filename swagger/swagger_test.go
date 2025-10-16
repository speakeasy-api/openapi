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
