package linter

import (
	"fmt"
	"sort"
)

// Registry holds registered rules
type Registry[T any] struct {
	rules    map[string]RuleRunner[T]
	rulesets map[string][]string // ruleset name -> rule IDs
}

// NewRegistry creates a new rule registry
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		rules:    make(map[string]RuleRunner[T]),
		rulesets: make(map[string][]string),
	}
}

// Register registers a rule
func (r *Registry[T]) Register(rule RuleRunner[T]) {
	r.rules[rule.ID()] = rule
}

// RegisterRuleset registers a ruleset
func (r *Registry[T]) RegisterRuleset(name string, ruleIDs []string) error {
	if _, exists := r.rulesets[name]; exists {
		return fmt.Errorf("ruleset %q already registered", name)
	}

	// Validate rule IDs
	for _, id := range ruleIDs {
		if _, exists := r.rules[id]; !exists {
			return fmt.Errorf("rule %q in ruleset %q not found", id, name)
		}
	}

	r.rulesets[name] = ruleIDs
	return nil
}

// GetRule returns a rule by ID
func (r *Registry[T]) GetRule(id string) (RuleRunner[T], bool) {
	rule, ok := r.rules[id]
	return rule, ok
}

// GetRuleset returns rule IDs for a ruleset
func (r *Registry[T]) GetRuleset(name string) ([]string, bool) {
	if name == "all" {
		return r.AllRuleIDs(), true
	}
	ids, ok := r.rulesets[name]
	return ids, ok
}

// AllRules returns all registered rules
func (r *Registry[T]) AllRules() []RuleRunner[T] {
	rules := make([]RuleRunner[T], 0, len(r.rules))
	for _, rule := range r.rules {
		rules = append(rules, rule)
	}
	// Sort for deterministic order
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID() < rules[j].ID()
	})
	return rules
}

// AllRuleIDs returns all registered rule IDs
func (r *Registry[T]) AllRuleIDs() []string {
	ids := make([]string, 0, len(r.rules))
	for id := range r.rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// AllCategories returns all unique categories
func (r *Registry[T]) AllCategories() []string {
	categories := make(map[string]bool)
	for _, rule := range r.rules {
		categories[rule.Category()] = true
	}

	cats := make([]string, 0, len(categories))
	for cat := range categories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}

// AllRulesets returns all registered ruleset names
func (r *Registry[T]) AllRulesets() []string {
	names := make([]string, 0, len(r.rulesets)+1)
	names = append(names, "all")
	for name := range r.rulesets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RulesetsContaining returns names of rulesets that contain the given rule ID
func (r *Registry[T]) RulesetsContaining(ruleID string) []string {
	var sets []string

	// "all" always contains everything
	sets = append(sets, "all")

	for name, ids := range r.rulesets {
		for _, id := range ids {
			if id == ruleID {
				sets = append(sets, name)
				break
			}
		}
	}
	sort.Strings(sets)
	return sets
}
