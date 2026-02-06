package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRuleTypeScript_TruthyRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "require-operation-summary",
		Description: "All operations must have a summary",
		Severity:    "warn",
		Given:       []string{"$.paths[*][*]"},
		Then:        []RuleCheck{{Field: "summary", Function: "truthy"}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "truthy-rule.ts", source)
}

func TestGenerateRuleTypeScript_PatternRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "version-semver",
		Description: "Version must be semver format",
		Severity:    "error",
		Given:       []string{"$.info.version"},
		Then: []RuleCheck{{
			Function:        "pattern",
			FunctionOptions: map[string]any{"match": `^[0-9]+\.[0-9]+\.[0-9]+`},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "pattern-rule.ts", source)
}

func TestGenerateRuleTypeScript_CasingRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "path-casing",
		Description: "Path segments must use kebab-case",
		Severity:    "error",
		Given:       []string{"$.paths[*]~"},
		Then: []RuleCheck{{
			Function:        "casing",
			FunctionOptions: map[string]any{"type": "kebab-case"},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "casing-rule.ts", source)
}

func TestGenerateRuleTypeScript_EnumerationRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "protocol-check",
		Description: "Must use approved protocols",
		Severity:    "error",
		Given:       []string{"$.servers[*].url"},
		Then: []RuleCheck{{
			Function:        "enumeration",
			FunctionOptions: map[string]any{"values": []any{"https://api.example.com", "https://staging.example.com"}},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "enumeration-rule.ts", source)
}

func TestGenerateRuleTypeScript_LengthRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "description-length",
		Description: "Description must be between 10 and 200 characters",
		Severity:    "warn",
		Given:       []string{"$.paths[*][*]"},
		Then: []RuleCheck{{
			Field:           "description",
			Function:        "length",
			FunctionOptions: map[string]any{"min": 10, "max": 200},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "length-rule.ts", source)
}

func TestGenerateRuleTypeScript_XorRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "xor-example",
		Description: "Must have exactly one of value or externalValue",
		Severity:    "error",
		Given:       []string{"$.components.examples[*]"},
		Then: []RuleCheck{{
			Function:        "xor",
			FunctionOptions: map[string]any{"properties": []any{"value", "externalValue"}},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "xor-rule.ts", source)
}

func TestGenerateRuleTypeScript_OrRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "or-example",
		Description: "Must have at least one of description or summary",
		Severity:    "warn",
		Given:       []string{"$.paths[*][*]"},
		Then: []RuleCheck{{
			Function:        "or",
			FunctionOptions: map[string]any{"properties": []any{"description", "summary"}},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "or-rule.ts", source)
}

func TestGenerateRuleTypeScript_UnsupportedJSONPath(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "deep-path-rule",
		Description: "Rule with unsupported path",
		Severity:    "warn",
		Given:       []string{"$.x-custom.deeply.nested[*].something"},
		Then:        []RuleCheck{{Function: "truthy"}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	require.Len(t, warnings, 1, "should have one warning")
	assert.Contains(t, warnings[0].Message, "unsupported JSONPath", "warning about unsupported path")
	assert.Equal(t, "generate", warnings[0].Phase, "warning phase")
	assertGoldenFile(t, "unsupported-jsonpath.ts", source)
}

func TestGenerateRuleTypeScript_CustomFunction(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "custom-fn-rule",
		Description: "Rule using custom function",
		Severity:    "warn",
		Given:       []string{"$.info"},
		Then:        []RuleCheck{{Function: "myCompanyValidator"}},
	}

	source, _ := GenerateRuleTypeScript(rule, "custom-")
	assertGoldenFile(t, "custom-function.ts", source)
}

func TestGenerateRuleTypeScript_SchemaFunction(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "schema-rule",
		Description: "Schema validation rule",
		Severity:    "error",
		Given:       []string{"$.info"},
		Then:        []RuleCheck{{Function: "schema"}},
	}

	source, _ := GenerateRuleTypeScript(rule, "custom-")
	assertGoldenFile(t, "schema-function.ts", source)
}

func TestGenerateRuleTypeScript_WithFormats(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "oas3-only",
		Description: "OAS3 only rule",
		Severity:    "warn",
		Formats:     []string{"oas3"},
		Given:       []string{"$.info"},
		Then:        []RuleCheck{{Field: "description", Function: "truthy"}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "with-formats.ts", source)
}

func TestGenerateRuleTypeScript_MultipleThenChecks(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "multi-check",
		Description: "Multiple checks on operations",
		Severity:    "error",
		Given:       []string{"$.paths[*][*]"},
		Then: []RuleCheck{
			{Field: "summary", Function: "truthy"},
			{Field: "description", Function: "truthy"},
		},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "multiple-then.ts", source)
}

func TestGenerateRuleTypeScript_MessageTemplate(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "message-template",
		Description: "Rule with message template",
		Severity:    "warn",
		Message:     "{{property}} must be present",
		Given:       []string{"$.info"},
		Then:        []RuleCheck{{Field: "description", Function: "truthy"}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "message-template.ts", source)
}

func TestGenerateRuleTypeScript_UnknownFieldAccess(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "extension-check",
		Description: "Check x-gateway extension",
		Severity:    "warn",
		Given:       []string{"$.paths[*][*]"},
		Then:        []RuleCheck{{Field: "x-gateway", Function: "truthy"}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "unknown-field.ts", source)
}

func TestGenerateRuleTypeScript_AlphabeticalRule(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "tags-alphabetical",
		Description: "Tags must be in alphabetical order",
		Severity:    "warn",
		Given:       []string{"$.tags[*]"},
		Then: []RuleCheck{{
			Function:        "alphabetical",
			FunctionOptions: map[string]any{"keyedBy": "name"},
		}},
	}

	source, warnings := GenerateRuleTypeScript(rule, "custom-")
	assert.Empty(t, warnings, "no warnings expected")
	assertGoldenFile(t, "alphabetical-rule.ts", source)
}

func TestToClassName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "kebab-case", input: "require-operation-summary", expected: "RequireOperationSummary"},
		{name: "with dots", input: "oas3-server-not-example.com", expected: "Oas3ServerNotExampleCom"},
		{name: "with underscores", input: "my_custom_rule", expected: "MyCustomRule"},
		{name: "already pascal", input: "MyRule", expected: "MyRule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, toClassName(tt.input), "class name")
		})
	}
}

func TestEscapeTS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "single quotes", input: "it's a test", expected: "it\\'s a test"},
		{name: "backslashes", input: `path\to\file`, expected: `path\\to\\file`},
		{name: "newlines", input: "line1\nline2", expected: "line1\\nline2"},
		{name: "clean string", input: "no special chars", expected: "no special chars"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, escapeTS(tt.input), "escaped string")
		})
	}
}

func TestCasingRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		caseType string
		expected string
	}{
		{name: "camelCase", caseType: "camelCase", expected: `/^[a-z][a-zA-Z0-9]*$/`},
		{name: "PascalCase", caseType: "PascalCase", expected: `/^[A-Z][a-zA-Z0-9]*$/`},
		{name: "kebab-case", caseType: "kebab-case", expected: `/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/`},
		{name: "snake_case", caseType: "snake_case", expected: `/^[a-z][a-z0-9]*(_[a-z0-9]+)*$/`},
		{name: "unknown fallback", caseType: "unknown", expected: `/^.+$/`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, casingRegex(tt.caseType), "regex")
		})
	}
}

func TestFormatsToVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		formats  []string
		expected []string
	}{
		{name: "oas2", formats: []string{"oas2"}, expected: []string{"2.0"}},
		{name: "oas3", formats: []string{"oas3"}, expected: []string{"3.0", "3.1"}},
		{name: "oas3.0", formats: []string{"oas3.0"}, expected: []string{"3.0"}},
		{name: "oas3.1", formats: []string{"oas3.1"}, expected: []string{"3.1"}},
		{name: "multiple", formats: []string{"oas2", "oas3"}, expected: []string{"2.0", "3.0", "3.1"}},
		{name: "dedup", formats: []string{"oas3", "oas3.0"}, expected: []string{"3.0", "3.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatsToVersions(tt.formats)
			assert.Equal(t, tt.expected, result, "versions")
		})
	}
}

func TestSummaryFromDesc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "short sentence", input: "All operations must have a summary.", expected: "All operations must have a summary."},
		{name: "long text", input: strings.Repeat("a", 100), expected: strings.Repeat("a", 77) + "..."},
		{name: "empty", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, summaryFromDesc(tt.input), "summary")
		})
	}
}

// assertGoldenFile compares the actual output against a golden file in testdata/golden/.
func assertGoldenFile(t *testing.T, filename, actual string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", "golden", filename)
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "should read golden file %s", filename)

	assert.Equal(t, string(expected), actual, "output should match golden file %s", filename)
}
