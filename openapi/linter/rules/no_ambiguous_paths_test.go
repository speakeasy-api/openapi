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

func TestNoAmbiguousPathsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "distinct paths",
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
          description: Success
  /posts:
    get:
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "different parameter names same position",
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
          description: Success
  /posts/{postId}:
    get:
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "same path different methods",
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
          description: Success
    post:
      responses:
        '201':
          description: Created
`,
		},
		{
			name: "concrete path vs template path different base",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/current:
    get:
      responses:
        '200':
          description: Success
  /posts/{id}:
    get:
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "different path lengths",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    get:
      responses:
        '200':
          description: Success
  /users/{id}/posts:
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

			rule := &rules.NoAmbiguousPathsRule{}
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

func TestNoAmbiguousPathsRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
	}{
		{
			name: "ambiguous parameters same method",
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
          description: Success
  /users/{id}:
    get:
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
		},
		{
			name: "multiple ambiguous paths",
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
          description: Success
  /users/{id}/posts/{id2}:
    get:
      responses:
        '200':
          description: Success
  /users/{uid}/posts/{pid}:
    get:
      responses:
        '200':
          description: Success
`,
			expectedCount: 3, // Second path conflicts with first (1), third conflicts with both first and second (2)
		},
		{
			name: "ambiguous paths regardless of methods",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /items/{itemId}:
    get:
      responses:
        '200':
          description: Success
  /items/{id}:
    post:
      responses:
        '201':
          description: Created
`,
			expectedCount: 1, // Paths are ambiguous regardless of HTTP methods
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.NoAmbiguousPathsRule{}
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
				assert.Contains(t, err.Error(), "paths are ambiguous with one another")
			}
		})
	}
}

func TestNoAmbiguousPathsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.NoAmbiguousPathsRule{}

	assert.Equal(t, "semantic-no-ambiguous-paths", rule.ID())
	assert.Equal(t, rules.CategorySemantic, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
