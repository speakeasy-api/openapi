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

func TestOwaspNoAdditionalPropertiesRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "object without additionalProperties set",
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
			name: "non-object type with additionalProperties is ok",
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

			rule := &rules.OwaspNoAdditionalPropertiesRule{}
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

func TestOwaspNoAdditionalPropertiesRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "object with additionalProperties true",
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
      additionalProperties: true
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "object with additionalProperties schema",
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
      additionalProperties:
        type: string
        maxLength: 50
paths: {}
`,
			expectedCount: 1,
		},
		{
			name: "multiple objects with additionalProperties violations",
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
      additionalProperties: true
    Config:
      type: object
      properties:
        setting:
          type: string
          maxLength: 50
      additionalProperties:
        type: string
        maxLength: 100
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

			rule := &rules.OwaspNoAdditionalPropertiesRule{}
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
				assert.Contains(t, err.Error(), "additionalProperties")
			}
		})
	}
}

func TestOwaspNoAdditionalPropertiesRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspNoAdditionalPropertiesRule{}

	assert.Equal(t, "owasp-no-additional-properties", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
