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

func TestInfoDescriptionRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "info with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  description: This is a test API
paths: {}
`,
		},
		{
			name: "info with multiline description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  description: |
    This is a test API
    with multiple lines
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

			rule := &rules.InfoDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestInfoDescriptionRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "info without description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`,
			expectedError: "[4:3] warning style-info-description info section is missing a description",
		},
		{
			name: "info with empty description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  description: ""
paths: {}
`,
			expectedError: "[4:3] warning style-info-description info section is missing a description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.InfoDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestInfoDescriptionRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.InfoDescriptionRule{}

	assert.Equal(t, "style-info-description", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
