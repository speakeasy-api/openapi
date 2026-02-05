package customrules_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/customrules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireDescriptionRule_FindsViolations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Load the require-description rule
	testdataPath, err := filepath.Abs("testdata/require-description.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Verify rule metadata
	rule := rules[0]
	assert.Equal(t, "custom-require-operation-description", rule.ID())
	assert.Equal(t, "style", rule.Category())

	// Parse an OpenAPI doc without descriptions
	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	// Build index
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Run the rule - should find violation (no description)
	errs := rules[0].Run(ctx, docInfo, nil)
	require.Len(t, errs, 1, "should find 1 violation")
	assert.Contains(t, errs[0].Error(), "missing a description")
}

func TestRequireDescriptionRule_NoViolations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Load the require-description rule
	testdataPath, err := filepath.Abs("testdata/require-description.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Parse an OpenAPI doc with descriptions
	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      description: Get all users from the system
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	// Build index
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Run the rule - should find no violations
	errs := rules[0].Run(ctx, docInfo, nil)
	assert.Empty(t, errs, "should find no violations")
}
