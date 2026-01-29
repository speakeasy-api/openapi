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

func TestOAS3NoNullableRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "OpenAPI 3.1 with type array including null",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableString:
      type: [string, "null"]
      maxLength: 100
paths: {}
`,
		},
		{
			name: "OpenAPI 3.1 without nullable",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Name:
      type: string
      maxLength: 100
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

			rule := &rules.OAS3NoNullableRule{}
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

func TestOAS3NoNullableRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "OpenAPI 3.1 with nullable true",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableName:
      type: string
      nullable: true
      maxLength: 100
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "OpenAPI 3.1 with nullable false also flagged",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Name:
      type: string
      nullable: false
      maxLength: 100
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "OpenAPI 3.1 with multiple nullable schemas",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableString:
      type: string
      nullable: true
      maxLength: 100
    NullableInteger:
      type: integer
      nullable: true
      format: int32
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

			rule := &rules.OAS3NoNullableRule{}
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
				assert.Contains(t, err.Error(), "nullable")
				assert.Contains(t, err.Error(), "3.1")
			}
		})
	}
}

func TestOAS3NoNullableRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OAS3NoNullableRule{}

	assert.Equal(t, "oas3-no-nullable", rule.ID())
	assert.Equal(t, rules.CategorySchemas, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.1"}, rule.Versions())
}
