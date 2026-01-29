package linter

import (
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
)

// Config represents the linter configuration
type Config struct {
	// Extends specifies rulesets to extend (e.g., "recommended", "all")
	Extends []string `yaml:"extends,omitempty" json:"extends,omitempty"`

	// Rules contains per-rule configuration
	Rules map[string]RuleConfig `yaml:"rules,omitempty" json:"rules,omitempty"`

	// Categories contains per-category configuration
	Categories map[string]CategoryConfig `yaml:"categories,omitempty" json:"categories,omitempty"`

	// Ignores contains global ignore patterns
	Ignores []IgnorePattern `yaml:"ignores,omitempty" json:"ignores,omitempty"`

	// OutputFormat specifies the output format
	OutputFormat OutputFormat `yaml:"output_format,omitempty" json:"output_format,omitempty"`
}

// RuleConfig configures a specific rule
type RuleConfig struct {
	// Enabled controls whether the rule is active
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Severity overrides the default severity
	Severity *validation.Severity `yaml:"severity,omitempty" json:"severity,omitempty"`

	// Options contains rule-specific configuration
	Options map[string]any `yaml:"options,omitempty" json:"options,omitempty"`

	// ResolveOptions contains runtime options for reference resolution (not serialized)
	// These are set by the linter engine when running rules
	ResolveOptions *references.ResolveOptions `yaml:"-" json:"-"`
}

// GetSeverity returns the effective severity, falling back to default if not overridden
func (c *RuleConfig) GetSeverity(defaultSeverity validation.Severity) validation.Severity {
	if c != nil && c.Severity != nil {
		return *c.Severity
	}
	return defaultSeverity
}

// CategoryConfig configures an entire category of rules
type CategoryConfig struct {
	// Enabled controls whether all rules in the category are active
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Severity overrides the default severity for all rules in the category
	Severity *validation.Severity `yaml:"severity,omitempty" json:"severity,omitempty"`
}

// IgnorePattern specifies a pattern for ignoring results
type IgnorePattern struct {
	// Rule is the rule ID to ignore (empty = all rules)
	Rule string `yaml:"rule,omitempty" json:"rule,omitempty"`

	// Path is a JSON pointer pattern to match
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// Message pattern to match (regex)
	MessagePattern string `yaml:"message_pattern,omitempty" json:"message_pattern,omitempty"`
}

type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
)

// NewConfig creates a new default configuration
func NewConfig() *Config {
	return &Config{
		Extends:      []string{"all"},
		Rules:        make(map[string]RuleConfig),
		Categories:   make(map[string]CategoryConfig),
		OutputFormat: OutputFormatText,
	}
}
