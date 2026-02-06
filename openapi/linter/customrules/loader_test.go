package customrules_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi/linter/customrules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_LoadRules_Success(t *testing.T) {
	t.Parallel()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1, "should load 1 rule")

	rule := rules[0]
	assert.Equal(t, "custom-require-operation-summary", rule.ID())
	assert.Equal(t, "style", rule.Category())
	assert.Contains(t, rule.Description(), "summary")
}

func TestLoader_LoadRules_NoFiles(t *testing.T) {
	t.Parallel()

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	assert.Nil(t, rules)
}

func TestLoader_LoadRules_GlobPattern(t *testing.T) {
	t.Parallel()

	// Use specific glob pattern for valid rules only (require-*.ts)
	testdataPath, err := filepath.Abs("testdata/require-*.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rules), 2, "should load at least 2 rules from glob pattern")
}

func TestCustomRule_Run_FindsViolations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Load the custom rule
	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Parse an OpenAPI doc without summaries
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

	// Run the rule
	errs := rules[0].Run(ctx, docInfo, nil)
	require.Len(t, errs, 1, "should find 1 violation")
	assert.Contains(t, errs[0].Error(), "missing a summary")
}

func TestCustomRule_Run_NoViolations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Load the custom rule
	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Parse an OpenAPI doc with summaries
	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Get all users
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

	// Run the rule
	errs := rules[0].Run(ctx, docInfo, nil)
	assert.Empty(t, errs, "should find no violations")
}

func TestIntegration_NewLinterWithCustomRules(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Get absolute path to testdata
	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	// Configure linter with custom rules
	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
	}

	// Create linter (this triggers custom rule loading via init())
	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	// Check that custom rule is registered
	registry := lint.Registry()
	rule, found := registry.GetRule("custom-require-operation-summary")
	assert.True(t, found, "custom rule should be registered")
	assert.NotNil(t, rule)

	// Parse and lint a document
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

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Check for our custom rule's error
	foundCustomError := false
	for _, result := range output.Results {
		if strings.Contains(result.Error(), "missing a summary") {
			foundCustomError = true
			break
		}
	}
	assert.True(t, foundCustomError, "should find custom rule violation")
}
