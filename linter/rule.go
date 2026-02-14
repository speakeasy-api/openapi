package linter

import (
	"context"

	"github.com/speakeasy-api/openapi/validation"
)

// Rule represents a single linting rule
type Rule interface {
	// ID returns the unique identifier for this rule (e.g., "style-path-params")
	ID() string

	// Category returns the rule category (e.g., "style", "validation", "security")
	Category() string

	// Description returns a human-readable description of what the rule checks
	Description() string

	// Summary returns a short summary of what the rule checks
	Summary() string

	// Link returns an optional URL to documentation for this rule
	Link() string

	// DefaultSeverity returns the default severity level for this rule
	DefaultSeverity() validation.Severity

	// Versions returns the spec versions this rule applies to (nil = all versions).
	// Supports exact versions ("3.1") and prefix matching ("3.0" matches "3.0.x").
	Versions() []string
}

// RuleRunner is the interface rules must implement to execute their logic
// This is separate from Rule to allow different runner types for different specs
type RuleRunner[T any] interface {
	Rule

	// Run executes the rule against the provided document
	// DocumentInfo provides both the document and its location for resolving external references
	// Returns any issues found as validation errors
	Run(ctx context.Context, docInfo *DocumentInfo[T], config *RuleConfig) []error
}

// DocumentedRule provides extended documentation for a rule
type DocumentedRule interface {
	Rule

	// GoodExample returns YAML showing correct usage
	GoodExample() string

	// BadExample returns YAML showing incorrect usage
	BadExample() string

	// Rationale explains why this rule exists
	Rationale() string

	// FixAvailable returns true if the rule provides auto-fix suggestions
	FixAvailable() bool
}

// ConfigurableRule indicates a rule has configurable options
type ConfigurableRule interface {
	Rule

	// ConfigSchema returns JSON Schema for rule-specific options
	ConfigSchema() map[string]any

	// ConfigDefaults returns default values for options
	ConfigDefaults() map[string]any
}
