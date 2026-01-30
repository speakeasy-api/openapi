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

func TestLicenseURLRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "license with URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT
paths: {}
`,
		},
		{
			name: "no license object",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
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

			rule := &rules.LicenseURLRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestLicenseURLRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "license without URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  license:
    name: MIT
paths: {}
`,
			expectedError: "[7:5] hint style-license-url license should contain a URL",
		},
		{
			name: "license with empty URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  license:
    name: MIT
    url: ""
paths: {}
`,
			expectedError: "[7:5] hint style-license-url license should contain a URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.LicenseURLRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestLicenseURLRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.LicenseURLRule{}

	assert.Equal(t, "style-license-url", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityHint, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
