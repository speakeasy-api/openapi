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

func TestOwaspProtectionGlobalUnsafeStrictRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "global security protects all operations",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
security:
  - apiKey: []
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
paths:
  /users:
    post:
      responses:
        '201':
          description: Created
    delete:
      responses:
        '204':
          description: Deleted
`,
		},
		{
			name: "operation-level security",
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
paths:
  /users:
    post:
      security:
        - apiKey: []
      responses:
        '201':
          description: Created
`,
		},
		{
			name: "safe methods dont require security",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "mixed global and operation security",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
security:
  - apiKey: []
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth
          scopes:
            write: Write access
paths:
  /users:
    post:
      responses:
        '201':
          description: Created
    put:
      security:
        - oauth: [write]
      responses:
        '200':
          description: Updated
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspProtectionGlobalUnsafeStrictRule{}
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

func TestOwaspProtectionGlobalUnsafeStrictRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "post without security",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    post:
      responses:
        '201':
          description: Created
`,
			expectedCount: 1,
			expectedText:  "post",
		},
		{
			name: "empty security array not allowed in strict mode",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /public:
    post:
      security: []
      responses:
        '201':
          description: Created
`,
			expectedCount: 1,
			expectedText:  "post",
		},
		{
			name: "delete without security",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    delete:
      responses:
        '204':
          description: Deleted
`,
			expectedCount: 1,
			expectedText:  "delete",
		},
		{
			name: "multiple unsafe operations without security",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    post:
      responses:
        '201':
          description: Created
    put:
      responses:
        '200':
          description: Updated
    patch:
      responses:
        '200':
          description: Patched
    delete:
      responses:
        '204':
          description: Deleted
`,
			expectedCount: 4,
			expectedText:  "",
		},
		{
			name: "get is safe but post is not protected",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
    post:
      responses:
        '201':
          description: Created
`,
			expectedCount: 1,
			expectedText:  "post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspProtectionGlobalUnsafeStrictRule{}
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
				assert.Contains(t, err.Error(), "must be protected")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspProtectionGlobalUnsafeStrictRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspProtectionGlobalUnsafeStrictRule{}

	assert.Equal(t, "owasp-protection-global-unsafe-strict", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityHint, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
