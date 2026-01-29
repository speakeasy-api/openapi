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

func TestTagsAlphabeticalRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "tags in alphabetical order",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: auth
    description: Authentication
  - name: products
    description: Products
  - name: users
    description: Users
paths: {}
`,
		},
		{
			name: "single tag",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: Users
paths: {}
`,
		},
		{
			name: "no tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
`,
		},
		{
			name: "tags with case variations in alphabetical order",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: Auth
    description: Authentication
  - name: products
    description: Products
  - name: Users
    description: Users
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

			rule := &rules.TagsAlphabeticalRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestTagsAlphabeticalRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "tags not in alphabetical order",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: Users
  - name: auth
    description: Authentication
paths: {}
`,
			expectedError: "[7:3] warning style-tags-alphabetical tag `auth` must be placed before `users` (alphabetical)",
		},
		{
			name: "tags reversed",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
    description: Users
  - name: products
    description: Products
  - name: auth
    description: Authentication
paths: {}
`,
			expectedError: "[7:3] warning style-tags-alphabetical tag `products` must be placed before `users` (alphabetical)",
		},
		{
			name: "middle tags out of order",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: auth
    description: Authentication
  - name: users
    description: Users
  - name: products
    description: Products
paths: {}
`,
			expectedError: "[7:3] warning style-tags-alphabetical tag `products` must be placed before `users` (alphabetical)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.TagsAlphabeticalRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestTagsAlphabeticalRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.TagsAlphabeticalRule{}

	assert.Equal(t, "style-tags-alphabetical", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
