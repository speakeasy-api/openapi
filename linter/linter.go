package linter

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/speakeasy-api/openapi/linter/format"
	"github.com/speakeasy-api/openapi/validation"
)

// Linter is the main linting engine
type Linter[T any] struct {
	config   *Config
	registry *Registry[T]
}

// NewLinter creates a new linter with the given configuration
func NewLinter[T any](config *Config, registry *Registry[T]) *Linter[T] {
	return &Linter[T]{
		config:   config,
		registry: registry,
	}
}

// Registry returns the rule registry for documentation generation
func (l *Linter[T]) Registry() *Registry[T] {
	return l.registry
}

// Lint runs all configured rules against the document
func (l *Linter[T]) Lint(ctx context.Context, docInfo *DocumentInfo[T], preExistingErrors []error, opts *LintOptions) (*Output, error) {
	var allErrs []error

	if len(preExistingErrors) > 0 {
		allErrs = append(allErrs, preExistingErrors...)
	}

	// Run lint rules - these also return validation.Error instances
	lintErrs := l.runRules(ctx, docInfo, opts)
	allErrs = append(allErrs, lintErrs...)

	// Apply severity overrides from config
	allErrs = l.applySeverityOverrides(allErrs)

	// Sort errors by location
	validation.SortValidationErrors(allErrs)

	// Format output
	return l.formatOutput(allErrs), nil
}

func (l *Linter[T]) runRules(ctx context.Context, docInfo *DocumentInfo[T], opts *LintOptions) []error {
	// Determine enabled rules
	enabledRules := l.getEnabledRules()

	// Run rules in parallel for better performance
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	for _, rule := range enabledRules {
		ruleConfig := l.getRuleConfig(rule.ID())

		// Skip if disabled (though getEnabledRules should handle this, double check)
		if ruleConfig.Enabled != nil && !*ruleConfig.Enabled {
			continue
		}

		// Filter rules based on version if VersionFilter is set
		if opts != nil && opts.VersionFilter != nil && *opts.VersionFilter != "" {
			ruleVersions := rule.Versions()
			// If rule specifies versions, check if current version matches
			if len(ruleVersions) > 0 {
				versionMatches := false
				for _, ruleVersion := range ruleVersions {
					// Match against rule's supported versions
					// Support both "3.1" and "3.1.0" formats
					if ruleVersion == *opts.VersionFilter ||
					   (len(*opts.VersionFilter) > len(ruleVersion) &&
					    (*opts.VersionFilter)[:len(ruleVersion)] == ruleVersion) {
						versionMatches = true
						break
					}
				}
				if !versionMatches {
					continue // Skip this rule - doesn't apply to this version
				}
			}
			// If rule.Versions() is nil/empty, it applies to all versions
		}

		// Set resolve options if provided
		if opts != nil && opts.ResolveOptions != nil {
			resolveOpts := *opts.ResolveOptions
			// Set document location as target location if not already set
			if resolveOpts.TargetLocation == "" && docInfo.Location != "" {
				resolveOpts.TargetLocation = docInfo.Location
			}
			ruleConfig.ResolveOptions = &resolveOpts
		}

		// Run rule in parallel
		wg.Add(1)
		go func(r RuleRunner[T], cfg RuleConfig) {
			defer wg.Done()

			ruleErrs := r.Run(ctx, docInfo, &cfg)

			mu.Lock()
			errs = append(errs, ruleErrs...)
			mu.Unlock()
		}(rule, ruleConfig)
	}

	wg.Wait()
	return errs
}

func (l *Linter[T]) getEnabledRules() []RuleRunner[T] {
	// Start with all rules if "all" is extended (default)
	// Or specific rulesets

	// For now, simple implementation: check config for enabled rules
	// If config.Extends contains "all", include all rules unless disabled

	// Map to track enabled status: ruleID -> enabled
	ruleStatus := make(map[string]bool)

	// Apply rulesets
	for _, ruleset := range l.config.Extends {
		if ids, ok := l.registry.GetRuleset(ruleset); ok {
			for _, id := range ids {
				ruleStatus[id] = true
			}
		}
	}

	// Apply category config
	// Category config overrides ruleset config but is overridden by individual rule config
	for _, rule := range l.registry.AllRules() {
		if catConfig, ok := l.config.Categories[rule.Category()]; ok {
			if catConfig.Enabled != nil {
				ruleStatus[rule.ID()] = *catConfig.Enabled
			}
		}
	}

	// Apply rule config
	for id, ruleConfig := range l.config.Rules {
		if ruleConfig.Enabled != nil {
			ruleStatus[id] = *ruleConfig.Enabled
		}
	}

	var enabled []RuleRunner[T]
	for id, enabledFlag := range ruleStatus {
		if enabledFlag {
			if rule, ok := l.registry.GetRule(id); ok {
				enabled = append(enabled, rule)
			}
		}
	}

	// Sort for deterministic order
	sort.Slice(enabled, func(i, j int) bool {
		return enabled[i].ID() < enabled[j].ID()
	})

	return enabled
}

func (l *Linter[T]) getRuleConfig(ruleID string) RuleConfig {
	// Start with default config
	config := RuleConfig{}

	// Apply category config
	if rule, ok := l.registry.GetRule(ruleID); ok {
		if catConfig, ok := l.config.Categories[rule.Category()]; ok {
			if catConfig.Severity != nil {
				config.Severity = catConfig.Severity
			}
		}
	}

	// Apply rule config
	if ruleConfig, ok := l.config.Rules[ruleID]; ok {
		if ruleConfig.Severity != nil {
			config.Severity = ruleConfig.Severity
		}
		if ruleConfig.Options != nil {
			config.Options = ruleConfig.Options
		}
	}

	return config
}

func (l *Linter[T]) applySeverityOverrides(errs []error) []error {
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			config := l.getRuleConfig(vErr.Rule)
			if config.Severity != nil {
				vErr.Severity = *config.Severity
			}
		}
	}
	return errs
}

func (l *Linter[T]) formatOutput(errs []error) *Output {
	return &Output{
		Results: errs,
		Format:  l.config.OutputFormat,
	}
}

// Output represents the result of linting
type Output struct {
	Results []error
	Format  OutputFormat
}

func (o *Output) HasErrors() bool {
	for _, err := range o.Results {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			if vErr.Severity == validation.SeverityError {
				return true
			}
		} else {
			// Non-validation errors are treated as errors
			return true
		}
	}
	return false
}

func (o *Output) ErrorCount() int {
	count := 0
	for _, err := range o.Results {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			if vErr.Severity == validation.SeverityError {
				count++
			}
		} else {
			count++
		}
	}
	return count
}

func (o *Output) FormatText() string {
	f := format.NewTextFormatter()
	s, _ := f.Format(o.Results)
	return s
}

func (o *Output) FormatJSON() string {
	f := format.NewJSONFormatter()
	s, _ := f.Format(o.Results)
	return s
}
