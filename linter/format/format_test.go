package format_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter/format"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
