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

func TestOwaspRateLimitRetryAfterRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "429 response with Retry-After header",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
          headers:
            Retry-After:
              description: Number of seconds to wait
              schema:
                type: integer
`,
		},
		{
			name: "429 with lowercase retry-after header",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
          headers:
            retry-after:
              schema:
                type: integer
`,
		},
		{
			name: "no 429 response is ok",
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
        '400':
          description: Bad Request
`,
		},
		{
			name: "429 with Retry-After and other headers",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
          headers:
            Retry-After:
              schema:
                type: integer
            X-RateLimit-Limit:
              schema:
                type: integer
            X-RateLimit-Remaining:
              schema:
                type: integer
`,
		},
		{
			name: "multiple operations with 429 and Retry-After",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
          headers:
            Retry-After:
              schema:
                type: integer
    post:
      responses:
        '429':
          description: Too Many Requests
          headers:
            Retry-After:
              schema:
                type: integer
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspRateLimitRetryAfterRule{}
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

func TestOwaspRateLimitRetryAfterRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "429 response missing Retry-After header",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
`,
			expectedCount: 1,
			expectedText:  "Retry-After",
		},
		{
			name: "429 has headers but no Retry-After",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
          headers:
            X-RateLimit-Limit:
              schema:
                type: integer
`,
			expectedCount: 1,
			expectedText:  "Retry-After",
		},
		{
			name: "multiple operations with 429 missing Retry-After",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
    post:
      responses:
        '429':
          description: Too Many Requests
  /posts:
    get:
      responses:
        '429':
          description: Too Many Requests
`,
			expectedCount: 3,
			expectedText:  "",
		},
		{
			name: "mixed operations some with some without Retry-After",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '429':
          description: Too Many Requests
          headers:
            Retry-After:
              schema:
                type: integer
    post:
      responses:
        '429':
          description: Too Many Requests
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

			rule := &rules.OwaspRateLimitRetryAfterRule{}
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
				assert.Contains(t, err.Error(), "429")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspRateLimitRetryAfterRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspRateLimitRetryAfterRule{}

	assert.Equal(t, "owasp-rate-limit-retry-after", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0", "3.1"}, rule.Versions())
}
