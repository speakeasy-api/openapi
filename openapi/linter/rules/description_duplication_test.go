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

func TestDescriptionDuplicationRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "different description and summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      description: Returns a list of all users in the system
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "only description no summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      description: Returns a list of all users in the system
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "only summary no description",
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
          description: Success
`,
		},
		{
			name: "same text in different operations is allowed",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get items
      description: Get items from the database
      responses:
        '200':
          description: Success
  /posts:
    get:
      summary: Get items
      description: Get items from the database
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "path item with different description and summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    summary: User endpoints
    description: All user-related endpoints
    get:
      responses:
        '200':
          description: Success
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.DescriptionDuplicationRule{}
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

func TestDescriptionDuplicationRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "operation with identical description and summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      description: Get all users
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
		},
		{
			name: "path item with identical description and summary",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    summary: User endpoints
    description: User endpoints
    get:
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
		},
		{
			name: "multiple operations with duplicates",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      description: Get all users
      responses:
        '200':
          description: Success
    post:
      summary: Create user
      description: Create user
      responses:
        '201':
          description: Created
`,
			expectedCount: 2,
		},
		{
			name: "path item and operation both with duplicates",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    summary: Users
    description: Users
    get:
      summary: Get all users
      description: Get all users
      responses:
        '200':
          description: Success
`,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.DescriptionDuplicationRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, tt.expectedCount)
			for _, err := range errs {
				assert.Contains(t, err.Error(), "summary is identical to description")
			}
		})
	}
}

func TestDescriptionDuplicationRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.DescriptionDuplicationRule{}

	assert.Equal(t, "style-description-duplication", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
