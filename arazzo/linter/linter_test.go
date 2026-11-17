package linter_test

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	arazzoLinter "github.com/speakeasy-api/openapi/arazzo/linter"
	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixture parses a lint fixture without validation.
func loadFixture(t *testing.T, name string) *baseLinter.DocumentInfo[*arazzo.Arazzo] {
	t.Helper()

	f, err := os.Open("../testdata/lint/" + name)
	require.NoError(t, err, "fixture should open")
	t.Cleanup(func() { _ = f.Close() })

	a, _, err := arazzo.Unmarshal(t.Context(), f, arazzo.WithSkipValidation())
	require.NoError(t, err, "fixture should parse")
	return baseLinter.NewDocumentInfo(a, "../testdata/lint/"+name)
}

// ruleIDs returns the rule ID of each result.
func ruleIDs(t *testing.T, results []error) []string {
	t.Helper()

	ids := make([]string, 0, len(results))
	for _, err := range results {
		var vErr *validation.Error
		require.ErrorAs(t, err, &vErr, "result should be a validation error")
		ids = append(ids, vErr.Rule)
	}
	return ids
}

var allRuleIDs = []string{
	rules.RuleSemanticUnreachableStep,
	rules.RuleSemanticDependencyCycle,
	rules.RuleSemanticDependencyCycle,
	rules.RuleSemanticUnusedComponent,
	rules.RuleSemanticUnusedComponent,
	rules.RuleSemanticUnusedWorkflowInput,
	rules.RuleSemanticUnusedStepOutput,
	rules.RuleStyleStepSuccessCriteria,
	rules.RuleStyleSourceDescriptionType,
	rules.RuleStyleInfoDescription,
	rules.RuleStyleWorkflowDescription,
	rules.RuleStyleRetryAfter,
	rules.RuleSecuritySourceDescriptionHTTPS,
}

func TestNewLinter_Registry_Success(t *testing.T) {
	t.Parallel()

	registry := arazzoLinter.NewLinter(nil).Registry()
	assert.Len(t, registry.AllRules(), 11, "all built-in rules should be registered")

	recommended, ok := registry.GetRuleset("recommended")
	require.True(t, ok, "recommended ruleset should be registered")
	assert.Len(t, recommended, 8, "recommended should leave out the hint rules")
	assert.NotContains(t, recommended, rules.RuleStyleRetryAfter, "recommended should leave out hints")

	assert.Empty(t, arazzoLinter.NewLinter(nil, arazzoLinter.WithoutDefaultRules()).Registry().AllRules(), "no rules should be registered")
}

func TestLinter_Lint_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *baseLinter.Config
		expected []string
	}{
		{
			name:     "all rules",
			config:   nil,
			expected: allRuleIDs,
		},
		{
			name:   "recommended ruleset",
			config: &baseLinter.Config{Extends: []string{"recommended"}},
			expected: []string{
				rules.RuleSemanticUnreachableStep,
				rules.RuleSemanticDependencyCycle,
				rules.RuleSemanticDependencyCycle,
				rules.RuleSemanticUnusedComponent,
				rules.RuleSemanticUnusedComponent,
				rules.RuleSemanticUnusedWorkflowInput,
				rules.RuleStyleStepSuccessCriteria,
				rules.RuleStyleSourceDescriptionType,
				rules.RuleStyleInfoDescription,
				rules.RuleSecuritySourceDescriptionHTTPS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := arazzoLinter.NewLinter(tt.config).Lint(t.Context(), loadFixture(t, "issues.arazzo.yaml"), nil, nil)
			require.NoError(t, err, "lint should succeed")
			assert.ElementsMatch(t, tt.expected, ruleIDs(t, output.Results), "findings should match")
		})
	}
}

func TestLinter_Lint_Clean_Success(t *testing.T) {
	t.Parallel()

	output, err := arazzoLinter.NewLinter(nil).Lint(t.Context(), loadFixture(t, "clean.arazzo.yaml"), nil, &baseLinter.LintOptions{})
	require.NoError(t, err, "lint should succeed")
	assert.Empty(t, output.Results, "clean document should have no findings")
}

func TestLinter_Lint_WithoutDefaultRules_Success(t *testing.T) {
	t.Parallel()

	l := arazzoLinter.NewLinter(nil, arazzoLinter.WithoutDefaultRules())
	l.Registry().Register(&rules.InfoDescriptionRule{})

	preExisting := &validation.Error{UnderlyingError: errors.New("invalid"), Severity: validation.SeverityError, Rule: "validation-required-field"}
	output, err := l.Lint(t.Context(), loadFixture(t, "issues.arazzo.yaml"), []error{preExisting}, nil)
	require.NoError(t, err, "lint should succeed")
	assert.ElementsMatch(t, []string{"validation-required-field", rules.RuleStyleInfoDescription}, ruleIDs(t, output.Results), "only registered rules and existing errors should be reported")
	assert.True(t, output.HasErrors(), "pre-existing errors should be kept")
}

func TestLinter_FilterErrors_Success(t *testing.T) {
	t.Parallel()

	l := arazzoLinter.NewLinter(&baseLinter.Config{
		Extends: []string{"all"},
		Rules:   []baseLinter.RuleEntry{{ID: rules.RuleStyleInfoDescription, Disabled: pointer.From(true)}},
	})
	errs := []error{
		&validation.Error{UnderlyingError: errors.New("no description"), Severity: validation.SeverityWarning, Rule: rules.RuleStyleInfoDescription},
		&validation.Error{UnderlyingError: errors.New("http"), Severity: validation.SeverityWarning, Rule: rules.RuleSecuritySourceDescriptionHTTPS},
	}
	assert.Equal(t, []string{rules.RuleSecuritySourceDescriptionHTTPS}, ruleIDs(t, l.FilterErrors(errs)), "disabled rules should be filtered")
}

// offlineClient fails every request, so tests never reach the network.
type offlineClient struct{}

func (offlineClient) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
}

func TestWithSourceDocuments_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	docInfo := loadFixture(t, "issues.arazzo.yaml")
	sources, _ := arazzo.LoadSourceDescriptions(ctx, docInfo.Document, arazzo.LoadSourcesOptions{
		DocumentLocation: docInfo.Location,
		HTTPClient:       offlineClient{},
	})

	l := arazzoLinter.NewLinter(nil, arazzoLinter.WithoutDefaultRules())
	l.Registry().Register(&rules.SourceDescriptionTypeRule{})
	output, err := l.Lint(arazzoLinter.WithSourceDocuments(ctx, sources), docInfo, nil, nil)
	require.NoError(t, err, "lint should succeed")
	require.Len(t, output.Results, 1, "untyped source should be reported")

	var vErr *validation.Error
	require.ErrorAs(t, output.Results[0], &vErr, "result should be a validation error")
	require.NotNil(t, vErr.Fix, "should have a fix")
	assert.False(t, vErr.Fix.Interactive(), "fix should use the loaded source's type")
}
