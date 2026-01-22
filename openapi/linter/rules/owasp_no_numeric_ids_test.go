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

func TestOwaspNoNumericIDsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "id parameter with string type",
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
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "user_id parameter with string type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{user_id}:
    get:
      parameters:
        - name: user_id
          in: path
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "non-id parameter with integer type is ok",
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
        - name: offset
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "referenced id parameter with string type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  parameters:
    UserId:
      name: user_id
      in: path
      schema:
        type: string
        format: uuid
paths:
  /users/{user_id}:
    get:
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "id with object type",
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
          schema:
            type: object
      responses:
        '200':
          description: Success
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoNumericIDsRule{}
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

func TestOwaspNoNumericIDsRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "id parameter with integer type",
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
          schema:
            type: integer
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedText:  "id",
		},
		{
			name: "user_id parameter with integer type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{user_id}:
    get:
      parameters:
        - name: user_id
          in: path
          schema:
            type: integer
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedText:  "user_id",
		},
		{
			name: "post-id parameter with integer type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /posts/{post-id}:
    get:
      parameters:
        - name: post-id
          in: path
          schema:
            type: integer
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedText:  "post-id",
		},
		{
			name: "multiple id parameters with integer type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{user_id}/posts/{post_id}:
    get:
      parameters:
        - name: user_id
          in: path
          schema:
            type: integer
        - name: post_id
          in: path
          schema:
            type: integer
      responses:
        '200':
          description: Success
`,
			expectedCount: 2,
			expectedText:  "",
		},
		{
			name: "component parameter id with integer type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  parameters:
    UserId:
      name: id
      in: path
      schema:
        type: integer
paths:
  /users/{id}:
    get:
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedText:  "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoNumericIDsRule{}
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
				assert.Contains(t, err.Error(), "integer type for ID")
				assert.Contains(t, err.Error(), "UUID")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspNoNumericIDsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspNoNumericIDsRule{}

	assert.Equal(t, "owasp-no-numeric-ids", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0", "3.1"}, rule.Versions())
}
