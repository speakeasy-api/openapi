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

func TestPathDeclarationsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "valid single parameter",
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
			name: "valid multiple parameters",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "no parameters",
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
			name: "parameter with underscores",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{user_id}:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "parameter with hyphens",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{user-id}:
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
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.PathDeclarationsRule{}
			config := &linter.RuleConfig{}

			docInfo := &linter.DocumentInfo[*openapi.OpenAPI]{
				Document: doc,
			}

			errs := rule.Run(ctx, docInfo, config)

			assert.Empty(t, errs, "should have no lint errors")
		})
	}
}

func TestPathDeclarationsRule_EmptyDeclarations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "single empty declaration",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/{}:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] error semantic-path-declarations path \"/api/{}\" contains empty parameter declaration `{}`",
		},
		{
			name: "empty declaration in middle of path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/{}/users:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] error semantic-path-declarations path \"/api/{}/users\" contains empty parameter declaration `{}`",
		},
		{
			name: "multiple empty declarations",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/{}/{}/users:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] error semantic-path-declarations path \"/api/{}/{}/users\" contains empty parameter declaration `{}`",
		},
		{
			name: "empty declaration with valid parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/{userId}/{}:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[7:3] error semantic-path-declarations path \"/api/{userId}/{}\" contains empty parameter declaration `{}`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.PathDeclarationsRule{}
			config := &linter.RuleConfig{}

			docInfo := &linter.DocumentInfo[*openapi.OpenAPI]{
				Document: doc,
			}

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1, "should have one lint error")
			assert.Equal(t, tt.expectedError, errs[0].Error(), "error message should match exactly")
		})
	}
}

func TestPathDeclarationsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.PathDeclarationsRule{}

	assert.Equal(t, "semantic-path-declarations", rule.ID())
	assert.Equal(t, rules.CategorySemantic, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
