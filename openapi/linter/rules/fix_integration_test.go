package rules

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// These integration tests verify that fixes actually resolve the violations
// they are meant to fix. The pattern is:
// 1. Parse a document with a known violation
// 2. Run the rule to get errors with fixes
// 3. Apply the fix
// 4. Re-parse/re-run to verify the violation is resolved

// helper: parse OpenAPI document from YAML string
func parseOpenAPIDoc(t *testing.T, yamlStr string) *openapi.OpenAPI {
	t.Helper()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlStr))
	require.NoError(t, err, "unmarshal should succeed")
	return doc
}

// helper: build index and create DocumentInfo
func buildDocInfo(t *testing.T, doc *openapi.OpenAPI) *linter.DocumentInfo[*openapi.OpenAPI] {
	t.Helper()
	ctx := t.Context()
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	return linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)
}

// helper: extract NodeFix from first error
func extractNodeFix(t *testing.T, errs []error) validation.NodeFix {
	t.Helper()
	require.NotEmpty(t, errs, "should have at least one error")
	var valErr *validation.Error
	require.ErrorAs(t, errs[0], &valErr, "error should be a *validation.Error")
	require.NotNil(t, valErr.Fix, "error should have a fix")
	nodeFix, ok := valErr.Fix.(validation.NodeFix)
	require.True(t, ok, "fix should implement NodeFix")
	return nodeFix
}

// helper: re-parse from modified YAML root node
func remarshalAndParse(t *testing.T, doc *openapi.OpenAPI) *openapi.OpenAPI {
	t.Helper()
	rootNode := doc.GetCore().GetRootNode()
	require.NotNil(t, rootNode, "root node should exist")

	out, err := yaml.Marshal(rootNode)
	require.NoError(t, err, "marshal should succeed")

	return parseOpenAPIDoc(t, string(out))
}

// ============================================================
// Non-interactive fix integration tests
// ============================================================

func TestFixIntegration_HostTrailingSlash(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
servers:
  - url: https://api.example.com/
paths: {}
`)

	// Step 1: Run rule and get violation
	rule := &OAS3HostTrailingSlashRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.Len(t, errs, 1, "should detect trailing slash")

	// Step 2: Apply fix
	nodeFix := extractNodeFix(t, errs)
	require.NoError(t, nodeFix.ApplyNode(nil))

	// Step 3: Re-parse and verify violation is gone
	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the trailing slash violation")
}

func TestFixIntegration_HTTPSUpgrade(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
servers:
  - url: http://api.example.com
paths: {}
`)

	rule := &OwaspSecurityHostsHttpsOAS3Rule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect http:// URL")

	nodeFix := extractNodeFix(t, errs)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the HTTPS violation")
}

func TestFixIntegration_DuplicateEnum(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - active
`)

	rule := &DuplicatedEnumRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect duplicate enum entry")

	nodeFix := extractNodeFix(t, errs)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the duplicate enum violation")
}

func TestFixIntegration_AdditionalPropertiesFalse(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      additionalProperties: true
`)

	rule := &OwaspNoAdditionalPropertiesRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect additionalProperties: true")

	nodeFix := extractNodeFix(t, errs)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the additionalProperties violation")
}

func TestFixIntegration_TagsAlphabetical(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
tags:
  - name: users
  - name: admin
  - name: pets
paths: {}
`)

	rule := &TagsAlphabeticalRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect unsorted tags")

	nodeFix := extractNodeFix(t, errs)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the alphabetical tag violation")
}

func TestFixIntegration_AddErrorResponse(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
`)

	rule := &OwaspDefineErrorResponses401Rule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect missing 401 response")

	nodeFix := extractNodeFix(t, errs)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	// The fix adds {401: {description: "Unauthorized"}} which resolves the "missing response"
	// violation. The rule may still report "missing content schema" — that's a different, lesser violation.
	for _, err := range errs2 {
		assert.NotContains(t, err.Error(), "must define", "the 'must define response' violation should be resolved")
	}
}

// ============================================================
// Interactive fix integration tests
// ============================================================

func TestFixIntegration_InteractiveIntegerFormat(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Age:
      type: integer
`)

	rule := &OwaspIntegerFormatRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect missing integer format")

	// Extract fix and simulate user input
	var valErr *validation.Error
	require.ErrorAs(t, errs[0], &valErr)
	require.NotNil(t, valErr.Fix)
	fix := valErr.Fix

	assert.True(t, fix.Interactive(), "should be an interactive fix")
	prompts := fix.Prompts()
	require.Len(t, prompts, 1)
	assert.Equal(t, validation.PromptChoice, prompts[0].Type)

	// Simulate user choosing "int32"
	require.NoError(t, fix.SetInput([]string{"int32"}))

	nodeFix, ok := fix.(validation.NodeFix)
	require.True(t, ok)
	require.NoError(t, nodeFix.ApplyNode(nil))

	// Re-parse and verify
	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the integer format violation")
}

func TestFixIntegration_InteractiveStringLimit(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Name:
      type: string
`)

	rule := &OwaspStringLimitRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect missing string limit")

	var valErr *validation.Error
	require.ErrorAs(t, errs[0], &valErr)
	require.NotNil(t, valErr.Fix)
	fix := valErr.Fix

	assert.True(t, fix.Interactive())

	// Simulate user entering maxLength = 255
	require.NoError(t, fix.SetInput([]string{"255"}))

	nodeFix, ok := fix.(validation.NodeFix)
	require.True(t, ok)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the string limit violation")
}

func TestFixIntegration_InteractiveIntegerLimits(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Count:
      type: integer
      format: int32
`)

	rule := &OwaspIntegerLimitRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect missing integer limits")

	var valErr *validation.Error
	require.ErrorAs(t, errs[0], &valErr)
	require.NotNil(t, valErr.Fix)
	fix := valErr.Fix

	assert.True(t, fix.Interactive())
	prompts := fix.Prompts()
	require.Len(t, prompts, 2)

	// Simulate user entering min=0, max=1000
	require.NoError(t, fix.SetInput([]string{"0", "1000"}))

	nodeFix, ok := fix.(validation.NodeFix)
	require.True(t, ok)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the integer limit violation")
}

func TestFixIntegration_InteractiveArrayLimit(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Items:
      type: array
      items:
        type: string
`)

	rule := &OwaspArrayLimitRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should detect missing array limit")

	var valErr *validation.Error
	require.ErrorAs(t, errs[0], &valErr)
	require.NotNil(t, valErr.Fix)
	fix := valErr.Fix

	// Simulate user entering maxItems = 100
	require.NoError(t, fix.SetInput([]string{"100"}))

	nodeFix, ok := fix.(validation.NodeFix)
	require.True(t, ok)
	require.NoError(t, nodeFix.ApplyNode(nil))

	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the array limit violation")
}

func TestFixIntegration_MissingPathParam(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        "200":
          description: OK
`)

	rule := &PathParamsRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.Len(t, errs, 1, "should detect missing userId param")

	// Extract and apply fix
	nodeFix := extractNodeFix(t, errs)
	var valErr0 *validation.Error
	require.ErrorAs(t, errs[0], &valErr0)
	assert.False(t, valErr0.Fix.Interactive(), "should be non-interactive")
	require.NoError(t, nodeFix.ApplyNode(nil))

	// Re-parse and verify violation is resolved
	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fix should resolve the missing path param violation")
}

func TestFixIntegration_MissingPathParam_TypeInference(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths:
  /users/{userId}/sessions/{sessionUuid}:
    get:
      operationId: getSession
      responses:
        "200":
          description: OK
`)

	rule := &PathParamsRule{}
	docInfo := buildDocInfo(t, doc)
	errs := rule.Run(ctx, docInfo, &linter.RuleConfig{})
	require.Len(t, errs, 2, "should detect both missing params")

	// Apply all fixes
	for _, err := range errs {
		var valErr *validation.Error
		require.ErrorAs(t, err, &valErr)
		require.NotNil(t, valErr.Fix, "should have a fix")
		nodeFix, ok := valErr.Fix.(validation.NodeFix)
		require.True(t, ok)
		require.NoError(t, nodeFix.ApplyNode(nil))
	}

	// Re-parse and verify
	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)
	errs2 := rule.Run(ctx, docInfo2, &linter.RuleConfig{})
	assert.Empty(t, errs2, "fixes should resolve all missing path param violations")
}

// ============================================================
// Fix engine integration test
// ============================================================

func TestFixIntegration_EngineAutoMode(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// A document with multiple non-interactive violations
	doc := parseOpenAPIDoc(t, `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
servers:
  - url: http://api.example.com/
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
`)

	docInfo := buildDocInfo(t, doc)

	// Run multiple rules to collect violations
	rules := []interface {
		Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error
	}{
		&OAS3HostTrailingSlashRule{},
		&OwaspSecurityHostsHttpsOAS3Rule{},
		&OwaspDefineErrorResponses401Rule{},
		&OwaspDefineErrorResponses500Rule{},
	}

	var allErrors []error
	config := &linter.RuleConfig{}
	for _, rule := range rules {
		errs := rule.Run(ctx, docInfo, config)
		allErrors = append(allErrors, errs...)
	}

	// Verify we have violations
	require.NotEmpty(t, allErrors, "should have multiple violations")

	// Collect fixable errors (non-interactive only)
	var fixCount int
	for _, err := range allErrors {
		var valErr *validation.Error
		if !errors.As(err, &valErr) || valErr.Fix == nil {
			continue
		}
		fix := valErr.Fix
		if fix.Interactive() {
			continue
		}
		// Apply the fix
		if nodeFix, ok := fix.(validation.NodeFix); ok {
			require.NoError(t, nodeFix.ApplyNode(nil))
			fixCount++
		}
	}
	require.Positive(t, fixCount, "should have applied at least one fix")

	// Re-parse and verify fixes resolved violations
	doc2 := remarshalAndParse(t, doc)
	docInfo2 := buildDocInfo(t, doc2)

	var remainingErrors []error
	for _, rule := range rules {
		errs := rule.Run(ctx, docInfo2, config)
		remainingErrors = append(remainingErrors, errs...)
	}
	assert.Less(t, len(remainingErrors), len(allErrors), "fixes should reduce the number of violations")
}
