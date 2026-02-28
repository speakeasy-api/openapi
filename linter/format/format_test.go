package format_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter/format"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// testFix is a minimal validation.Fix for testing formatter output.
type testFix struct {
	description string
	interactive bool
}

func (f *testFix) Description() string          { return f.description }
func (f *testFix) Interactive() bool            { return f.interactive }
func (f *testFix) Prompts() []validation.Prompt { return nil }
func (f *testFix) SetInput([]string) error      { return nil }
func (f *testFix) Apply(any) error              { return nil }

func TestTextFormatter_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errors   []error
		contains []string
	}{
		{
			name:     "empty errors",
			errors:   []error{},
			contains: []string{},
		},
		{
			name: "single error",
			errors: []error{
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("test error message"), nil),
			},
			contains: []string{"error", "test-rule", "test error message"},
		},
		{
			name: "multiple errors with different severities",
			errors: []error{
				validation.NewValidationError(validation.SeverityError, "error-rule", errors.New("error message"), nil),
				validation.NewValidationError(validation.SeverityWarning, "warning-rule", errors.New("warning message"), nil),
				validation.NewValidationError(validation.SeverityHint, "hint-rule", errors.New("hint message"), nil),
			},
			contains: []string{
				"error", "error-rule", "error message",
				"warning", "warning-rule", "warning message",
				"hint", "hint-rule", "hint message",
			},
		},
		{
			name: "error with line number",
			errors: []error{
				&validation.Error{
					UnderlyingError: errors.New("at specific location"),
					Node:            &yaml.Node{Line: 42, Column: 10},
					Severity:        validation.SeverityError,
					Rule:            "location-rule",
				},
			},
			contains: []string{"42", "10", "location-rule"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			formatter := format.NewTextFormatter()
			result, err := formatter.Format(tt.errors)
			require.NoError(t, err)

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr, "output should contain %q", substr)
			}
		})
	}
}

func TestTextFormatter_Format_ColumnAlignment(t *testing.T) {
	t.Parallel()

	formatter := format.NewTextFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError: errors.New("first"),
			Node:            &yaml.Node{Line: 1, Column: 1},
			Severity:        validation.SeverityWarning,
			Rule:            "short-rule",
		},
		&validation.Error{
			UnderlyingError: errors.New("second"),
			Node:            &yaml.Node{Line: 1200, Column: 20},
			Severity:        validation.SeverityWarning,
			Rule:            "longer-rule",
		},
	})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 2, "should have at least 2 output lines")

	// location is right-aligned to width 7 ("1200:20"), severity left-aligned to width 7 ("warning"),
	// rule left-aligned to width 11 ("longer-rule")
	assert.Equal(t, "    1:1 warning short-rule  first", lines[0], "first line should be padded to align columns")
	assert.Equal(t, "1200:20 warning longer-rule second", lines[1], "second line should fill location column exactly")

	// Verify severity column starts at the same index in both lines
	severityIdx0 := strings.Index(lines[0], "warning")
	severityIdx1 := strings.Index(lines[1], "warning")
	assert.Equal(t, severityIdx0, severityIdx1, "severity column should start at the same character index")
}

func TestJSONFormatter_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errors   []error
		contains []string
	}{
		{
			name:     "empty errors",
			errors:   []error{},
			contains: []string{`"results"`, `"summary"`},
		},
		{
			name: "single error",
			errors: []error{
				validation.NewValidationError(validation.SeverityError, "test-rule", errors.New("test error message"), nil),
			},
			contains: []string{`"error"`, `"test-rule"`, `"test error message"`},
		},
		{
			name: "multiple errors",
			errors: []error{
				validation.NewValidationError(validation.SeverityError, "rule-1", errors.New("error 1"), nil),
				validation.NewValidationError(validation.SeverityWarning, "rule-2", errors.New("error 2"), nil),
			},
			contains: []string{
				`"rule-1"`, `"error 1"`,
				`"rule-2"`, `"error 2"`,
				`"warning"`,
			},
		},
		{
			name: "error with location",
			errors: []error{
				&validation.Error{
					UnderlyingError: errors.New("located error"),
					Node:            &yaml.Node{Line: 15, Column: 25},
					Severity:        validation.SeverityError,
					Rule:            "location-rule",
				},
			},
			contains: []string{`"line": 15`, `"column": 25`, `"location-rule"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			formatter := format.NewJSONFormatter()
			result, err := formatter.Format(tt.errors)
			require.NoError(t, err)

			// Verify it's valid JSON by checking structure (it's an object, not an array)
			assert.True(t, strings.HasPrefix(strings.TrimSpace(result), "{"), "should start with {")
			assert.True(t, strings.HasSuffix(strings.TrimSpace(result), "}"), "should end with }")

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr, "JSON should contain %q", substr)
			}
		})
	}
}

func TestTextFormatter_FixableMarker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fix           validation.Fix
		shouldHave    string
		shouldNotHave string
	}{
		{
			name:       "error with fix shows fixable marker",
			fix:        &testFix{description: "auto fix", interactive: false},
			shouldHave: "[fixable]",
		},
		{
			name:          "error without fix has no fixable marker",
			fix:           nil,
			shouldNotHave: "[fixable]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs := []error{
				&validation.Error{
					UnderlyingError: errors.New("test issue"),
					Node:            &yaml.Node{Line: 1, Column: 1},
					Severity:        validation.SeverityWarning,
					Rule:            "test-rule",
					Fix:             tt.fix,
				},
			}

			formatter := format.NewTextFormatter()
			result, err := formatter.Format(errs)
			require.NoError(t, err)

			if tt.shouldHave != "" {
				assert.Contains(t, result, tt.shouldHave, "text output should contain fixable marker")
			}
			if tt.shouldNotHave != "" {
				assert.NotContains(t, result, tt.shouldNotHave, "text output should not contain fixable marker")
			}
		})
	}
}

func TestTextFormatter_NonValidationError(t *testing.T) {
	t.Parallel()

	formatter := format.NewTextFormatter()
	result, err := formatter.Format([]error{
		errors.New("something went wrong internally"),
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "internal", "should show 'internal' rule for non-validation errors")
	assert.Contains(t, result, "something went wrong internally", "should show error message")
	assert.Contains(t, result, "error", "should show error severity")
	assert.Contains(t, result, "1 errors", "summary should count as error")
}

func TestTextFormatter_DocumentLocation(t *testing.T) {
	t.Parallel()

	formatter := format.NewTextFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError:  errors.New("ref issue"),
			Node:             &yaml.Node{Line: 5, Column: 3},
			Severity:         validation.SeverityWarning,
			Rule:             "ref-rule",
			DocumentLocation: "/path/to/other.yaml",
		},
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "(document: /path/to/other.yaml)", "should include document location")
}

func TestJSONFormatter_NonValidationError(t *testing.T) {
	t.Parallel()

	formatter := format.NewJSONFormatter()
	result, err := formatter.Format([]error{
		errors.New("internal failure"),
	})
	require.NoError(t, err, "Format should not fail")

	var output struct {
		Results []struct {
			Rule     string `json:"rule"`
			Category string `json:"category"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"results"`
		Summary struct {
			Errors int `json:"errors"`
			Total  int `json:"total"`
		} `json:"summary"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &output), "should be valid JSON")
	require.Len(t, output.Results, 1, "should have one result")
	assert.Equal(t, "internal", output.Results[0].Rule, "should use 'internal' rule")
	assert.Equal(t, "internal", output.Results[0].Category, "should use 'internal' category")
	assert.Equal(t, "error", output.Results[0].Severity, "should use error severity")
	assert.Equal(t, "internal failure", output.Results[0].Message, "should have error message")
	assert.Equal(t, 1, output.Summary.Errors, "should count as error")
	assert.Equal(t, 1, output.Summary.Total, "should count in total")
}

func TestJSONFormatter_DocumentLocation(t *testing.T) {
	t.Parallel()

	formatter := format.NewJSONFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError:  errors.New("ref issue"),
			Node:             &yaml.Node{Line: 5, Column: 3},
			Severity:         validation.SeverityWarning,
			Rule:             "ref-rule",
			DocumentLocation: "/path/to/other.yaml",
		},
	})
	require.NoError(t, err, "Format should not fail")

	var output struct {
		Results []struct {
			Document string `json:"document"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &output), "should be valid JSON")
	require.Len(t, output.Results, 1, "should have one result")
	assert.Equal(t, "/path/to/other.yaml", output.Results[0].Document, "should include document location")
}

func TestJSONFormatter_HintSeverity(t *testing.T) {
	t.Parallel()

	formatter := format.NewJSONFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError: errors.New("hint message"),
			Node:            &yaml.Node{Line: 1, Column: 1},
			Severity:        validation.SeverityHint,
			Rule:            "hint-rule",
		},
	})
	require.NoError(t, err, "Format should not fail")

	var output struct {
		Summary struct {
			Hints int `json:"hints"`
			Total int `json:"total"`
		} `json:"summary"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &output), "should be valid JSON")
	assert.Equal(t, 1, output.Summary.Hints, "should count hint")
	assert.Equal(t, 1, output.Summary.Total, "should count in total")
}

func TestSummaryFormatter_Empty(t *testing.T) {
	t.Parallel()

	formatter := format.NewSummaryFormatter()
	result, err := formatter.Format([]error{})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "0 problems", "should show zero problems")
	assert.Contains(t, result, "Rule", "should contain table header")
}

func TestSummaryFormatter_SingleRule(t *testing.T) {
	t.Parallel()

	formatter := format.NewSummaryFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError: errors.New("error 1"),
			Node:            &yaml.Node{Line: 1, Column: 1},
			Severity:        validation.SeverityError,
			Rule:            "owasp-rate-limit",
		},
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "owasp-rate-limit", "should show rule name")
	assert.Contains(t, result, "owasp", "should extract category from rule prefix")
	assert.Contains(t, result, "1 problems", "should show problem count")
	assert.Contains(t, result, "1 errors", "should count errors")
}

func TestSummaryFormatter_MultipleSeverities(t *testing.T) {
	t.Parallel()

	formatter := format.NewSummaryFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError: errors.New("err"),
			Node:            &yaml.Node{Line: 1, Column: 1},
			Severity:        validation.SeverityError,
			Rule:            "err-rule",
		},
		&validation.Error{
			UnderlyingError: errors.New("warn"),
			Node:            &yaml.Node{Line: 2, Column: 1},
			Severity:        validation.SeverityWarning,
			Rule:            "warn-rule",
		},
		&validation.Error{
			UnderlyingError: errors.New("hint"),
			Node:            &yaml.Node{Line: 3, Column: 1},
			Severity:        validation.SeverityHint,
			Rule:            "hint-rule",
		},
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "3 problems", "should count all problems")
	assert.Contains(t, result, "1 errors", "should count errors")
	assert.Contains(t, result, "1 warnings", "should count warnings")
	assert.Contains(t, result, "1 hints", "should count hints")
	assert.Contains(t, result, "3 rules", "should count rules")
}

func TestSummaryFormatter_AggregatesSameRule(t *testing.T) {
	t.Parallel()

	formatter := format.NewSummaryFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError: errors.New("first"),
			Node:            &yaml.Node{Line: 1, Column: 1},
			Severity:        validation.SeverityWarning,
			Rule:            "oas-description",
		},
		&validation.Error{
			UnderlyingError: errors.New("second"),
			Node:            &yaml.Node{Line: 2, Column: 1},
			Severity:        validation.SeverityWarning,
			Rule:            "oas-description",
		},
		&validation.Error{
			UnderlyingError: errors.New("other"),
			Node:            &yaml.Node{Line: 3, Column: 1},
			Severity:        validation.SeverityWarning,
			Rule:            "oas-other",
		},
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "2 rules", "should count distinct rules")

	// The rule with count 2 should appear first (sorted by count descending)
	descIdx := strings.Index(result, "oas-description")
	otherIdx := strings.Index(result, "oas-other")
	assert.Greater(t, otherIdx, descIdx, "higher-count rule should appear before lower-count rule")
}

func TestSummaryFormatter_NonValidationError(t *testing.T) {
	t.Parallel()

	formatter := format.NewSummaryFormatter()
	result, err := formatter.Format([]error{
		errors.New("plain error"),
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "internal", "should bucket non-validation errors as 'internal'")
	assert.Contains(t, result, "1 errors", "should count as error")
}

func TestSummaryFormatter_RuleWithoutCategory(t *testing.T) {
	t.Parallel()

	formatter := format.NewSummaryFormatter()
	result, err := formatter.Format([]error{
		&validation.Error{
			UnderlyingError: errors.New("issue"),
			Node:            &yaml.Node{Line: 1, Column: 1},
			Severity:        validation.SeverityWarning,
			Rule:            "norule",
		},
	})
	require.NoError(t, err, "Format should not fail")
	assert.Contains(t, result, "unknown", "should use 'unknown' category for rules without a dash")
}

func TestJSONFormatter_FixMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fix            validation.Fix
		expectFix      bool
		expectInteract bool
	}{
		{
			name:           "non-interactive fix",
			fix:            &testFix{description: "trim slash", interactive: false},
			expectFix:      true,
			expectInteract: false,
		},
		{
			name:           "interactive fix",
			fix:            &testFix{description: "add description", interactive: true},
			expectFix:      true,
			expectInteract: true,
		},
		{
			name:      "no fix",
			fix:       nil,
			expectFix: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs := []error{
				&validation.Error{
					UnderlyingError: errors.New("test issue"),
					Node:            &yaml.Node{Line: 1, Column: 1},
					Severity:        validation.SeverityWarning,
					Rule:            "test-rule",
					Fix:             tt.fix,
				},
			}

			formatter := format.NewJSONFormatter()
			result, err := formatter.Format(errs)
			require.NoError(t, err)

			var output struct {
				Results []struct {
					Fix *struct {
						Description string `json:"description"`
						Interactive bool   `json:"interactive,omitempty"`
					} `json:"fix,omitempty"`
				} `json:"results"`
			}
			require.NoError(t, json.Unmarshal([]byte(result), &output), "should be valid JSON")
			require.Len(t, output.Results, 1, "should have one result")

			if tt.expectFix {
				require.NotNil(t, output.Results[0].Fix, "should have fix metadata")
				assert.Equal(t, tt.fix.Description(), output.Results[0].Fix.Description, "should have correct description")
				assert.Equal(t, tt.expectInteract, output.Results[0].Fix.Interactive, "should have correct interactive flag")
			} else {
				assert.Nil(t, output.Results[0].Fix, "should not have fix metadata")
			}
		})
	}
}
