package linter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocGenerator_GenerateRuleDoc(t *testing.T) {
	t.Parallel()

	t.Run("basic rule documentation", func(t *testing.T) {
		t.Parallel()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "test-rule",
			category:        "style",
			summary:         "Test rule summary",
			description:     "Test rule description",
			link:            "https://example.com/rules/test-rule",
			defaultSeverity: validation.SeverityError,
			versions:        []string{"3.1.0", "3.2.0"},
		})

		generator := linter.NewDocGenerator(registry)
		rule, _ := registry.GetRule("test-rule")
		doc := generator.GenerateRuleDoc(rule)

		assert.Equal(t, "test-rule", doc.ID)
		assert.Equal(t, "style", doc.Category)
		assert.Equal(t, "Test rule summary", doc.Summary)
		assert.Equal(t, "Test rule description", doc.Description)
		assert.Equal(t, "https://example.com/rules/test-rule", doc.Link)
		assert.Equal(t, "error", doc.DefaultSeverity)
		assert.Equal(t, []string{"3.1.0", "3.2.0"}, doc.Versions)
		assert.Contains(t, doc.Rulesets, "all")
	})

	t.Run("documented rule with examples", func(t *testing.T) {
		t.Parallel()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&documentedMockRule{
			mockRule: mockRule{
				id:              "documented-rule",
				category:        "style",
				summary:         "Documented rule summary",
				description:     "Rule with examples",
				defaultSeverity: validation.SeverityWarning,
			},
			goodExample:  "good:\n  example: value",
			badExample:   "bad:\n  example: value",
			rationale:    "This is why the rule exists",
			fixAvailable: true,
		})

		generator := linter.NewDocGenerator(registry)
		rule, _ := registry.GetRule("documented-rule")
		doc := generator.GenerateRuleDoc(rule)

		assert.Equal(t, "good:\n  example: value", doc.GoodExample)
		assert.Equal(t, "bad:\n  example: value", doc.BadExample)
		assert.Equal(t, "This is why the rule exists", doc.Rationale)
		assert.True(t, doc.FixAvailable)
	})

	t.Run("configurable rule with schema", func(t *testing.T) {
		t.Parallel()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&configurableMockRule{
			mockRule: mockRule{
				id:              "configurable-rule",
				category:        "style",
				summary:         "Configurable rule summary",
				description:     "Configurable rule",
				defaultSeverity: validation.SeverityError,
			},
			configSchema: map[string]any{
				"maxLength": map[string]any{"type": "integer"},
			},
			configDefaults: map[string]any{
				"maxLength": 100,
			},
		})

		generator := linter.NewDocGenerator(registry)
		rule, _ := registry.GetRule("configurable-rule")
		doc := generator.GenerateRuleDoc(rule)

		assert.NotNil(t, doc.ConfigSchema)
		assert.Contains(t, doc.ConfigSchema, "maxLength")
		assert.NotNil(t, doc.ConfigDefaults)
		assert.Equal(t, 100, doc.ConfigDefaults["maxLength"])
	})
}

func TestDocGenerator_GenerateAllRuleDocs(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError, description: "Rule 1"})
	registry.Register(&mockRule{id: "rule-2", category: "security", defaultSeverity: validation.SeverityWarning, description: "Rule 2"})
	registry.Register(&mockRule{id: "rule-3", category: "style", defaultSeverity: validation.SeverityHint, description: "Rule 3"})

	generator := linter.NewDocGenerator(registry)
	docs := generator.GenerateAllRuleDocs()

	assert.Len(t, docs, 3)

	// Verify all rules are documented
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	assert.ElementsMatch(t, []string{"rule-1", "rule-2", "rule-3"}, ids)
}

func TestDocGenerator_GenerateCategoryDocs(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{id: "style-1", category: "style", defaultSeverity: validation.SeverityError, description: "Style 1"})
	registry.Register(&mockRule{id: "style-2", category: "style", defaultSeverity: validation.SeverityError, description: "Style 2"})
	registry.Register(&mockRule{id: "security-1", category: "security", defaultSeverity: validation.SeverityError, description: "Security 1"})

	generator := linter.NewDocGenerator(registry)
	categoryDocs := generator.GenerateCategoryDocs()

	assert.Len(t, categoryDocs, 2)
	assert.Len(t, categoryDocs["style"], 2)
	assert.Len(t, categoryDocs["security"], 1)

	// Verify correct grouping
	styleIDs := []string{categoryDocs["style"][0].ID, categoryDocs["style"][1].ID}
	assert.ElementsMatch(t, []string{"style-1", "style-2"}, styleIDs)
	assert.Equal(t, "security-1", categoryDocs["security"][0].ID)
}

func TestDocGenerator_WriteJSON(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{
		id:              "test-rule",
		category:        "style",
		summary:         "Test rule summary",
		description:     "Test description",
		link:            "https://example.com",
		defaultSeverity: validation.SeverityError,
	})
	_ = registry.RegisterRuleset("recommended", []string{"test-rule"})

	generator := linter.NewDocGenerator(registry)

	var buf bytes.Buffer
	err := generator.WriteJSON(&buf)
	require.NoError(t, err)

	// Verify valid JSON
	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, result, "rules")
	assert.Contains(t, result, "categories")
	assert.Contains(t, result, "rulesets")

	// Verify rules array
	rules, ok := result["rules"].([]any)
	require.True(t, ok)
	assert.Len(t, rules, 1)

	// Verify rule details
	ruleMap, ok := rules[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-rule", ruleMap["id"])
	assert.Equal(t, "style", ruleMap["category"])
}

func TestDocGenerator_WriteMarkdown(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&documentedMockRule{
		mockRule: mockRule{
			id:              "test-rule",
			category:        "style",
			summary:         "Test rule summary",
			description:     "Test rule description",
			link:            "https://docs.example.com/rules/test-rule",
			defaultSeverity: validation.SeverityError,
		},
		goodExample:  "good:\n  value: correct",
		badExample:   "bad:\n  value: incorrect",
		rationale:    "This rule ensures consistency",
		fixAvailable: true,
	})

	generator := linter.NewDocGenerator(registry)

	var buf bytes.Buffer
	err := generator.WriteMarkdown(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Verify markdown structure
	assert.Contains(t, output, "# Lint Rules Reference")
	assert.Contains(t, output, "## Categories")
	assert.Contains(t, output, "## style")      // Category header
	assert.Contains(t, output, "### test-rule") // Rule header
	assert.Contains(t, output, "**Severity:** error")
	assert.Contains(t, output, "**Category:** style")
	assert.Contains(t, output, "**Summary:** Test rule summary")
	assert.Contains(t, output, "Test rule description")
	assert.Contains(t, output, "#### Rationale")
	assert.Contains(t, output, "This rule ensures consistency")
	assert.Contains(t, output, "#### ❌ Incorrect")
	assert.Contains(t, output, "bad:\n  value: incorrect")
	assert.Contains(t, output, "#### ✅ Correct")
	assert.Contains(t, output, "good:\n  value: correct")
	assert.Contains(t, output, "**Auto-fix available:** Yes")
	assert.Contains(t, output, "[Documentation →](https://docs.example.com/rules/test-rule)")
	assert.Contains(t, output, "---") // Separator
}

func TestDocGenerator_WriteMarkdown_WithVersions(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{
		id:              "versioned-rule",
		category:        "validation",
		summary:         "Versioned rule summary",
		description:     "Version-specific rule",
		defaultSeverity: validation.SeverityError,
		versions:        []string{"3.1.0", "3.2.0"},
	})

	generator := linter.NewDocGenerator(registry)

	var buf bytes.Buffer
	err := generator.WriteMarkdown(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "**Applies to:** 3.1.0, 3.2.0")
}

func TestDocGenerator_WriteMarkdown_OptionalSectionsAndErrors(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&configurableDocumentedRule{documentedMockRule: documentedMockRule{
		mockRule: mockRule{
			id: "full-rule", category: "style", summary: "Rule summary", description: "Rule description",
			link: "https://example.com/rule", defaultSeverity: validation.SeverityWarning, versions: []string{"3.1.0"},
		},
		goodExample: "good: true", badExample: "bad: true", rationale: "Rule rationale", fixAvailable: true,
	}})
	generator := linter.NewDocGenerator(registry)
	writeErr := errors.New("documentation writer failed")
	complete := &docFailureWriter{err: writeErr}
	require.NoError(t, generator.WriteMarkdown(complete), "fully documented rules should render")
	for _, section := range []string{"**Auto-fix available:** Yes", "#### Rationale", "Rule rationale", "#### ❌ Incorrect", "bad: true", "#### ✅ Correct", "good: true", "#### Configuration", "| Option | Type | Default | Description |", "[Documentation →](https://example.com/rule)"} {
		assert.Contains(t, complete.String(), section, "optional section should be included")
	}

	for failAt := 1; failAt <= complete.calls; failAt++ {
		t.Run("write_"+strconv.Itoa(failAt), func(t *testing.T) {
			t.Parallel()
			writer := &docFailureWriter{failAt: failAt, err: writeErr}
			err := generator.WriteMarkdown(writer)
			require.ErrorIs(t, err, writeErr, "each failed write should preserve the original error")
			assert.Equal(t, failAt, writer.calls, "documentation generation should stop immediately after a write failure")
		})
	}
}

func TestDocGenerator_WriteJSON_Error(t *testing.T) {
	t.Parallel()

	generator := linter.NewDocGenerator(linter.NewRegistry[*MockDoc]())
	writeErr := errors.New("JSON writer failed")
	writer := &docFailureWriter{failAt: 1, err: writeErr}
	require.ErrorIs(t, generator.WriteJSON(writer), writeErr, "JSON generation should preserve writer errors")
}

type docFailureWriter struct {
	bytes.Buffer
	calls  int
	failAt int
	err    error
}

func (w *docFailureWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, w.err
	}
	return w.Buffer.Write(data)
}

type configurableDocumentedRule struct {
	documentedMockRule
}

func (r *configurableDocumentedRule) ConfigSchema() map[string]any {
	return map[string]any{"enabled": map[string]any{"type": "boolean"}}
}

func (r *configurableDocumentedRule) ConfigDefaults() map[string]any {
	return map[string]any{"enabled": true}
}

// documentedMockRule implements DocumentedRule interface
type documentedMockRule struct {
	mockRule
	goodExample  string
	badExample   string
	rationale    string
	fixAvailable bool
}

func (r *documentedMockRule) GoodExample() string { return r.goodExample }
func (r *documentedMockRule) BadExample() string  { return r.badExample }
func (r *documentedMockRule) Rationale() string   { return r.rationale }
func (r *documentedMockRule) FixAvailable() bool  { return r.fixAvailable }

// configurableMockRule implements ConfigurableRule interface
type configurableMockRule struct {
	mockRule
	configSchema   map[string]any
	configDefaults map[string]any
}

func (r *configurableMockRule) ConfigSchema() map[string]any   { return r.configSchema }
func (r *configurableMockRule) ConfigDefaults() map[string]any { return r.configDefaults }
