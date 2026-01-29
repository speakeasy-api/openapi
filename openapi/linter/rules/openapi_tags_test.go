package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPITagsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "single tag defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
paths: {}
`,
		},
		{
			name: "multiple tags defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
  - name: pets
    description: Pet operations
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

			rule := &rules.OpenAPITagsRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOpenAPITagsRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "no tags defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
			expectedError: "[2:1] warning style-openapi-tags OpenAPI object must have a non-empty tags array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OpenAPITagsRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestOpenAPITagsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OpenAPITagsRule{}

	assert.Equal(t, "style-openapi-tags", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
