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

func TestOperationIdRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "all operations have ids",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /melody:
    post:
      operationId: littleSong
      responses:
        '200':
          description: ok
  /ember:
    get:
      operationId: littleChampion
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "empty paths",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OperationIdRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
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

func TestOperationIdRule_Violations(t *testing.T) {
	t.Parallel()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /melody:
    post:
      operationId: littleSong
      responses:
        '200':
          description: ok
  /ember:
    get:
      responses:
        '200':
          description: ok
`

	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err, "unmarshal should succeed")

	rule := &rules.OperationIdRule{}
	config := &linter.RuleConfig{}

	// Build index for the rule
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	errs := rule.Run(ctx, docInfo, config)

	require.Len(t, errs, 1, "should have one lint error")
	assert.Equal(t, "[15:7] warning semantic-operation-operation-id the `GET` operation does not contain an `operationId`", errs[0].Error())
}

func TestOperationIdRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationIdRule{}

	assert.Equal(t, "semantic-operation-operation-id", rule.ID(), "rule ID should match")
	assert.Equal(t, rules.CategorySemantic, rule.Category(), "rule category should match")
	assert.NotEmpty(t, rule.Description(), "rule should have description")
	assert.NotEmpty(t, rule.Link(), "rule should have documentation link")
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity(), "default severity should be warning")
	assert.Nil(t, rule.Versions(), "versions should be nil (all versions)")
}
