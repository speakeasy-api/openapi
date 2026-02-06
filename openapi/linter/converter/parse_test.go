package converter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_SpectralBasic(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-basic.yaml")
	require.NoError(t, err, "should parse spectral basic config")
	require.NotNil(t, ir, "IR should not be nil")

	// Check extends
	require.Len(t, ir.Extends, 1, "should have one extends entry")
	assert.Equal(t, "spectral:oas", ir.Extends[0].Name, "extends name")
	assert.Equal(t, "recommended", ir.Extends[0].Modifier, "extends modifier")

	// Check severity overrides
	overrides := findRulesByOverride(ir.Rules)
	assert.Len(t, overrides, 3, "should have 3 severity overrides")

	opTags := findRule(ir.Rules, "operation-tags")
	require.NotNil(t, opTags, "should have operation-tags rule")
	assert.True(t, opTags.IsOverride(), "operation-tags should be an override")
	assert.Equal(t, "error", opTags.Severity, "operation-tags severity")

	opId := findRule(ir.Rules, "operation-operationId")
	require.NotNil(t, opId, "should have operation-operationId rule")
	assert.Equal(t, "warn", opId.Severity, "operation-operationId severity")

	opDesc := findRule(ir.Rules, "operation-description")
	require.NotNil(t, opDesc, "should have operation-description rule")
	assert.Equal(t, "off", opDesc.Severity, "operation-description severity")
	assert.True(t, opDesc.IsDisabled(), "operation-description should be disabled")

	// Check full custom rules
	headerCheck := findRule(ir.Rules, "custom-header-check")
	require.NotNil(t, headerCheck, "should have custom-header-check rule")
	assert.False(t, headerCheck.IsOverride(), "custom-header-check should not be override")
	assert.Equal(t, "All responses must include X-Request-ID header", headerCheck.Description, "description")
	assert.Contains(t, headerCheck.Message, "X-Request-ID", "message template")
	assert.Equal(t, "warn", headerCheck.Severity, "severity")
	require.Len(t, headerCheck.Given, 1, "should have one given path")
	assert.Equal(t, "$.paths[*][*].responses[*]", headerCheck.Given[0], "given path")
	require.Len(t, headerCheck.Then, 1, "should have one then check")
	assert.Equal(t, "headers.X-Request-ID", headerCheck.Then[0].Field, "then field")
	assert.Equal(t, "truthy", headerCheck.Then[0].Function, "then function")

	pathCasing := findRule(ir.Rules, "custom-path-casing")
	require.NotNil(t, pathCasing, "should have custom-path-casing rule")
	assert.Equal(t, "error", pathCasing.Severity, "severity")
	assert.Equal(t, "$.paths[*]~", pathCasing.Given[0], "given with ~ operator")
	assert.Equal(t, "casing", pathCasing.Then[0].Function, "casing function")
	assert.Equal(t, "kebab-case", pathCasing.Then[0].FunctionOptions["type"], "casing type option")
}

func TestParse_SpectralEdgeCases(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-edge-cases.yaml")
	require.NoError(t, err, "should parse edge cases config")

	// Extends as simple string
	require.Len(t, ir.Extends, 1, "should have one extends entry")
	assert.Equal(t, "spectral:oas", ir.Extends[0].Name, "extends name")
	assert.Empty(t, ir.Extends[0].Modifier, "no modifier for string extends")

	// Numeric severity (0 = error)
	opTags := findRule(ir.Rules, "operation-tags")
	require.NotNil(t, opTags, "should have operation-tags")
	assert.Equal(t, "error", opTags.Severity, "numeric 0 should be error")

	// Boolean true = use default (empty severity, still an override)
	opId := findRule(ir.Rules, "operation-operationId")
	require.NotNil(t, opId, "should have operation-operationId")
	assert.True(t, opId.IsOverride(), "true should be an override")
	assert.Empty(t, opId.Severity, "true should have empty severity (use default)")

	// Boolean false = off
	opDesc := findRule(ir.Rules, "operation-description")
	require.NotNil(t, opDesc, "should have operation-description")
	assert.Equal(t, "off", opDesc.Severity, "false should be off")

	// Singleton given (string not array)
	singleton := findRule(ir.Rules, "singleton-given-then")
	require.NotNil(t, singleton, "should have singleton-given-then")
	require.Len(t, singleton.Given, 1, "should have 1 given path from string")
	assert.Equal(t, "$.info", singleton.Given[0], "given path")
	require.Len(t, singleton.Then, 1, "should have 1 then check from object")
	assert.Equal(t, "description", singleton.Then[0].Field, "then field")

	// Multiple then entries
	multiThen := findRule(ir.Rules, "multi-then")
	require.NotNil(t, multiThen, "should have multi-then")
	require.Len(t, multiThen.Then, 2, "should have 2 then checks")
	assert.Equal(t, "summary", multiThen.Then[0].Field, "first check field")
	assert.Equal(t, "description", multiThen.Then[1].Field, "second check field")

	// Pattern with functionOptions
	patternRule := findRule(ir.Rules, "pattern-rule")
	require.NotNil(t, patternRule, "should have pattern-rule")
	match, notMatch := PatternOptions(patternRule.Then[0])
	assert.Contains(t, match, "^[0-9]+", "should have match pattern")
	assert.Empty(t, notMatch, "should have no notMatch")

	// Custom/unknown function preserved as-is
	customFn := findRule(ir.Rules, "custom-function-rule")
	require.NotNil(t, customFn, "should have custom-function-rule")
	assert.Equal(t, "myCompanyValidator", customFn.Then[0].Function, "custom function preserved")
	assert.Equal(t, true, customFn.Then[0].FunctionOptions["checkLicense"], "custom function options preserved")

	// Resolved false
	unresolved := findRule(ir.Rules, "unresolved-rule")
	require.NotNil(t, unresolved, "should have unresolved-rule")
	require.NotNil(t, unresolved.Resolved, "resolved should not be nil")
	assert.False(t, *unresolved.Resolved, "resolved should be false")

	// Formats field
	oas3Only := findRule(ir.Rules, "oas3-only-rule")
	require.NotNil(t, oas3Only, "should have oas3-only-rule")
	require.Len(t, oas3Only.Formats, 1, "should have 1 format")
	assert.Equal(t, "oas3", oas3Only.Formats[0], "format")

	// enabled: false (Vacuum extension) -> off severity
	disabledVacuum := findRule(ir.Rules, "disabled-vacuum-rule")
	require.NotNil(t, disabledVacuum, "should have disabled-vacuum-rule")
	assert.Equal(t, "off", disabledVacuum.Severity, "enabled:false should set severity to off")
	assert.True(t, disabledVacuum.IsDisabled(), "should be disabled")
}

func TestParse_LegacyKombo(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/legacy-kombo.yaml")
	require.NoError(t, err, "should parse legacy Kombo config")
	require.NotNil(t, ir, "IR should not be nil")

	// Check extends from nested rulesets references
	require.Len(t, ir.Extends, 1, "should have one extends entry")
	assert.Equal(t, "speakeasy-recommended", ir.Extends[0].Name, "extends from nested ruleset")
	assert.Empty(t, ir.Extends[0].Modifier, "no modifier for legacy extends")

	// Check rule
	rule := findRule(ir.Rules, "require-endpoint-renamings")
	require.NotNil(t, rule, "should have require-endpoint-renamings rule")
	assert.Equal(t, "error", rule.Severity, "severity")
	assert.Equal(t, "kombo", rule.Source, "source ruleset")
	assert.Contains(t, rule.Description, "x-speakeasy-group", "description")

	// Check given path with filter
	require.Len(t, rule.Given, 1, "should have one given path")
	assert.Contains(t, rule.Given[0], "$.paths", "given path")

	// Check then - two truthy checks
	require.Len(t, rule.Then, 2, "should have 2 then checks")
	assert.Equal(t, "x-speakeasy-group", rule.Then[0].Field, "first field")
	assert.Equal(t, "truthy", rule.Then[0].Function, "first function")
	assert.Equal(t, "x-speakeasy-name-override", rule.Then[1].Field, "second field")
	assert.Equal(t, "truthy", rule.Then[1].Function, "second function")

	// Should warn about defaultRuleset
	hasDefaultRulesetWarning := false
	for _, w := range ir.Warnings {
		if w.Phase == "parse" && strings.Contains(w.Message, "defaultRuleset") {
			hasDefaultRulesetWarning = true
			break
		}
	}
	assert.True(t, hasDefaultRulesetWarning, "should warn about defaultRuleset")
}

func TestParse_VacuumBasic(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/vacuum-basic.yaml")
	require.NoError(t, err, "should parse vacuum config")
	require.NotNil(t, ir, "IR should not be nil")

	// Check extends
	require.Len(t, ir.Extends, 1, "should have one extends entry")
	assert.Equal(t, "spectral:oas", ir.Extends[0].Name, "extends name")
	assert.Equal(t, "all", ir.Extends[0].Modifier, "extends modifier")

	// Check overrides
	opId := findRule(ir.Rules, "operation-operationId")
	require.NotNil(t, opId, "should have operation-operationId")
	assert.Equal(t, "error", opId.Severity, "severity override")

	tagDesc := findRule(ir.Rules, "tag-description")
	require.NotNil(t, tagDesc, "should have tag-description")
	assert.Equal(t, "warn", tagDesc.Severity, "severity override")

	// Check custom rule
	serverUrl := findRule(ir.Rules, "check-server-url")
	require.NotNil(t, serverUrl, "should have check-server-url")
	assert.Equal(t, "error", serverUrl.Severity, "severity")
	assert.Equal(t, "$.servers[*].url", serverUrl.Given[0], "given path")
	assert.Equal(t, "pattern", serverUrl.Then[0].Function, "function")
}

func TestParse_EmptyConfig(t *testing.T) {
	t.Parallel()

	ir, err := Parse(strings.NewReader(""))
	require.NoError(t, err, "should parse empty config")
	require.NotNil(t, ir, "IR should not be nil")
	assert.Empty(t, ir.Rules, "no rules in empty config")
	assert.Empty(t, ir.Extends, "no extends in empty config")
}

func TestParse_InvalidYAML(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader("{{invalid yaml"))
	require.Error(t, err, "should fail on invalid YAML")
}

func TestParse_IRHelpers(t *testing.T) {
	t.Parallel()

	t.Run("PatternOptions", func(t *testing.T) {
		t.Parallel()
		rc := RuleCheck{
			FunctionOptions: map[string]any{
				"match":    "^test",
				"notMatch": "bad$",
			},
		}
		match, notMatch := PatternOptions(rc)
		assert.Equal(t, "^test", match, "match")
		assert.Equal(t, "bad$", notMatch, "notMatch")
	})

	t.Run("EnumerationOptions", func(t *testing.T) {
		t.Parallel()
		rc := RuleCheck{
			FunctionOptions: map[string]any{
				"values": []any{"a", "b", "c"},
			},
		}
		values := EnumerationOptions(rc)
		assert.Equal(t, []string{"a", "b", "c"}, values, "enumeration values")
	})

	t.Run("LengthOptions", func(t *testing.T) {
		t.Parallel()
		rc := RuleCheck{
			FunctionOptions: map[string]any{
				"min": 1,
				"max": 100,
			},
		}
		minVal, maxVal := LengthOptions(rc)
		require.NotNil(t, minVal, "min should not be nil")
		require.NotNil(t, maxVal, "max should not be nil")
		assert.Equal(t, 1, *minVal, "min value")
		assert.Equal(t, 100, *maxVal, "max value")
	})

	t.Run("CasingOptions", func(t *testing.T) {
		t.Parallel()
		rc := RuleCheck{
			FunctionOptions: map[string]any{
				"type": "camelCase",
			},
		}
		assert.Equal(t, "camelCase", CasingOptions(rc), "casing type")
	})

	t.Run("PropertyOptions", func(t *testing.T) {
		t.Parallel()
		rc := RuleCheck{
			FunctionOptions: map[string]any{
				"properties": []any{"a", "b"},
			},
		}
		props := PropertyOptions(rc)
		assert.Equal(t, []string{"a", "b"}, props, "properties")
	})

	t.Run("nil FunctionOptions", func(t *testing.T) {
		t.Parallel()
		rc := RuleCheck{}
		match, notMatch := PatternOptions(rc)
		assert.Empty(t, match, "empty match")
		assert.Empty(t, notMatch, "empty notMatch")
		assert.Nil(t, EnumerationOptions(rc), "nil enumeration")
		minVal, maxVal := LengthOptions(rc)
		assert.Nil(t, minVal, "nil min")
		assert.Nil(t, maxVal, "nil max")
		assert.Empty(t, CasingOptions(rc), "empty casing")
		assert.Nil(t, PropertyOptions(rc), "nil properties")
	})
}

func TestParse_IRPreservesRawValues(t *testing.T) {
	t.Parallel()

	// Verify that Parse produces raw source values — no native-format interpretation.
	// Rule IDs stay as Spectral names, extends stay as source values.

	ir, err := ParseFile("testdata/spectral-basic.yaml")
	require.NoError(t, err, "should parse config")

	// Extends should be raw Spectral values, NOT mapped to native ("recommended"/"all")
	require.Len(t, ir.Extends, 1, "should have one extends entry")
	assert.Equal(t, "spectral:oas", ir.Extends[0].Name, "extends name should be raw Spectral value")
	assert.Equal(t, "recommended", ir.Extends[0].Modifier, "modifier should be preserved as-is")

	// Rule IDs should be original Spectral names, NOT native IDs
	opTagsRule := findRule(ir.Rules, "operation-tags")
	require.NotNil(t, opTagsRule, "should have operation-tags (not style-operation-tags)")
	assert.Equal(t, "error", opTagsRule.Severity, "severity should be normalized string")

	opDescRule := findRule(ir.Rules, "operation-description")
	require.NotNil(t, opDescRule, "should have operation-description (not style-operation-description)")
	assert.Equal(t, "off", opDescRule.Severity, "off severity preserved")

	// Custom rules should have raw Given/Then, not interpreted
	headerRule := findRule(ir.Rules, "custom-header-check")
	require.NotNil(t, headerRule, "should have custom-header-check")
	assert.Equal(t, []string{"$.paths[*][*].responses[*]"}, headerRule.Given, "Given should be raw JSONPath")
	require.Len(t, headerRule.Then, 1, "should have one then check")
	assert.Equal(t, "truthy", headerRule.Then[0].Function, "function preserved as-is")
	assert.Equal(t, "headers.X-Request-ID", headerRule.Then[0].Field, "field preserved as-is")
}

func TestParse_AdidasFixture(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-adidas.yaml")
	require.NoError(t, err, "should parse Adidas fixture")

	// Should have 30+ rules total
	assert.GreaterOrEqual(t, len(ir.Rules), 30, "Adidas fixture should have 30+ rules")

	// Verify a mix of overrides and custom rules
	overrides := findRulesByOverride(ir.Rules)
	assert.GreaterOrEqual(t, len(overrides), 25, "should have many overrides")

	customRules := 0
	for _, r := range ir.Rules {
		if !r.IsOverride() {
			customRules++
		}
	}
	assert.GreaterOrEqual(t, customRules, 5, "should have several custom rules")

	// Verify specific custom rules parsed correctly
	semverRule := findRule(ir.Rules, "adidas-version-semver")
	require.NotNil(t, semverRule, "should have version-semver rule")
	assert.Equal(t, "error", semverRule.Severity)
	require.Len(t, semverRule.Then, 1)
	assert.Equal(t, "pattern", semverRule.Then[0].Function)

	// Verify custom function rule
	namingRule := findRule(ir.Rules, "adidas-custom-naming")
	require.NotNil(t, namingRule, "should have custom-naming rule")
	assert.Equal(t, "adidasNamingConvention", namingRule.Then[0].Function)

	// Verify resolved: false rule
	circularRule := findRule(ir.Rules, "adidas-no-circular-refs")
	require.NotNil(t, circularRule, "should have no-circular-refs rule")
	require.NotNil(t, circularRule.Resolved)
	assert.False(t, *circularRule.Resolved, "resolved should be false")
}

// --- helpers ---

func findRule(rules []Rule, id string) *Rule {
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i]
		}
	}
	return nil
}

func TestParse_LenientPerRule(t *testing.T) {
	t.Parallel()

	// Config with one good rule and one malformed rule (given is a mapping, not string/[]string)
	input := `
extends: spectral:oas
rules:
  good-rule:
    description: A good rule
    given: "$.info"
    then:
      function: truthy
      field: description
  bad-rule:
    description: A bad rule
    given:
      invalid: structure
    then:
      function: truthy
  another-good-rule:
    description: Another good rule
    given: "$.info"
    then:
      function: truthy
      field: contact
`
	ir, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "parse should not fail entirely due to one bad rule")

	// Should have the two good rules
	goodRules := 0
	for _, r := range ir.Rules {
		if !r.IsOverride() {
			goodRules++
		}
	}
	assert.GreaterOrEqual(t, goodRules, 1, "should have at least one good rule parsed")

	// Should have a warning about the bad rule
	hasWarning := false
	for _, w := range ir.Warnings {
		if w.RuleID == "bad-rule" && w.Phase == "parse" {
			hasWarning = true
			break
		}
	}
	assert.True(t, hasWarning, "should have a parse warning for bad-rule")
}

func TestParse_FunctionsDirWarning(t *testing.T) {
	t.Parallel()

	input := `
extends: spectral:oas
functionsDir: ./custom-functions
functions:
  - myValidator
rules:
  my-rule: error
`
	ir, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "should parse successfully")

	// Should have warnings about functionsDir and functions
	hasFuncsDirWarning := false
	hasFuncsWarning := false
	for _, w := range ir.Warnings {
		if strings.Contains(w.Message, "functionsDir") {
			hasFuncsDirWarning = true
		}
		if strings.Contains(w.Message, "functions") && strings.Contains(w.Message, "myValidator") {
			hasFuncsWarning = true
		}
	}
	assert.True(t, hasFuncsDirWarning, "should warn about functionsDir")
	assert.True(t, hasFuncsWarning, "should warn about functions")
}

func TestParse_MessageTemplateExpansion(t *testing.T) {
	t.Parallel()

	rule := Rule{
		ID:          "test-rule",
		Description: "Test description",
		Severity:    "warn",
		Message:     "{{property}} must be present for {{description}}",
		Given:       []string{"$.info"},
		Then:        []RuleCheck{{Field: "contact", Function: "truthy"}},
	}

	source, _ := GenerateRuleTypeScript(rule, "custom-")

	// Should NOT contain ${property} (which would be an undefined JS variable)
	assert.NotContains(t, source, "${property}", "should not contain JS template literal with undefined variable")
	assert.NotContains(t, source, "${description}", "should not contain JS template literal with undefined variable")

	// Should contain the expanded static values
	assert.Contains(t, source, "contact must be present", "should expand {{property}} to the field name")
	assert.Contains(t, source, "Test description", "should expand {{description}} to the rule description")
}

func findRulesByOverride(rules []Rule) []Rule {
	var result []Rule
	for _, r := range rules {
		if r.IsOverride() {
			result = append(result, r)
		}
	}
	return result
}
