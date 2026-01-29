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

func TestOwaspDefineErrorValidationRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "operation with 400 response",
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
          content:
            application/json:
              schema:
                type: object
`,
		},
		{
			name: "operation with 422 response",
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
        '422':
          description: Unprocessable Entity
          content:
            application/json:
              schema:
                type: object
`,
		},
		{
			name: "operation with 4XX wildcard",
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
        '4XX':
          description: Client Error
          content:
            application/json:
              schema:
                type: object
`,
		},
		{
			name: "operation with multiple validation error codes",
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
        '400':
          description: Bad Request
        '422':
          description: Unprocessable Entity
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspDefineErrorValidationRule{}
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

func TestOwaspDefineErrorValidationRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "missing validation error response",
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
        '401':
          description: Unauthorized
        '500':
          description: Server Error
`,
			expectedCount: 1,
			expectedText:  "missing validation error",
		},
		{
			name: "only success response",
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
			expectedText:  "missing validation error",
		},
		{
			name: "multiple operations missing validation errors",
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
			expectedCount: 2,
			expectedText:  "",
		},
		{
			name: "one operation with validation error one without",
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

			rule := &rules.OwaspDefineErrorValidationRule{}
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
				assert.Contains(t, err.Error(), "validation error")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspDefineErrorValidationRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspDefineErrorValidationRule{}

	assert.Equal(t, "owasp-define-error-validation", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
