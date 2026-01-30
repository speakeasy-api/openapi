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

func TestOwaspNoAPIKeysInURLRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "api key in header",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
paths: {}
`,
		},
		{
			name: "api key in cookie",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: cookie
      name: api_key
paths: {}
`,
		},
		{
			name: "bearer authentication",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
paths: {}
`,
		},
		{
			name: "no security schemes",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoAPIKeysInURLRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOwaspNoAPIKeysInURLRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "api key in query parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: query
      name: api_key
paths: {}
`,
			expectedCount: 1,
			expectedText:  "query",
		},
		{
			name: "api key in path parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: path
      name: api_key
paths: {}
`,
			expectedCount: 1,
			expectedText:  "path",
		},
		{
			name: "multiple api keys in url",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    apiKeyQuery:
      type: apiKey
      in: query
      name: api_key
    apiKeyPath:
      type: apiKey
      in: path
      name: key
paths: {}
`,
			expectedCount: 2,
			expectedText:  "",
		},
		{
			name: "mixed secure and insecure api keys",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    apiKeyHeader:
      type: apiKey
      in: header
      name: X-API-Key
    apiKeyQuery:
      type: apiKey
      in: query
      name: api_key
paths: {}
`,
			expectedCount: 1,
			expectedText:  "query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoAPIKeysInURLRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, tt.expectedCount)
			for _, err := range errs {
				assert.Contains(t, err.Error(), "API key via URL")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspNoAPIKeysInURLRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspNoAPIKeysInURLRule{}

	assert.Equal(t, "owasp-no-api-keys-in-url", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
