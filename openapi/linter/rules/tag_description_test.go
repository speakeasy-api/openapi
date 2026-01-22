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

func TestTagDescriptionRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "tag with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: User management endpoints
paths: {}
`,
		},
		{
			name: "multiple tags with descriptions",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: User management endpoints
  - name: products
    description: Product management endpoints
paths: {}
`,
		},
		{
			name: "no tags defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
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

			rule := &rules.TagDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestTagDescriptionRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "tag without description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
paths: {}
`,
			expectedError: "[7:5] hint style-tag-description tag `users` must have a description",
		},
		{
			name: "tag with empty description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: ""
paths: {}
`,
			expectedError: "[7:5] hint style-tag-description tag `users` must have a description",
		},
		{
			name: "one tag with description, one without",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: User management endpoints
  - name: products
paths: {}
`,
			expectedError: "[9:5] hint style-tag-description tag `products` must have a description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.TagDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestTagDescriptionRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.TagDescriptionRule{}

	assert.Equal(t, "style-tag-description", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityHint, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
