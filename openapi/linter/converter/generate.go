package converter

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v4"
)

// GenerateOptions configures the output generation.
type GenerateOptions struct {
	// RulesDir is the relative path for .ts files in config (default "./rules").
	RulesDir string

	// RulePrefix is the prefix for generated rule IDs (default "custom-").
	RulePrefix string
}

// GenerateOption is a functional option for Generate.
type GenerateOption func(*GenerateOptions)

// WithRulesDir sets the rules directory.
func WithRulesDir(dir string) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.RulesDir = dir
	}
}

// WithRulePrefix sets the rule ID prefix.
func WithRulePrefix(prefix string) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.RulePrefix = prefix
	}
}

func defaultOptions() GenerateOptions {
	return GenerateOptions{
		RulesDir:   "./rules",
		RulePrefix: "custom-",
	}
}

// GenerateResult holds all generated output.
type GenerateResult struct {
	// Config is the native lint config, serializable to YAML.
	Config *linter.Config

	// GeneratedRules maps ruleID -> TypeScript source code.
	GeneratedRules map[string]string

	// Warnings from the generation phase.
	Warnings []Warning

	// rulesDir is the directory for generated .ts files (relative to output dir).
	rulesDir string
}

// WriteFiles writes lint.yaml and rules/*.ts to the output directory.
func (r *GenerateResult) WriteFiles(outputDir string) error {
	// Write lint.yaml
	configData, err := yaml.Marshal(r.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configPath := filepath.Join(outputDir, "lint.yaml")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write lint.yaml: %w", err)
	}

	// Write rule .ts files
	if len(r.GeneratedRules) > 0 {
		dir := r.rulesDir
		if dir == "" {
			dir = "rules"
		}
		rulesDir := filepath.Join(outputDir, dir)
		if err := os.MkdirAll(rulesDir, 0o755); err != nil { //nolint:gosec
			return fmt.Errorf("failed to create rules directory: %w", err)
		}

		ruleIDs := make([]string, 0, len(r.GeneratedRules))
		for ruleID := range r.GeneratedRules {
			ruleIDs = append(ruleIDs, ruleID)
		}
		sort.Strings(ruleIDs)

		for _, ruleID := range ruleIDs {
			source := r.GeneratedRules[ruleID]
			filename := ruleID + ".ts"
			rulePath := filepath.Join(rulesDir, filename)
			if err := os.WriteFile(rulePath, []byte(source), 0o644); err != nil { //nolint:gosec
				return fmt.Errorf("failed to write rule file %s: %w", filename, err)
			}
		}
	}

	return nil
}

// Generate converts an IntermediateConfig into native linter output.
// This is stage 2 of the pipeline — all native-format-specific interpretation
// happens here.
func Generate(ir *IntermediateConfig, opts ...GenerateOption) (*GenerateResult, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	result := &GenerateResult{
		rulesDir: options.RulesDir,
		Config: &linter.Config{
			Rules:      []linter.RuleEntry{},
			Categories: make(map[string]linter.CategoryConfig),
		},
		GeneratedRules: make(map[string]string),
	}

	// Carry forward parse-phase warnings
	result.Warnings = append(result.Warnings, ir.Warnings...)

	// 1. Map extends
	result.Config.Extends = mapExtends(ir.Extends, &result.Warnings)

	// 2. Track seen rules for deduplication (last occurrence wins)
	seen := make(map[string]int) // ruleID -> index in result list

	// 3. Classify and process each rule
	for _, rule := range ir.Rules {
		if rule.IsOverride() {
			processOverride(rule, result, seen)
		} else {
			processCustomRule(rule, result, options, seen)
		}
	}

	// 4. Add custom_rules config if we generated any rules
	if len(result.GeneratedRules) > 0 {
		result.Config.CustomRules = &linter.CustomRulesConfig{
			Paths: []string{path.Join(options.RulesDir, "*.ts")},
		}
	}

	// Default extends if empty
	if len(result.Config.Extends) == 0 {
		result.Config.Extends = []string{"all"}
	}

	return result, nil
}

// mapExtends interprets IR extends entries for the native format.
func mapExtends(entries []ExtendsEntry, warnings *[]Warning) []string {
	var extends []string
	for _, entry := range entries {
		switch entry.Name {
		case "spectral:oas":
			switch entry.Modifier {
			case "recommended":
				extends = append(extends, "recommended")
			case "off":
				// Disable = don't extend
				*warnings = append(*warnings, Warning{
					Phase:   "generate",
					Message: fmt.Sprintf("extends %q with modifier %q was disabled — not included in native config", entry.Name, entry.Modifier),
				})
			default:
				// "all" or empty -> extends all
				extends = append(extends, "all")
			}
		case "spectral:asyncapi":
			*warnings = append(*warnings, Warning{
				Phase:   "generate",
				Message: fmt.Sprintf("extends %q is not supported in native format — skipped", entry.Name),
			})
		case "speakeasy-recommended":
			extends = append(extends, "recommended")
		case "speakeasy-generation":
			extends = append(extends, "all")
		default:
			// Unknown extends value: pass through with warning
			extends = append(extends, entry.Name)
			*warnings = append(*warnings, Warning{
				Phase:   "generate",
				Message: fmt.Sprintf("unknown extends %q — passed through as-is; may not be valid in native config", entry.Name),
			})
		}
	}
	return extends
}

// processOverride handles severity-only rule overrides.
func processOverride(rule Rule, result *GenerateResult, seen map[string]int) {
	// Look up native rule ID
	nativeID, found := LookupNativeRule(rule.ID)
	if !found {
		// Unmapped rule: disable and warn
		nativeID = "unmapped-" + rule.ID
		result.Warnings = append(result.Warnings, Warning{
			RuleID:  rule.ID,
			Phase:   "generate",
			Message: fmt.Sprintf("no native equivalent for rule %q — added as disabled", rule.ID),
		})

		disabled := true
		entry := linter.RuleEntry{
			ID:       nativeID,
			Disabled: &disabled,
		}

		if idx, ok := seen[nativeID]; ok {
			result.Warnings = append(result.Warnings, Warning{
				RuleID:  rule.ID,
				Phase:   "generate",
				Message: fmt.Sprintf("duplicate rule %q (last occurrence wins)", nativeID),
			})
			result.Config.Rules[idx] = entry
		} else {
			seen[nativeID] = len(result.Config.Rules)
			result.Config.Rules = append(result.Config.Rules, entry)
		}
		return
	}

	// Empty severity (from `rule: true`) means "use default" — skip override
	if rule.Severity == "" {
		return
	}

	// Build native rule entry
	entry := linter.RuleEntry{ID: nativeID}

	if rule.IsDisabled() {
		disabled := true
		entry.Disabled = &disabled
	} else {
		nativeSev := mapSeverityToNative(rule.Severity)
		sev := toValidationSeverity(nativeSev)
		entry.Severity = &sev
	}

	// Dedup: last occurrence wins
	if idx, ok := seen[nativeID]; ok {
		result.Warnings = append(result.Warnings, Warning{
			RuleID:  rule.ID,
			Phase:   "generate",
			Message: fmt.Sprintf("duplicate override for %q (last occurrence wins)", nativeID),
		})
		result.Config.Rules[idx] = entry
	} else {
		seen[nativeID] = len(result.Config.Rules)
		result.Config.Rules = append(result.Config.Rules, entry)
	}
}

// processCustomRule generates TypeScript for a full rule definition.
func processCustomRule(rule Rule, result *GenerateResult, opts GenerateOptions, seen map[string]int) {
	ruleID := opts.RulePrefix + rule.ID

	// Warn about resolved: false
	if rule.Resolved != nil && !*rule.Resolved {
		result.Warnings = append(result.Warnings, Warning{
			RuleID:  rule.ID,
			Phase:   "generate",
			Message: "rule uses resolved: false, but native linter always operates on resolved documents",
		})
	}

	// Generate TypeScript
	source, codegenWarnings := GenerateRuleTypeScript(rule, opts.RulePrefix)
	result.Warnings = append(result.Warnings, codegenWarnings...)

	// Dedup: last occurrence wins
	if _, exists := result.GeneratedRules[ruleID]; exists {
		result.Warnings = append(result.Warnings, Warning{
			RuleID:  rule.ID,
			Phase:   "generate",
			Message: fmt.Sprintf("duplicate custom rule %q (last occurrence wins)", ruleID),
		})
	}
	result.GeneratedRules[ruleID] = source

	// Add severity override to config if non-default
	nativeSev := mapSeverityToNative(rule.Severity)
	if rule.IsDisabled() {
		disabled := true
		entry := linter.RuleEntry{ID: ruleID, Disabled: &disabled}
		if idx, ok := seen[ruleID]; ok {
			result.Config.Rules[idx] = entry
		} else {
			seen[ruleID] = len(result.Config.Rules)
			result.Config.Rules = append(result.Config.Rules, entry)
		}
	} else if nativeSev != "warning" {
		sev := toValidationSeverity(nativeSev)
		entry := linter.RuleEntry{ID: ruleID, Severity: &sev}
		if idx, ok := seen[ruleID]; ok {
			result.Config.Rules[idx] = entry
		} else {
			seen[ruleID] = len(result.Config.Rules)
			result.Config.Rules = append(result.Config.Rules, entry)
		}
	}
}

// toValidationSeverity converts a native severity string to validation.Severity.
func toValidationSeverity(s string) validation.Severity {
	switch s {
	case "error":
		return validation.SeverityError
	case "warning":
		return validation.SeverityWarning
	case "hint":
		return validation.SeverityHint
	default:
		return validation.SeverityWarning
	}
}
