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

func TestOwaspIntegerLimitRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "integer with minimum and maximum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Age:
      type: integer
      format: int32
      minimum: 0
      maximum: 120
paths: {}
`,
		},
		{
			name: "integer with minimum and exclusiveMaximum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      minimum: 0
      exclusiveMaximum: 100
paths: {}
`,
		},
		{
			name: "integer with exclusiveMinimum and maximum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      exclusiveMinimum: 0
      maximum: 100
paths: {}
`,
		},
		{
			name: "integer with exclusiveMinimum and exclusiveMaximum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      exclusiveMinimum: 0
      exclusiveMaximum: 100
paths: {}
`,
		},
		{
			name: "non-integer type without limits is ok",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Name:
      type: string
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

			rule := &rules.OwaspIntegerLimitRule{}
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

func TestOwaspIntegerLimitRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "integer without any limits",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "integer with only minimum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      minimum: 0
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "integer with only maximum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      maximum: 100
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "integer with only exclusiveMinimum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      exclusiveMinimum: 0
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "integer with only exclusiveMaximum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Counter:
      type: integer
      format: int32
      exclusiveMaximum: 100
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "multiple integers without proper limits",
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
        age:
          type: integer
          format: int32
        count:
          type: integer
          format: int32
          minimum: 0
paths: {}
`,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspIntegerLimitRule{}
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
				assert.Contains(t, err.Error(), "must specify `minimum` and `maximum`")
			}
		})
	}
}

func TestOwaspIntegerLimitRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspIntegerLimitRule{}

	assert.Equal(t, "owasp-integer-limit", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0", "3.1"}, rule.Versions())
}
