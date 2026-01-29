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

func TestOwaspStringLimitRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "string with maxLength",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
          maxLength: 100
paths: {}
`,
		},
		{
			name: "string with const",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      const: active
paths: {}
`,
		},
		{
			name: "string with enum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
paths: {}
`,
		},
		{
			name: "non-string type without maxLength is ok",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
paths: {}
`,
		},
		{
			name: "object type is ok",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          maxLength: 36
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

			rule := &rules.OwaspStringLimitRule{}
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

func TestOwaspStringLimitRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "string without maxLength, const, or enum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "multiple strings without constraints",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
        description:
          type: string
paths: {}
`,
			expectedCount: 3,
		},
		{
			name: "inline string parameter without maxLength",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: search
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
		},
		{
			name: "response with string without constraint",
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
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: string
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

			rule := &rules.OwaspStringLimitRule{}
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
				assert.Contains(t, err.Error(), "must specify maxLength, const, or enum")
			}
		})
	}
}

func TestOwaspStringLimitRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspStringLimitRule{}

	assert.Equal(t, "owasp-string-limit", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0", "3.1"}, rule.Versions())
}
