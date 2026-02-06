package converter

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Parse reads a Spectral, Vacuum, or Legacy Speakeasy config and returns
// the intermediate representation. Parse is lenient: it collects as many
// rules as possible and adds warnings for malformed entries rather than
// failing entirely.
func Parse(r io.Reader) (*IntermediateConfig, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// First unmarshal into a raw map to detect format
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if raw == nil {
		return &IntermediateConfig{}, nil
	}

	// Detect format based on top-level keys
	if _, hasLintVersion := raw["lintVersion"]; hasLintVersion {
		if _, hasRulesets := raw["rulesets"]; hasRulesets {
			return parseLegacy(data)
		}
	}

	return parseSpectral(data)
}

// ParseFile reads a config file and returns the intermediate representation.
func ParseFile(path string) (*IntermediateConfig, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return Parse(f)
}

// parseSpectral parses Spectral/Vacuum format configs.
func parseSpectral(data []byte) (*IntermediateConfig, error) {
	var raw rawSpectralConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Spectral config: %w", err)
	}

	ir := &IntermediateConfig{}

	// Parse extends
	ir.Extends = raw.parseExtends()

	// Parse rules individually for leniency — a malformed rule should not
	// fail the entire parse. The RulesNode is a YAML mapping node.
	if raw.RulesNode.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(raw.RulesNode.Content); i += 2 {
			keyNode := raw.RulesNode.Content[i]
			valueNode := raw.RulesNode.Content[i+1]
			id := keyNode.Value

			var rawRule rawSpectralRuleRef
			if err := valueNode.Decode(&rawRule); err != nil {
				ir.Warnings = append(ir.Warnings, Warning{
					RuleID:  id,
					Phase:   "parse",
					Message: fmt.Sprintf("failed to decode rule: %v — skipped", err),
				})
				continue
			}

			rule, warnings := rawRule.toRule(id, "")
			if rule != nil {
				ir.Rules = append(ir.Rules, *rule)
			}
			ir.Warnings = append(ir.Warnings, warnings...)
		}
	}

	// Warn about unsupported top-level fields
	if len(raw.Formats) > 0 {
		ir.Warnings = append(ir.Warnings, Warning{
			Phase:   "parse",
			Message: fmt.Sprintf("top-level formats %v are not supported — per-rule formats are preserved; configure version targeting manually", raw.Formats),
		})
	}
	if len(raw.Overrides) > 0 {
		ir.Warnings = append(ir.Warnings, Warning{
			Phase:   "parse",
			Message: "overrides are not supported — apply overrides manually in the native config",
		})
	}
	if raw.FunctionsDir != "" {
		ir.Warnings = append(ir.Warnings, Warning{
			Phase:   "parse",
			Message: fmt.Sprintf("functionsDir %q is not supported — custom function rules will need manual implementation", raw.FunctionsDir),
		})
	}
	if len(raw.Functions) > 0 {
		ir.Warnings = append(ir.Warnings, Warning{
			Phase:   "parse",
			Message: fmt.Sprintf("functions %v are not supported — custom function rules will need manual implementation", raw.Functions),
		})
	}

	return ir, nil
}

// parseLegacy parses legacy Speakeasy lint.yaml format.
func parseLegacy(data []byte) (*IntermediateConfig, error) {
	var raw rawLegacyConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	ir := &IntermediateConfig{}

	// Warn about defaultRuleset — it's not mapped to the native format.
	if raw.DefaultRuleset != "" {
		ir.Warnings = append(ir.Warnings, Warning{
			Phase:   "parse",
			Message: fmt.Sprintf("defaultRuleset %q is not directly mapped — configure extends manually if needed", raw.DefaultRuleset),
		})
	}

	// Sort ruleset names for deterministic iteration order.
	// Go map iteration is non-deterministic; sorting ensures stable output
	// when multiple rulesets define the same rule (last wins in Generate).
	rulesetNames := make([]string, 0, len(raw.Rulesets))
	for name := range raw.Rulesets {
		rulesetNames = append(rulesetNames, name)
	}
	sort.Strings(rulesetNames)

	// Flatten nested rulesets
	for _, rulesetName := range rulesetNames {
		ruleset := raw.Rulesets[rulesetName]

		// Each referenced ruleset becomes an extends entry
		for _, ref := range ruleset.Rulesets {
			ir.Extends = append(ir.Extends, ExtendsEntry{
				Name: ref,
			})
		}

		// Sort rule IDs within each ruleset for deterministic order
		ruleIDs := make([]string, 0, len(ruleset.Rules))
		for ruleID := range ruleset.Rules {
			ruleIDs = append(ruleIDs, ruleID)
		}
		sort.Strings(ruleIDs)

		// Each rule in the ruleset becomes an IR rule
		for _, ruleID := range ruleIDs {
			rawRule := ruleset.Rules[ruleID]
			rule, warnings := rawRule.toRule(ruleID, rulesetName)
			if rule != nil {
				ir.Rules = append(ir.Rules, *rule)
			}
			ir.Warnings = append(ir.Warnings, warnings...)
		}
	}

	return ir, nil
}

// --- Raw Spectral/Vacuum YAML types ---

type rawSpectralConfig struct {
	Extends      rawExtends `yaml:"extends"`
	RulesNode    yaml.Node  `yaml:"rules"`        // decoded per-rule for leniency
	Formats      []string   `yaml:"formats"`      // top-level; per-rule formats preserved in Rule.Formats
	Overrides    []any      `yaml:"overrides"`    // unsupported, warned
	Functions    []string   `yaml:"functions"`    // unsupported, warned
	FunctionsDir string     `yaml:"functionsDir"` // unsupported, warned
}

// rawExtends handles the polymorphic "extends" field.
// Can be: string | string[] | [string, string][]
type rawExtends struct {
	entries []ExtendsEntry
}

func (e *rawExtends) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// Simple string: "spectral:oas"
		if value.Tag == "!!null" {
			return nil
		}
		e.entries = append(e.entries, ExtendsEntry{Name: value.Value})
		return nil
	case yaml.SequenceNode:
		for _, item := range value.Content {
			switch item.Kind {
			case yaml.ScalarNode:
				// String in list: ["spectral:oas"]
				e.entries = append(e.entries, ExtendsEntry{Name: item.Value})
			case yaml.SequenceNode:
				// Tuple: [["spectral:oas", "recommended"]]
				if len(item.Content) >= 1 {
					entry := ExtendsEntry{Name: item.Content[0].Value}
					if len(item.Content) >= 2 {
						entry.Modifier = item.Content[1].Value
					}
					e.entries = append(e.entries, entry)
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("extends: expected string, list, or tuple list, got %v", value.Kind)
	}
}

func (c *rawSpectralConfig) parseExtends() []ExtendsEntry {
	return c.Extends.entries
}

// rawSpectralRuleRef handles the polymorphic rule definition.
// Can be: string (severity only) | number (numeric severity) | bool | object (full rule)
type rawSpectralRuleRef struct {
	isOverride bool
	severity   string
	rule       *rawSpectralRule
}

func (r *rawSpectralRuleRef) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// String severity override: "error", "warn", "off", etc.
		// Or numeric: 0, 1, 2, 3
		// Or boolean: true (use default), false (off)
		switch value.Tag {
		case "!!bool":
			r.isOverride = true
			if value.Value == "false" {
				r.severity = "off"
			}
			// true = use default, empty severity means "leave as default"
			return nil
		case "!!int":
			r.isOverride = true
			r.severity = normalizeNumericSeverity(value.Value)
			return nil
		default:
			r.isOverride = true
			r.severity = normalizeSeverity(value.Value)
			return nil
		}
	case yaml.MappingNode:
		// Full rule definition
		var rule rawSpectralRule
		if err := value.Decode(&rule); err != nil {
			return fmt.Errorf("failed to decode rule: %w", err)
		}
		r.rule = &rule
		return nil
	default:
		return fmt.Errorf("rule: expected string, number, bool, or object, got %v", value.Kind)
	}
}

type rawSpectralRule struct {
	Description string   `yaml:"description"`
	Message     string   `yaml:"message"`
	Severity    string   `yaml:"severity"`
	Recommended *bool    `yaml:"recommended"`
	Formats     []string `yaml:"formats"`
	Resolved    *bool    `yaml:"resolved"`
	Given       rawGiven `yaml:"given"`
	Then        rawThen  `yaml:"then"`
	Enabled     *bool    `yaml:"enabled"` // Vacuum extension
}

// rawGiven handles "given" as string or string[].
type rawGiven struct {
	paths []string
}

func (g *rawGiven) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		g.paths = []string{value.Value}
		return nil
	case yaml.SequenceNode:
		var paths []string
		if err := value.Decode(&paths); err != nil {
			return err
		}
		g.paths = paths
		return nil
	default:
		return fmt.Errorf("given: expected string or string[], got %v", value.Kind)
	}
}

// rawThen handles "then" as object or object[].
type rawThen struct {
	checks []rawThenEntry
}

func (t *rawThen) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var entry rawThenEntry
		if err := value.Decode(&entry); err != nil {
			return err
		}
		t.checks = []rawThenEntry{entry}
		return nil
	case yaml.SequenceNode:
		var entries []rawThenEntry
		if err := value.Decode(&entries); err != nil {
			return err
		}
		t.checks = entries
		return nil
	default:
		return fmt.Errorf("then: expected object or object[], got %v", value.Kind)
	}
}

type rawThenEntry struct {
	Field           string         `yaml:"field"`
	Function        string         `yaml:"function"`
	FunctionOptions map[string]any `yaml:"functionOptions"`
	Message         string         `yaml:"message"` // Vacuum extension
}

func (ref *rawSpectralRuleRef) toRule(id, source string) (*Rule, []Warning) {
	var warnings []Warning

	if ref.isOverride {
		return &Rule{
			ID:       id,
			Severity: ref.severity,
			Source:   source,
		}, nil
	}

	if ref.rule == nil {
		warnings = append(warnings, Warning{
			RuleID:  id,
			Phase:   "parse",
			Message: "rule has no definition",
		})
		return nil, warnings
	}

	raw := ref.rule

	// Determine severity
	severity := normalizeSeverity(raw.Severity)
	if severity == "" {
		severity = "warn" // Spectral default
	}

	// Handle enabled: false (Vacuum extension)
	if raw.Enabled != nil && !*raw.Enabled {
		severity = "off"
	}

	// Handle recommended: false
	if raw.Recommended != nil && !*raw.Recommended {
		severity = "off"
	}

	rule := &Rule{
		ID:          id,
		Description: raw.Description,
		Message:     raw.Message,
		Severity:    severity,
		Resolved:    raw.Resolved,
		Formats:     raw.Formats,
		Source:      source,
	}

	// Parse given paths
	if len(raw.Given.paths) == 0 {
		warnings = append(warnings, Warning{
			RuleID:  id,
			Phase:   "parse",
			Message: "rule has no 'given' paths",
		})
	} else {
		rule.Given = raw.Given.paths
	}

	// Parse then checks
	if len(raw.Then.checks) == 0 {
		warnings = append(warnings, Warning{
			RuleID:  id,
			Phase:   "parse",
			Message: "rule has no 'then' checks",
		})
	}
	for _, entry := range raw.Then.checks {
		if entry.Function == "" {
			warnings = append(warnings, Warning{
				RuleID:  id,
				Phase:   "parse",
				Message: "then entry has no function",
			})
			continue
		}
		rule.Then = append(rule.Then, RuleCheck(entry))
	}

	return rule, warnings
}

// --- Raw Legacy Speakeasy YAML types ---

type rawLegacyConfig struct {
	LintVersion    string                      `yaml:"lintVersion"`
	DefaultRuleset string                      `yaml:"defaultRuleset"`
	Rulesets       map[string]rawLegacyRuleset `yaml:"rulesets"`
}

type rawLegacyRuleset struct {
	Rulesets []string                      `yaml:"rulesets"`
	Rules    map[string]rawSpectralRuleRef `yaml:"rules"`
}

// --- Severity normalization ---

// normalizeSeverity converts various severity representations to canonical form.
func normalizeSeverity(s string) string {
	switch s {
	case "error":
		return "error"
	case "warn", "warning":
		return "warn"
	case "info":
		return "info"
	case "hint":
		return "hint"
	case "off", "false":
		return "off"
	default:
		// Try numeric
		if n, err := strconv.Atoi(s); err == nil {
			return numericToSeverity(n)
		}
		return s // preserve unknown values
	}
}

// normalizeNumericSeverity converts a numeric string severity to canonical form.
func normalizeNumericSeverity(s string) string {
	n, err := strconv.Atoi(s)
	if err != nil {
		return s
	}
	return numericToSeverity(n)
}

// numericToSeverity converts Spectral's numeric severity to string.
// 0=error, 1=warn, 2=info, 3=hint
func numericToSeverity(n int) string {
	switch n {
	case 0:
		return "error"
	case 1:
		return "warn"
	case 2:
		return "info"
	case 3:
		return "hint"
	default:
		return "warn" // safe default
	}
}
