package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAS3ParameterDescriptionRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "path parameter with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          description: The user ID
          schema:
            type: string
`,
		},
		{
			name: "query parameter with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: limit
          in: query
          description: Maximum number of results
          schema:
            type: integer
`,
		},
		{
			name: "no parameters",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3ParameterDescriptionRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs, "should have no lint errors")
		})
	}
}

func TestOAS3ParameterDescriptionRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "path parameter missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
`,
			expectedError: "parameter `id` in `get /users/{id}` is missing a description",
		},
		{
			name: "query parameter missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
`,
			expectedError: "parameter `limit` in `get /users` is missing a description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3ParameterDescriptionRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			require.NotEmpty(t, errs, "should have lint errors")
			assert.Contains(t, errs[0].Error(), tt.expectedError, "error message should contain expected text")
		})
	}
}

func TestOAS3ParameterDescriptionRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OAS3ParameterDescriptionRule{}

	assert.Equal(t, "style-oas3-parameter-description", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.NotNil(t, rule.Versions())
}
