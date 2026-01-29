package linter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock document type for testing
type MockDoc struct {
	ID string
}

// Mock rule for testing
type mockRule struct {
	id              string
	category        string
	description     string
	link            string
	defaultSeverity validation.Severity
	versions        []string
	runFunc         func(ctx context.Context, docInfo *linter.DocumentInfo[*MockDoc], config *linter.RuleConfig) []error
}

func (r *mockRule) ID() string                           { return r.id }
func (r *mockRule) Category() string                     { return r.category }
func (r *mockRule) Description() string                  { return r.description }
func (r *mockRule) Link() string                         { return r.link }
func (r *mockRule) DefaultSeverity() validation.Severity { return r.defaultSeverity }
func (r *mockRule) Versions() []string                   { return r.versions }

func (r *mockRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*MockDoc], config *linter.RuleConfig) []error {
	if r.runFunc != nil {
		return r.runFunc(ctx, docInfo, config)
	}
	return nil
}

func TestLinter_RuleSelection(t *testing.T) {
	t.Parallel()

	t.Run("extends all includes all rules", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "test-rule-1",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "test-rule-1", errors.New("test error"), nil)}
			},
		})
		registry.Register(&mockRule{
			id:              "test-rule-2",
			category:        "security",
			defaultSeverity: validation.SeverityWarning,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityWarning, "test-rule-2", errors.New("test warning"), nil)}
			},
		})

		config := &linter.Config{
			Extends: []string{"all"},
		}

		lntr := linter.NewLinter(config, registry)
		docInfo := &linter.DocumentInfo[*MockDoc]{
			Document: &MockDoc{ID: "test"},
		}

		output, err := lntr.Lint(ctx, docInfo, nil, nil)
		require.NoError(t, err)

		// Should have errors from both rules
		assert.Len(t, output.Results, 2)
	})

	t.Run("disabled rule not executed", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "test-rule-1",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "test-rule-1", errors.New("test error"), nil)}
			},
		})

		falseVal := false
		config := &linter.Config{
			Extends: []string{"all"},
			Rules: map[string]linter.RuleConfig{
				"test-rule-1": {
					Enabled: &falseVal,
				},
			},
		}

		lntr := linter.NewLinter(config, registry)
		docInfo := &linter.DocumentInfo[*MockDoc]{
			Document: &MockDoc{ID: "test"},
		}

		output, err := lntr.Lint(ctx, docInfo, nil, nil)
		require.NoError(t, err)

		// Should have no errors since rule is disabled
		assert.Empty(t, output.Results)
	})

	t.Run("category disabled affects all rules in category", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "style-rule-1",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "style-rule-1", errors.New("style error 1"), nil)}
			},
		})
		registry.Register(&mockRule{
			id:              "style-rule-2",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "style-rule-2", errors.New("style error 2"), nil)}
			},
		})
		registry.Register(&mockRule{
			id:              "security-rule-1",
			category:        "security",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "security-rule-1", errors.New("security error"), nil)}
			},
		})

		falseVal := false
		config := &linter.Config{
			Extends: []string{"all"},
			Categories: map[string]linter.CategoryConfig{
				"style": {
					Enabled: &falseVal,
				},
			},
		}

		lntr := linter.NewLinter(config, registry)
		docInfo := &linter.DocumentInfo[*MockDoc]{
			Document: &MockDoc{ID: "test"},
		}

		output, err := lntr.Lint(ctx, docInfo, nil, nil)
		require.NoError(t, err)

		// Should only have security error, style rules disabled
		require.Len(t, output.Results, 1)
		assert.Contains(t, output.Results[0].Error(), "security-rule-1")
	})
}

func TestLinter_SeverityOverrides(t *testing.T) {
	t.Parallel()

	t.Run("rule severity override", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "test-rule",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("test error"), nil)}
			},
		})

		warningSeverity := validation.SeverityWarning
		config := &linter.Config{
			Extends: []string{"all"},
			Rules: map[string]linter.RuleConfig{
				"test-rule": {
					Severity: &warningSeverity,
				},
			},
		}

		lntr := linter.NewLinter(config, registry)
		docInfo := &linter.DocumentInfo[*MockDoc]{
			Document: &MockDoc{ID: "test"},
		}

		output, err := lntr.Lint(ctx, docInfo, nil, nil)
		require.NoError(t, err)

		require.Len(t, output.Results, 1)
		var vErr *validation.Error
		require.ErrorAs(t, output.Results[0], &vErr)
		assert.Equal(t, validation.SeverityWarning, vErr.Severity)
	})

	t.Run("category severity override", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "style-rule",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "style-rule", errors.New("style error"), nil)}
			},
		})

		warningSeverity := validation.SeverityWarning
		config := &linter.Config{
			Extends: []string{"all"},
			Categories: map[string]linter.CategoryConfig{
				"style": {
					Severity: &warningSeverity,
				},
			},
		}

		lntr := linter.NewLinter(config, registry)
		docInfo := &linter.DocumentInfo[*MockDoc]{
			Document: &MockDoc{ID: "test"},
		}

		output, err := lntr.Lint(ctx, docInfo, nil, nil)
		require.NoError(t, err)

		require.Len(t, output.Results, 1)
		var vErr *validation.Error
		require.ErrorAs(t, output.Results[0], &vErr)
		assert.Equal(t, validation.SeverityWarning, vErr.Severity)
	})

	t.Run("rule severity override takes precedence over category", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		registry := linter.NewRegistry[*MockDoc]()
		registry.Register(&mockRule{
			id:              "style-rule",
			category:        "style",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, "style-rule", errors.New("style error"), nil)}
			},
		})

		warningSeverity := validation.SeverityWarning
		hintSeverity := validation.SeverityHint
		config := &linter.Config{
			Extends: []string{"all"},
			Categories: map[string]linter.CategoryConfig{
				"style": {
					Severity: &warningSeverity,
				},
			},
			Rules: map[string]linter.RuleConfig{
				"style-rule": {
					Severity: &hintSeverity,
				},
			},
		}

		lntr := linter.NewLinter(config, registry)
		docInfo := &linter.DocumentInfo[*MockDoc]{
			Document: &MockDoc{ID: "test"},
		}

		output, err := lntr.Lint(ctx, docInfo, nil, nil)
		require.NoError(t, err)

		require.Len(t, output.Results, 1)
		var vErr *validation.Error
		require.ErrorAs(t, output.Results[0], &vErr)
		// Rule severity should override category severity
		assert.Equal(t, validation.SeverityHint, vErr.Severity)
	})
}

func TestLinter_PreExistingErrors(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{
		id:              "test-rule",
		category:        "style",
		defaultSeverity: validation.SeverityError,
		runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
			return []error{validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("lint error"), nil)}
		},
	})

	config := &linter.Config{
		Extends: []string{"all"},
	}

	lntr := linter.NewLinter(config, registry)
	docInfo := &linter.DocumentInfo[*MockDoc]{
		Document: &MockDoc{ID: "test"},
	}

	preExistingErrs := []error{
		validation.NewValidationError(validation.SeverityError, "validation-required", errors.New("validation error"), nil),
	}

	output, err := lntr.Lint(ctx, docInfo, preExistingErrs, nil)
	require.NoError(t, err)

	// Should include both pre-existing and lint errors
	assert.Len(t, output.Results, 2)
}

func TestLinter_ParallelExecution(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	registry := linter.NewRegistry[*MockDoc]()

	// Create multiple rules that all run
	for i := 0; i < 10; i++ {
		ruleID := fmt.Sprintf("test-rule-%d", i)
		registry.Register(&mockRule{
			id:              ruleID,
			category:        "test",
			defaultSeverity: validation.SeverityError,
			runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
				return []error{validation.NewValidationError(validation.SeverityError, ruleID, fmt.Errorf("error from %s", ruleID), nil)}
			},
		})
	}

	config := &linter.Config{
		Extends: []string{"all"},
	}

	lntr := linter.NewLinter(config, registry)
	docInfo := &linter.DocumentInfo[*MockDoc]{
		Document: &MockDoc{ID: "test"},
	}

	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Should have errors from all 10 rules
	assert.Len(t, output.Results, 10)

	// Verify all rules executed
	foundRules := make(map[string]bool)
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) {
			foundRules[vErr.Rule] = true
		}
	}
	assert.Len(t, foundRules, 10, "all rules should have executed")
}

func TestOutput_HasErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		results   []error
		hasErrors bool
	}{
		{
			name:      "no errors",
			results:   []error{},
			hasErrors: false,
		},
		{
			name: "only warnings",
			results: []error{
				validation.NewValidationError(validation.SeverityWarning, "test-rule", errors.New("warning"), nil),
			},
			hasErrors: false,
		},
		{
			name: "only hints",
			results: []error{
				validation.NewValidationError(validation.SeverityHint, "test-rule", errors.New("hint"), nil),
			},
			hasErrors: false,
		},
		{
			name: "has error severity",
			results: []error{
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("error"), nil),
			},
			hasErrors: true,
		},
		{
			name: "mixed severities with error",
			results: []error{
				validation.NewValidationError(validation.SeverityWarning, "test-rule", errors.New("warning"), nil),
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("error"), nil),
			},
			hasErrors: true,
		},
		{
			name: "non-validation error treated as error",
			results: []error{
				errors.New("plain error"),
			},
			hasErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := &linter.Output{
				Results: tt.results,
			}

			assert.Equal(t, tt.hasErrors, output.HasErrors())
		})
	}
}

func TestOutput_ErrorCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		results    []error
		errorCount int
	}{
		{
			name:       "no errors",
			results:    []error{},
			errorCount: 0,
		},
		{
			name: "only warnings",
			results: []error{
				validation.NewValidationError(validation.SeverityWarning, "test-rule", errors.New("warning"), nil),
			},
			errorCount: 0,
		},
		{
			name: "one error",
			results: []error{
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("error"), nil),
			},
			errorCount: 1,
		},
		{
			name: "mixed severities",
			results: []error{
				validation.NewValidationError(validation.SeverityWarning, "test-rule", errors.New("warning"), nil),
				validation.NewValidationError(validation.SeverityError, "test-rule-1", errors.New("error 1"), nil),
				validation.NewValidationError(validation.SeverityHint, "test-rule", errors.New("hint"), nil),
				validation.NewValidationError(validation.SeverityError, "test-rule-2", errors.New("error 2"), nil),
			},
			errorCount: 2,
		},
		{
			name: "non-validation errors counted",
			results: []error{
				errors.New("plain error 1"),
				validation.NewValidationError(validation.SeverityWarning, "test-rule", errors.New("warning"), nil),
				errors.New("plain error 2"),
			},
			errorCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := &linter.Output{
				Results: tt.results,
			}

			assert.Equal(t, tt.errorCount, output.ErrorCount())
		})
	}
}

func TestOutput_Formatting(t *testing.T) {
	t.Parallel()

	output := &linter.Output{
		Results: []error{
			validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("test error"), nil),
		},
		Format: linter.OutputFormatText,
	}

	t.Run("format text non-empty", func(t *testing.T) {
		t.Parallel()
		text := output.FormatText()
		assert.NotEmpty(t, text)
		assert.Contains(t, text, "test-rule")
	})

	t.Run("format json non-empty", func(t *testing.T) {
		t.Parallel()
		json := output.FormatJSON()
		assert.NotEmpty(t, json)
		assert.Contains(t, json, "test-rule")
	})
}

func TestLinter_ErrorSorting(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{
		id:              "test-rule",
		category:        "style",
		defaultSeverity: validation.SeverityError,
		runFunc: func(_ context.Context, _ *linter.DocumentInfo[*MockDoc], _ *linter.RuleConfig) []error {
			// Return errors in unsorted order
			return []error{
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("error 3"), nil),
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("error 1"), nil),
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("error 2"), nil),
			}
		},
	})

	config := &linter.Config{
		Extends: []string{"all"},
	}

	lntr := linter.NewLinter(config, registry)
	docInfo := &linter.DocumentInfo[*MockDoc]{
		Document: &MockDoc{ID: "test"},
	}

	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Errors should be sorted by validation.SortValidationErrors
	assert.Len(t, output.Results, 3)
}

func TestLinter_Registry(t *testing.T) {
	t.Parallel()

	registry := linter.NewRegistry[*MockDoc]()
	registry.Register(&mockRule{
		id:              "test-rule",
		category:        "style",
		defaultSeverity: validation.SeverityError,
	})

	config := &linter.Config{}
	lntr := linter.NewLinter(config, registry)

	// Should be able to access registry for documentation
	reg := lntr.Registry()
	require.NotNil(t, reg)

	rule, exists := reg.GetRule("test-rule")
	assert.True(t, exists)
	assert.Equal(t, "test-rule", rule.ID())
}
