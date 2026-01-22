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

func TestNoEvalInMarkdownRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "no descriptions",
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
			name: "description without eval",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
  description: safe content
paths:
  /users:
    get:
      description: plain text
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

			rule := &rules.NoEvalInMarkdownRule{}
			config := &linter.RuleConfig{}
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

func TestNoEvalInMarkdownRule_Violations(t *testing.T) {
	t.Parallel()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
  description: eval("alert('x')")
paths:
  /users:
    get:
      description: "safe"
      responses:
        '200':
          description: ok
  /admin:
    get:
      description: eval("evil")
      responses:
        '200':
          description: ok
`

	expectedErrors := []string{
		"[6:16] error semantic-no-eval-in-markdown description contains content with `eval\\(`, forbidden",
		"[16:20] error semantic-no-eval-in-markdown description contains content with `eval\\(`, forbidden",
	}

	ctx := t.Context()

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	rule := &rules.NoEvalInMarkdownRule{}
	config := &linter.RuleConfig{}
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	errs := rule.Run(ctx, docInfo, config)

	require.Len(t, errs, 2)
	assert.Equal(t, expectedErrors[0], errs[0].Error())
	assert.Equal(t, expectedErrors[1], errs[1].Error())
}

func TestNoEvalInMarkdownRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.NoEvalInMarkdownRule{}

	assert.Equal(t, "semantic-no-eval-in-markdown", rule.ID())
	assert.Equal(t, rules.CategorySemantic, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
