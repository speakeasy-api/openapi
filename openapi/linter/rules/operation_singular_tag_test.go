package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationSingularTagRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "operation with single tag",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      tags:
        - users
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "operation with no tags",
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
			name: "multiple operations each with single tag",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      tags:
        - users
      responses:
        '200':
          description: ok
  /products:
    get:
      tags:
        - products
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

			rule := &rules.OperationSingularTagRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOperationSingularTagRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "operation with two tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      tags:
        - users
        - admin
      responses:
        '200':
          description: ok
`,
			expectedError: "[10:9] warning style-operation-singular-tag the `GET` operation at path `/users` contains more than one tag (2 is too many)",
		},
		{
			name: "operation with three tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      tags:
        - users
        - admin
        - public
      responses:
        '200':
          description: ok
`,
			expectedError: "[10:9] warning style-operation-singular-tag the `GET` operation at path `/users` contains more than one tag (3 is too many)",
		},
		{
			name: "operation with operationId and multiple tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      tags:
        - users
        - admin
      responses:
        '200':
          description: ok
`,
			expectedError: "[11:9] warning style-operation-singular-tag the `listUsers` operation contains more than one tag (2 is too many)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OperationSingularTagRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestOperationSingularTagRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationSingularTagRule{}

	assert.Equal(t, "style-operation-singular-tag", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
