package tests_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/converter"
	"github.com/speakeasy-api/openapi/openapi/linter/customrules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral tests: generate TypeScript from converter Rule, load via goja runtime,
// run against an OpenAPI doc, and assert expected validation errors.
// These catch logic bugs in codegen, not just syntax correctness.
//
// Tests use fields with known getters (summary, description, operationId) on
// collection-backed nodes to verify that codegen generates correct runtime code.

func TestBehavioral_TruthyRule(t *testing.T) {
	t.Parallel()

	rule := converter.Rule{
		ID:          "require-op-summary",
		Description: "Operations must have a summary",
		Severity:    "warn",
		Given:       []string{"$.paths[*][*]"},
		Then:        []converter.RuleCheck{{Field: "summary", Function: "truthy"}},
	}

	t.Run("finds violation when summary missing", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
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
`)
		require.Len(t, errs, 1, "should find 1 violation")
		assert.Contains(t, errs[0].Error(), "Operations must have a summary")
	})

	t.Run("no violation when summary present", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Get all users
      responses:
        '200':
          description: ok
`)
		assert.Empty(t, errs, "should find no violations")
	})
}

func TestBehavioral_PatternRule(t *testing.T) {
	t.Parallel()

	// Test pattern on info description (known getter) to avoid
	// dynamic field access issues in the runtime.
	rule := converter.Rule{
		ID:          "desc-format",
		Description: "Description must start with uppercase",
		Severity:    "error",
		Given:       []string{"$.paths[*][*]"},
		Then: []converter.RuleCheck{{
			Field:           "description",
			Function:        "pattern",
			FunctionOptions: map[string]any{"match": `^[A-Z]`},
		}},
	}

	t.Run("valid description passes", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      description: Returns a list of users
      responses:
        '200':
          description: ok
`)
		assert.Empty(t, errs, "uppercase description should pass")
	})

	t.Run("invalid description fails", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      description: returns a list of users
      responses:
        '200':
          description: ok
`)
		require.Len(t, errs, 1, "lowercase description should fail")
		assert.Contains(t, errs[0].Error(), "Description must start with uppercase")
	})
}

func TestBehavioral_EnumerationRule(t *testing.T) {
	t.Parallel()

	// Test enumeration on operation description (known getter) since
	// dynamic field access on server.url doesn't work in the goja runtime.
	rule := converter.Rule{
		ID:          "status-check",
		Description: "Operation description must be an approved value",
		Severity:    "error",
		Given:       []string{"$.paths[*][*]"},
		Then: []converter.RuleCheck{{
			Field:           "description",
			Function:        "enumeration",
			FunctionOptions: map[string]any{"values": []any{"Returns users", "Creates a user"}},
		}},
	}

	t.Run("approved value passes", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      description: Returns users
      responses:
        '200':
          description: ok
`)
		assert.Empty(t, errs, "approved description should pass")
	})

	t.Run("unapproved value fails", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      description: Does something
      responses:
        '200':
          description: ok
`)
		require.Len(t, errs, 1, "unapproved description should fail")
		assert.Contains(t, errs[0].Error(), "Operation description must be an approved value")
	})
}

func TestBehavioral_LengthRule(t *testing.T) {
	t.Parallel()

	rule := converter.Rule{
		ID:          "summary-length",
		Description: "Summary must be 5-50 chars",
		Severity:    "warn",
		Given:       []string{"$.paths[*][*]"},
		Then: []converter.RuleCheck{{
			Field:           "summary",
			Function:        "length",
			FunctionOptions: map[string]any{"min": 5, "max": 50},
		}},
	}

	t.Run("valid length passes", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Get all users
      responses:
        '200':
          description: ok
`)
		assert.Empty(t, errs, "valid length should pass")
	})

	t.Run("too short fails", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Hi
      responses:
        '200':
          description: ok
`)
		require.Len(t, errs, 1, "too short should fail")
		assert.Contains(t, errs[0].Error(), "Summary must be 5-50 chars")
	})
}

func TestBehavioral_MultipleThenChecks(t *testing.T) {
	t.Parallel()

	rule := converter.Rule{
		ID:          "op-complete",
		Description: "Operations must have summary and description",
		Severity:    "error",
		Given:       []string{"$.paths[*][*]"},
		Then: []converter.RuleCheck{
			{Field: "summary", Function: "truthy"},
			{Field: "description", Function: "truthy"},
		},
	}

	t.Run("both present passes", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Get users
      description: Returns all users
      responses:
        '200':
          description: ok
`)
		assert.Empty(t, errs, "both present should pass")
	})

	t.Run("both missing fails with two errors", func(t *testing.T) {
		t.Parallel()
		errs := runGeneratedRule(t, rule, `
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
`)
		require.Len(t, errs, 2, "should find 2 violations (one per missing field)")
	})
}

// runGeneratedRule generates TypeScript from a converter Rule, loads it into
// the goja runtime, runs it against the given OpenAPI YAML, and returns
// validation errors.
func runGeneratedRule(t *testing.T, rule converter.Rule, openAPIYAML string) []error {
	t.Helper()
	ctx := t.Context()

	// Generate TypeScript
	source, _ := converter.GenerateRuleTypeScript(rule, "custom-")

	// Write to temp file (the loader reads from files)
	tmpDir := t.TempDir()
	ruleFile := filepath.Join(tmpDir, "rule.ts")
	err := os.WriteFile(ruleFile, []byte(source), 0o644)
	require.NoError(t, err, "should write temp rule file")

	// Load via customrules loader
	loader := customrules.NewLoader(nil)
	rulesConfig := &linter.CustomRulesConfig{
		Paths: []string{ruleFile},
	}

	rules, err := loader.LoadRules(rulesConfig)
	require.NoError(t, err, "should load generated rule")
	require.Len(t, rules, 1, "should have exactly 1 rule")

	// Parse OpenAPI doc
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(openAPIYAML))
	require.NoError(t, err, "should parse OpenAPI YAML")

	// Build index
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Run the rule
	return rules[0].Run(ctx, docInfo, nil)
}
