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

func TestOperationErrorResponseRule_ValidCases(t *testing.T) {
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
`,
		},
		{
			name: "operation with 404 response",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    get:
      responses:
        '200':
          description: Success
        '404':
          description: Not Found
`,
		},
		{
			name: "operation with multiple 4xx responses",
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
        '401':
          description: Unauthorized
        '403':
          description: Forbidden
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
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OperationErrorResponseRule{}
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

func TestOperationErrorResponseRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "operation with only success response",
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
		},
		{
			name: "operation with only 5xx response",
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
        '500':
          description: Internal Server Error
`,
			expectedCount: 1,
		},
		{
			name: "multiple operations without 4xx responses",
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
  /posts:
    get:
      responses:
        '200':
          description: Success
`,
			expectedCount: 3,
		},
		{
			name: "operation with default response only",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        default:
          description: Default response
`,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OperationErrorResponseRule{}
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
				assert.Contains(t, err.Error(), "operation must define at least one 4xx error response")
			}
		})
	}
}

func TestOperationErrorResponseRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationErrorResponseRule{}

	assert.Equal(t, "style-operation-error-response", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
