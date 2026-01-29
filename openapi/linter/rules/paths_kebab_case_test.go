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

func TestPathsKebabCaseRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "kebab-case path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api-users:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "kebab-case with numbers",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api-v1-users:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "path with variables",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "mixed kebab-case and variables",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api-users/{userId}/user-profile:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "path with dots and dashes",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api.v1/user-data:
    get:
      responses:
        '200':
          description: ok
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.PathsKebabCaseRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestPathsKebabCaseRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "camelCase path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /apiUsers:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-paths-kebab-case path segments `apiUsers` are not kebab-case",
		},
		{
			name: "snake_case path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api_users:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-paths-kebab-case path segments `api_users` are not kebab-case",
		},
		{
			name: "uppercase path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /API/USERS:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-paths-kebab-case path segments `API`, `USERS` are not kebab-case",
		},
		{
			name: "mixed valid and invalid segments",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api-users/userId:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-paths-kebab-case path segments `userId` are not kebab-case",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.PathsKebabCaseRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestPathsKebabCaseRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.PathsKebabCaseRule{}

	assert.Equal(t, "style-paths-kebab-case", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
