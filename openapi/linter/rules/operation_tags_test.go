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

func TestOperationTagsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "operation with tags",
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
			name: "operation with multiple tags",
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
        - public
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "all operations have tags",
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
    post:
      tags:
        - users
      responses:
        '201':
          description: created
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			rule := &rules.OperationTagsRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOperationTagsRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "operation without tags",
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
			expectedError: "[9:7] warning style-operation-tags the `GET` /users is missing tags",
		},
		{
			name: "operation with operationId but no tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    post:
      operationId: createUser
      responses:
        '201':
          description: created
`,
			expectedError: "[9:7] warning style-operation-tags the createUser is missing tags",
		},
		{
			name: "one operation with tags, one without",
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
    post:
      responses:
        '201':
          description: created
`,
			expectedError: "[15:7] warning style-operation-tags the `POST` /users is missing tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			rule := &rules.OperationTagsRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestOperationTagsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationTagsRule{}

	assert.Equal(t, "style-operation-tags", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
