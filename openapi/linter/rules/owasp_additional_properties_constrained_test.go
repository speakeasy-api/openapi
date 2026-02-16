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

func TestOwaspAdditionalPropertiesConstrainedRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "object without additionalProperties",
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
			name: "object with additionalProperties false",
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
      additionalProperties: false
paths: {}
`,
		},
		{
			name: "object with additionalProperties true and maxProperties",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        setting:
          type: string
          maxLength: 50
      additionalProperties: true
      maxProperties: 10
paths: {}
`,
		},
		{
			name: "object with additionalProperties schema and maxProperties",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        name:
          type: string
          maxLength: 100
      additionalProperties:
        type: string
        maxLength: 50
      maxProperties: 20
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

			rule := &rules.OwaspAdditionalPropertiesConstrainedRule{}
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

func TestOwaspAdditionalPropertiesConstrainedRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "object with additionalProperties true without maxProperties",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        setting:
          type: string
          maxLength: 50
      additionalProperties: true
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "object with additionalProperties schema without maxProperties",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        name:
          type: string
          maxLength: 100
      additionalProperties:
        type: string
        maxLength: 50
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "multiple objects with violations",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config1:
      type: object
      properties:
        setting:
          type: string
          maxLength: 50
      additionalProperties: true
    Config2:
      type: object
      properties:
        value:
          type: string
          maxLength: 100
      additionalProperties:
        type: integer
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

			rule := &rules.OwaspAdditionalPropertiesConstrainedRule{}
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
				assert.Contains(t, err.Error(), "maxProperties")
			}
		})
	}
}

func TestOwaspAdditionalPropertiesConstrainedRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspAdditionalPropertiesConstrainedRule{}

	assert.Equal(t, "owasp-additional-properties-constrained", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityHint, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
