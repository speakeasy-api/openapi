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

func TestOwaspJWTBestPracticesRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "oauth2 with RFC8725 in description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      description: OAuth2 authentication supporting RFC8725
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth
          scopes:
            read: Read access
paths: {}
`,
		},
		{
			name: "jwt bearer with RFC8725 in description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT bearer token supporting RFC8725
paths: {}
`,
		},
		{
			name: "non-jwt bearer without RFC8725 is ok",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
      description: Bearer token authentication
paths: {}
`,
		},
		{
			name: "api key without RFC8725 is ok",
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
      description: API key authentication
paths: {}
`,
		},
		{
			name: "no security schemes is ok",
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

			rule := &rules.OwaspJWTBestPracticesRule{}
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

func TestOwaspJWTBestPracticesRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "oauth2 without RFC8725",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      description: OAuth2 authentication
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth
          scopes:
            read: Read access
paths: {}
`,
			expectedCount: 1,
			expectedText:  "oauth",
		},
		{
			name: "jwt bearer without RFC8725",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT bearer token
paths: {}
`,
			expectedCount: 1,
			expectedText:  "bearer",
		},
		{
			name: "oauth2 with no description",
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
          authorizationUrl: https://example.com/oauth
          scopes:
            read: Read access
paths: {}
`,
			expectedCount: 1,
			expectedText:  "RFC8725",
		},
		{
			name: "multiple jwt schemes without RFC8725",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      description: OAuth2 authentication
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth
          scopes:
            read: Read access
    bearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT bearer token
paths: {}
`,
			expectedCount: 2,
			expectedText:  "",
		},
		{
			name: "mixed schemes one with one without RFC8725",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      description: OAuth2 authentication supporting RFC8725
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth
          scopes:
            read: Read access
    bearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT bearer token
paths: {}
`,
			expectedCount: 1,
			expectedText:  "bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspJWTBestPracticesRule{}
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
				assert.Contains(t, err.Error(), "RFC8725")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspJWTBestPracticesRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspJWTBestPracticesRule{}

	assert.Equal(t, "owasp-jwt-best-practices", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0", "3.1"}, rule.Versions())
}
