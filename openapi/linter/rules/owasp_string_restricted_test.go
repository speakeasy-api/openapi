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

func TestOwaspStringRestrictedRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "string with format",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Email:
      type: string
      format: email
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
paths: {}
`,
		},
		{
			name: "string with pattern",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    PhoneNumber:
      type: string
      pattern: '^\d{3}-\d{3}-\d{4}$'
paths: {}
`,
		},
		{
			name: "non-string type without restrictions is ok",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspStringRestrictedRule{}
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

func TestOwaspStringRestrictedRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "string without any restrictions",
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
			name: "multiple strings without restrictions",
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
			name: "inline string parameter without restrictions",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspStringRestrictedRule{}
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
				assert.Contains(t, err.Error(), "must specify format, const, enum, or pattern")
			}
		})
	}
}

func TestOwaspStringRestrictedRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspStringRestrictedRule{}

	assert.Equal(t, "owasp-string-restricted", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0", "3.1"}, rule.Versions())
}
