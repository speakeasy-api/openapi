package linter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// Config represents the linter configuration
type Config struct {
	// Extends specifies rulesets to extend (e.g., "recommended", "all")
	Extends []string `yaml:"extends,omitempty" json:"extends,omitempty"`

	// Rules contains per-rule configuration
	Rules []RuleEntry `yaml:"rules,omitempty" json:"rules,omitempty"`

	// Categories contains per-category configuration
	Categories map[string]CategoryConfig `yaml:"categories,omitempty" json:"categories,omitempty"`

	// OutputFormat specifies the output format
	OutputFormat OutputFormat `yaml:"output_format,omitempty" json:"output_format,omitempty"`
}

// UnmarshalYAML supports "extends" as string or list and severity aliases.
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Extends      yaml.Node                 `yaml:"extends,omitempty"`
		Rules        []RuleEntry               `yaml:"rules,omitempty"`
		Categories   map[string]CategoryConfig `yaml:"categories,omitempty"`
		OutputFormat OutputFormat              `yaml:"output_format,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	if raw.Extends.Kind != 0 {
		switch raw.Extends.Kind {
		case yaml.ScalarNode:
			switch raw.Extends.Tag {
			case "!!null":
				c.Extends = nil
			case "!!str", "":
				c.Extends = []string{raw.Extends.Value}
			default:
				return errors.New("extends must be a string or list of strings")
			}
		case yaml.SequenceNode:
			var list []string
			if err := raw.Extends.Decode(&list); err != nil {
				return err
			}
			c.Extends = list
		default:
			return errors.New("extends must be a string or list of strings")
		}
	}

	c.Rules = raw.Rules
	c.Categories = raw.Categories
	c.OutputFormat = raw.OutputFormat
	return nil
}

// RuleEntry configures rule behavior in lint.yaml.
type RuleEntry struct {
	ID       string               `yaml:"id" json:"id"`
	Severity *validation.Severity `yaml:"severity,omitempty" json:"severity,omitempty"`
	Disabled *bool                `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Match    *string              `yaml:"match,omitempty" json:"match,omitempty"`
}

// UnmarshalYAML allows severity aliases (warn, info) in rule entries.
func (r *RuleEntry) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		ID       string  `yaml:"id"`
		Severity *string `yaml:"severity,omitempty"`
		Disabled *bool   `yaml:"disabled,omitempty"`
		Match    *string `yaml:"match,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	r.ID = raw.ID
	r.Disabled = raw.Disabled
	r.Match = raw.Match
	if raw.Severity != nil {
		sev, err := parseSeverity(*raw.Severity)
		if err != nil {
			return err
		}
		r.Severity = &sev
	}
	return nil
}

// RuleConfig configures a specific rule
type RuleConfig struct {
	// Enabled controls whether the rule is active
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Severity overrides the default severity
	Severity *validation.Severity `yaml:"severity,omitempty" json:"severity,omitempty"`

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

// UnmarshalYAML allows severity aliases (warn, info) in categories.
func (c *CategoryConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Enabled  *bool   `yaml:"enabled,omitempty"`
		Severity *string `yaml:"severity,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if raw.Severity != nil {
		sev, err := parseSeverity(*raw.Severity)
		if err != nil {
			return err
		}
		c.Severity = &sev
	}
	c.Enabled = raw.Enabled
	return nil
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
		Rules:        []RuleEntry{},
		Categories:   make(map[string]CategoryConfig),
		OutputFormat: OutputFormatText,
	}
}

func parseSeverity(value string) (validation.Severity, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "error":
		return validation.SeverityError, nil
	case "warn", "warning":
		return validation.SeverityWarning, nil
	case "hint", "info":
		return validation.SeverityHint, nil
	default:
		return "", fmt.Errorf("unknown severity %q", value)
	}
}
