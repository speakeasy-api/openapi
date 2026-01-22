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

func TestOwaspNoHttpBasicRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
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
			name: "oauth2 authentication",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth/authorize
          scopes:
            read: Read access
paths: {}
`,
		},
		{
			name: "api key authentication",
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

			rule := &rules.OwaspNoHttpBasicRule{}
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

func TestOwaspNoHttpBasicRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "basic authentication",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic
paths: {}
`,
			expectedCount: 1,
			expectedText:  "basic",
		},
		{
			name: "negotiate authentication",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    negotiateAuth:
      type: http
      scheme: negotiate
paths: {}
`,
			expectedCount: 1,
			expectedText:  "negotiate",
		},
		{
			name: "multiple insecure schemes",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic
    negotiateAuth:
      type: http
      scheme: negotiate
paths: {}
`,
			expectedCount: 2,
			expectedText:  "",
		},
		{
			name: "basic with uppercase",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: Basic
paths: {}
`,
			expectedCount: 1,
			expectedText:  "Basic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoHttpBasicRule{}
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
				assert.Contains(t, err.Error(), "insecure")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspNoHttpBasicRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspNoHttpBasicRule{}

	assert.Equal(t, "owasp-no-http-basic", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
