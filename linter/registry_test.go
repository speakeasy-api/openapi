package linter_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterRuleset(t *testing.T) {
	t.Parallel()

	t.Run("successfully register ruleset", func(t *testing.T) {
		t.Parallel()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError})
		registry.Register(&mockRule{id: "rule-2", category: "style", defaultSeverity: validation.SeverityError})

		err := registry.RegisterRuleset("recommended", []string{"rule-1", "rule-2"})
		require.NoError(t, err)

		ruleIDs, exists := registry.GetRuleset("recommended")
		assert.True(t, exists)
		assert.ElementsMatch(t, []string{"rule-1", "rule-2"}, ruleIDs)
	})

	t.Run("error when rule not found", func(t *testing.T) {
		t.Parallel()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError})

		err := registry.RegisterRuleset("test", []string{"rule-1", "nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("error when ruleset already registered", func(t *testing.T) {
		t.Parallel()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError})

		err := registry.RegisterRuleset("test", []string{"rule-1"})
		require.NoError(t, err)

		err = registry.RegisterRuleset("test", []string{"rule-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestRegistry_AllCategories(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError})
	registry.Register(&mockRule{id: "rule-2", category: "style", defaultSeverity: validation.SeverityError})
	registry.Register(&mockRule{id: "rule-3", category: "security", defaultSeverity: validation.SeverityError})
	registry.Register(&mockRule{id: "rule-4", category: "best-practices", defaultSeverity: validation.SeverityError})

	categories := registry.AllCategories()
	// Should be sorted
	assert.Equal(t, []string{"best-practices", "security", "style"}, categories)
}

func TestRegistry_AllRulesets(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError})
	require.NoError(t, registry.RegisterRuleset("recommended", []string{"rule-1"}))
	require.NoError(t, registry.RegisterRuleset("strict", []string{"rule-1"}))

	rulesets := registry.AllRulesets()
	assert.Contains(t, rulesets, "all")
	assert.Contains(t, rulesets, "recommended")
	assert.Contains(t, rulesets, "strict")
	// Should be sorted
	assert.Equal(t, "all", rulesets[0])
}

func TestRegistry_RulesetsContaining(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{id: "rule-1", category: "style", defaultSeverity: validation.SeverityError})
	registry.Register(&mockRule{id: "rule-2", category: "security", defaultSeverity: validation.SeverityError})
	require.NoError(t, registry.RegisterRuleset("recommended", []string{"rule-1"}))
	require.NoError(t, registry.RegisterRuleset("strict", []string{"rule-1", "rule-2"}))

	t.Run("rule in multiple rulesets", func(t *testing.T) {
		t.Parallel()
		rulesets := registry.RulesetsContaining("rule-1")
		assert.Contains(t, rulesets, "all")
		assert.Contains(t, rulesets, "recommended")
		assert.Contains(t, rulesets, "strict")
	})

	t.Run("rule in subset of rulesets", func(t *testing.T) {
		t.Parallel()
		rulesets := registry.RulesetsContaining("rule-2")
		assert.Contains(t, rulesets, "all")
		assert.Contains(t, rulesets, "strict")
		assert.NotContains(t, rulesets, "recommended")
	})
}

func TestRegistry_GetRuleset_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	_, exists := registry.GetRuleset("nonexistent")
	assert.False(t, exists)
}

func TestRegistry_GetRule_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	_, exists := registry.GetRule("nonexistent")
	assert.False(t, exists)
}
