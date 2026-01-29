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

func TestOperationDescriptionRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "operation with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      description: Get all users
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "operation with summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "operation with both description and summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      description: This endpoint retrieves all users from the database
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

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			rule := &rules.OperationDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOperationDescriptionRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "operation without description or summary",
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
			expectedError: "[9:7] warning style-operation-description the `GET` /users is missing a description or summary",
		},
		{
			name: "operation with operationId but no description",
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
			expectedError: "[9:7] warning style-operation-description the createUser is missing a description or summary",
		},
		{
			name: "multiple operations missing description",
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
  /products:
    get:
      responses:
        '200':
          description: ok
`,
			expectedError: "[9:7] warning style-operation-description the `GET` /users is missing a description or summary",
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

			rule := &rules.OperationDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestOperationDescriptionRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationDescriptionRule{}

	assert.Equal(t, "style-operation-description", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
