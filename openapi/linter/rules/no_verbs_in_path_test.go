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

func TestNoVerbsInPathRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "path without verbs",
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
			name: "path with resource names",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}/profile:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "path with compound words containing verb letters",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /budget:
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

			rule := &rules.NoVerbsInPathRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestNoVerbsInPathRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "path with GET verb",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /get/users:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-no-verbs-in-path path `/get/users` must not contain HTTP verb `get`",
		},
		{
			name: "path with POST verb",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/post:
    post:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-no-verbs-in-path path `/users/post` must not contain HTTP verb `post`",
		},
		{
			name: "path with DELETE verb",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /delete-users:
    delete:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-no-verbs-in-path path `/delete-users` must not contain HTTP verb `delete-users`",
		},
		{
			name: "path with QUERY verb (OpenAPI 3.2)",
			yaml: `
openapi: 3.2.0
info:
  title: Test
  version: 1.0.0
paths:
  /query/users:
    query:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-no-verbs-in-path path `/query/users` must not contain HTTP verb `query`",
		},
		{
			name: "path with uppercase verb",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/GET:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] warning style-no-verbs-in-path path `/users/GET` must not contain HTTP verb `GET`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.NoVerbsInPathRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestNoVerbsInPathRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.NoVerbsInPathRule{}

	assert.Equal(t, "style-no-verbs-in-path", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
