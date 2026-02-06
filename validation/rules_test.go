package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleInfoForID_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ruleID     string
		expectOK   bool
		expectInfo RuleInfo
	}{
		{
			name:     "known rule returns info",
			ruleID:   RuleValidationRequiredField,
			expectOK: true,
			expectInfo: RuleInfo{
				Summary:     "Missing required field.",
				Description: "Required fields must be present in the document. Missing required fields cause validation to fail.",
				HowToFix:    "Provide the required field in the document.",
			},
		},
		{
			name:     "another known rule returns info",
			ruleID:   RuleValidationCircularReference,
			expectOK: true,
			expectInfo: RuleInfo{
				Summary:     "Circular reference.",
				Description: "Schemas must not contain circular references that cannot be resolved. Unresolvable cycles can break validation and tooling.",
				HowToFix:    "Refactor schemas to break the reference cycle.",
			},
		},
		{
			name:       "unknown rule returns empty info",
			ruleID:     "unknown-rule-id",
			expectOK:   false,
			expectInfo: RuleInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info, ok := RuleInfoForID(tt.ruleID)
			assert.Equal(t, tt.expectOK, ok, "ok should match expected")
			assert.Equal(t, tt.expectInfo, info, "info should match expected")
		})
	}
}

func TestRuleSummary_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ruleID   string
		expected string
	}{
		{
			name:     "known rule returns summary",
			ruleID:   RuleValidationTypeMismatch,
			expected: "Type mismatch.",
		},
		{
			name:     "unknown rule returns empty string",
			ruleID:   "unknown-rule",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := RuleSummary(tt.ruleID)
			assert.Equal(t, tt.expected, result, "summary should match expected")
		})
	}
}

func TestRuleDescription_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ruleID   string
		expected string
	}{
		{
			name:     "known rule returns description",
			ruleID:   RuleValidationDuplicateKey,
			expected: "Duplicate keys are not allowed in objects. Remove duplicates to avoid parsing ambiguity.",
		},
		{
			name:     "unknown rule returns empty string",
			ruleID:   "unknown-rule",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := RuleDescription(tt.ruleID)
			assert.Equal(t, tt.expected, result, "description should match expected")
		})
	}
}

func TestRuleHowToFix_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ruleID   string
		expected string
	}{
		{
			name:     "known rule returns how to fix",
			ruleID:   RuleValidationInvalidReference,
			expected: "Fix the $ref target or define the referenced component.",
		},
		{
			name:     "unknown rule returns empty string",
			ruleID:   "unknown-rule",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := RuleHowToFix(tt.ruleID)
			assert.Equal(t, tt.expected, result, "how to fix should match expected")
		})
	}
}
