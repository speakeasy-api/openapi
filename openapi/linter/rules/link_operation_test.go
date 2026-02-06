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

func TestLinkOperationRule_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "link with valid operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          links:
            self:
              operationId: getUsers
`,
		},
		{
			name: "link with operationRef",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          links:
            self:
              operationRef: '#/paths/~1users/get'
`,
		},
		{
			name: "link references operation from different path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          links:
            details:
              operationId: getUserById
  /users/{id}:
    get:
      operationId: getUserById
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "no links",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "component link with valid operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
components:
  links:
    UserLink:
      operationId: getUsers
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.LinkOperationRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			require.Empty(t, errs, "expected no lint errors")
		})
	}
}

func TestLinkOperationRule_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "link with invalid operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          links:
            details:
              operationId: nonExistentOperation
`,
			expectedError: "link.operationId value `nonExistentOperation` does not exist in document",
		},
		{
			name: "component link with invalid operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
components:
  links:
    InvalidLink:
      operationId: invalidOperation
`,
			expectedError: "link.operationId value `invalidOperation` does not exist in document",
		},
		{
			name: "multiple invalid operationIds",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          links:
            link1:
              operationId: invalid1
            link2:
              operationId: invalid2
`,
			expectedError: "link.operationId value `invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.LinkOperationRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs, "should have lint errors")

			// Check that at least one error contains the expected message
			found := false
			for _, err := range errs {
				if strings.Contains(err.Error(), tt.expectedError) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected error message containing: %s", tt.expectedError)
		})
	}
}

func TestLinkOperationRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.LinkOperationRule{}

	assert.Equal(t, "semantic-link-operation", rule.ID(), "rule ID should match")
	assert.Equal(t, rules.CategorySemantic, rule.Category(), "rule category should match")
	assert.NotEmpty(t, rule.Description(), "rule should have description")
	assert.NotEmpty(t, rule.Link(), "rule should have documentation link")
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity(), "default severity should be error")
	assert.Nil(t, rule.Versions(), "versions should be nil (all versions)")
}
