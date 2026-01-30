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

func TestPathQueryRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "no query string",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "path with parameters but no query",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.PathQueryRule{}
			config := &linter.RuleConfig{}
			docInfo := &linter.DocumentInfo[*openapi.OpenAPI]{Document: doc}

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestPathQueryRule_QueryInPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "query string at end",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users?filter=active:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: `[7:3] error semantic-path-query path "/users?filter=active" contains query string - use parameters array instead`,
		},
		{
			name: "query string with parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}?include=details:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: `[7:3] error semantic-path-query path "/users/{id}?include=details" contains query string - use parameters array instead`,
		},
		{
			name: "single question mark at end",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users?:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: `[7:3] error semantic-path-query path "/users?" contains query string - use parameters array instead`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.PathQueryRule{}
			config := &linter.RuleConfig{}
			docInfo := &linter.DocumentInfo[*openapi.OpenAPI]{Document: doc}

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestPathQueryRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.PathQueryRule{}

	assert.Equal(t, "semantic-path-query", rule.ID())
	assert.Equal(t, rules.CategorySemantic, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
