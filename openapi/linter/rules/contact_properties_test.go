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

func TestContactPropertiesRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "contact with all properties",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  contact:
    name: API Support
    url: https://www.example.com/support
    email: support@example.com
paths: {}
`,
		},
		{
			name: "no contact object",
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

			rule := &rules.ContactPropertiesRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestContactPropertiesRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "contact missing name",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  contact:
    url: https://www.example.com/support
    email: support@example.com
paths: {}
`,
			expectedError: "[7:5] warning style-contact-properties `contact` section must contain a `name`",
		},
		{
			name: "contact missing url",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  contact:
    name: API Support
    email: support@example.com
paths: {}
`,
			expectedError: "[7:5] warning style-contact-properties `contact` section must contain a `url`",
		},
		{
			name: "contact missing email",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  contact:
    name: API Support
    url: https://www.example.com/support
paths: {}
`,
			expectedError: "[7:5] warning style-contact-properties `contact` section must contain an `email`",
		},
		{
			name: "contact missing all properties",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
  contact: {}
paths: {}
`,
			expectedError: "[6:12] warning style-contact-properties `contact` section must contain a `name`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.ContactPropertiesRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestContactPropertiesRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.ContactPropertiesRule{}

	assert.Equal(t, "style-contact-properties", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
