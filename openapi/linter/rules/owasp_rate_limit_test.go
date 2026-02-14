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

func TestOwaspRateLimitRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "2xx response with X-RateLimit-Limit header",
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
          headers:
            X-RateLimit-Limit:
              description: Request limit per hour
              schema:
                type: integer
`,
		},
		{
			name: "4xx response with RateLimit header",
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
            RateLimit:
              description: Rate limit info
              schema:
                type: string
`,
		},
		{
			name: "response with RateLimit-Limit header",
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
          headers:
            RateLimit-Limit:
              schema:
                type: integer
`,
		},
		{
			name: "3xx response without rate limit headers is ok",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '301':
          description: Moved Permanently
`,
		},
		{
			name: "5xx response without rate limit headers is ok",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '500':
          description: Server Error
`,
		},
		{
			name: "multiple responses with rate limit headers",
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
          headers:
            X-Rate-Limit-Limit:
              schema:
                type: integer
        '400':
          description: Bad Request
          headers:
            RateLimit-Reset:
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

			rule := &rules.OwaspRateLimitRule{}
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

func TestOwaspRateLimitRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "2xx response missing rate limit headers",
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
			expectedCount: 1,
			expectedText:  "200",
		},
		{
			name: "4xx response missing rate limit headers",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '400':
          description: Bad Request
`,
			expectedCount: 1,
			expectedText:  "400",
		},
		{
			name: "2xx response has headers but no rate limit header",
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
          headers:
            X-Request-ID:
              schema:
                type: string
`,
			expectedCount: 1,
			expectedText:  "missing rate limiting headers",
		},
		{
			name: "multiple responses missing rate limit headers",
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
        '201':
          description: Created
        '400':
          description: Bad Request
        '404':
          description: Not Found
`,
			expectedCount: 4,
			expectedText:  "",
		},
		{
			name: "mixed responses some with some without rate limit headers",
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
          headers:
            RateLimit:
              schema:
                type: string
        '400':
          description: Bad Request
`,
			expectedCount: 1,
			expectedText:  "400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspRateLimitRule{}
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
				assert.Contains(t, err.Error(), "rate limiting")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspRateLimitRule_RefResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "ref response without rate limit headers is flagged",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  responses:
    Success:
      description: Success
paths:
  /users:
    get:
      responses:
        '200':
          $ref: '#/components/responses/Success'
`,
			expectedCount: 1,
		},
		{
			name: "ref response with rate limit headers is valid",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  responses:
    Success:
      description: Success
      headers:
        RateLimit:
          schema:
            type: string
paths:
  /users:
    get:
      responses:
        '200':
          $ref: '#/components/responses/Success'
`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			resolveOpts := references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			}

			rule := &rules.OwaspRateLimitRule{}
			config := &linter.RuleConfig{
				ResolveOptions: &resolveOpts,
			}

			idx := openapi.BuildIndex(ctx, doc, resolveOpts)
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Len(t, errs, tt.expectedCount)
		})
	}
}

func TestOwaspRateLimitRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspRateLimitRule{}

	assert.Equal(t, "owasp-rate-limit", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
