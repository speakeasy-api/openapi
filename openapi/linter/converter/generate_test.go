package converter

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerate_SpectralBasic(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-basic.yaml")
	require.NoError(t, err, "should parse config")

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")
	require.NotNil(t, result, "result should not be nil")

	// Extends should be mapped: spectral:oas + recommended -> recommended
	assert.Contains(t, result.Config.Extends, "recommended", "extends should include recommended")

	// Severity overrides should be mapped to native IDs
	opTagsEntry := findRuleEntry(result.Config.Rules, "style-operation-tags")
	require.NotNil(t, opTagsEntry, "should have style-operation-tags entry")
	require.NotNil(t, opTagsEntry.Severity, "should have severity set")
	assert.Equal(t, validation.SeverityError, *opTagsEntry.Severity, "should be error severity")

	opDescEntry := findRuleEntry(result.Config.Rules, "style-operation-description")
	require.NotNil(t, opDescEntry, "should have style-operation-description entry")
	require.NotNil(t, opDescEntry.Disabled, "should have disabled set")
	assert.True(t, *opDescEntry.Disabled, "should be disabled")

	// Custom rules should generate TypeScript
	assert.Contains(t, result.GeneratedRules, "custom-custom-header-check", "should generate header check rule")
	assert.Contains(t, result.GeneratedRules, "custom-custom-path-casing", "should generate path casing rule")

	// Custom rules config should be set
	require.NotNil(t, result.Config.CustomRules, "custom rules config should be set")
	require.Len(t, result.Config.CustomRules.Paths, 1, "should have one custom rules path")
	assert.Equal(t, "rules/*.ts", result.Config.CustomRules.Paths[0], "custom rules glob")
}

func TestGenerate_LegacyKombo(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/legacy-kombo.yaml")
	require.NoError(t, err, "should parse config")

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	// Legacy "speakeasy-recommended" -> extends: recommended
	assert.Contains(t, result.Config.Extends, "recommended", "extends should include recommended")

	// Custom rule should be generated
	assert.Contains(t, result.GeneratedRules, "custom-require-endpoint-renamings", "should generate custom rule")
	source := result.GeneratedRules["custom-require-endpoint-renamings"]
	assert.Contains(t, source, "x-speakeasy-group", "generated source should reference field")

	// Should have error severity override in config
	entry := findRuleEntry(result.Config.Rules, "custom-require-endpoint-renamings")
	require.NotNil(t, entry, "should have rule entry for custom rule")
	require.NotNil(t, entry.Severity, "should have severity")
	assert.Equal(t, validation.SeverityError, *entry.Severity, "should be error severity")
}

func TestGenerate_ConfigIsLoadable(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-basic.yaml")
	require.NoError(t, err, "should parse config")

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	// Marshal to YAML and reload to verify it's valid
	yamlData, err := marshalConfig(result.Config)
	require.NoError(t, err, "should marshal config")

	loaded, err := linter.LoadConfig(bytes.NewReader(yamlData))
	require.NoError(t, err, "generated config should be loadable by native loader")
	require.NotNil(t, loaded, "loaded config should not be nil")
	require.NotNil(t, loaded.CustomRules, "custom_rules should survive round-trip through LoadConfig")
	assert.NotEmpty(t, loaded.CustomRules.Paths, "custom_rules.paths should survive round-trip")
}

func TestGenerate_WriteFiles(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-basic.yaml")
	require.NoError(t, err, "should parse config")

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	// Write to temp directory
	tmpDir := t.TempDir()
	err = result.WriteFiles(tmpDir)
	require.NoError(t, err, "should write files")

	// Check lint.yaml exists and is loadable
	configPath := filepath.Join(tmpDir, "lint.yaml")
	_, err = os.Stat(configPath)
	require.NoError(t, err, "lint.yaml should exist")
	_, err = linter.LoadConfigFromFile(configPath)
	require.NoError(t, err, "lint.yaml should be loadable")

	// Check rules directory and files exist
	rulesDir := filepath.Join(tmpDir, "rules")
	_, err = os.Stat(rulesDir)
	require.NoError(t, err, "rules directory should exist")

	for ruleID := range result.GeneratedRules {
		rulePath := filepath.Join(rulesDir, ruleID+".ts")
		_, err = os.Stat(rulePath)
		require.NoError(t, err, "rule file %s should exist", ruleID)
	}
}

func TestGenerate_MapExtends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		extends         []ExtendsEntry
		expectedExtends []string
		expectWarnings  bool
	}{
		{
			name:            "spectral:oas with recommended",
			extends:         []ExtendsEntry{{Name: "spectral:oas", Modifier: "recommended"}},
			expectedExtends: []string{"recommended"},
		},
		{
			name:            "spectral:oas with all",
			extends:         []ExtendsEntry{{Name: "spectral:oas", Modifier: "all"}},
			expectedExtends: []string{"all"},
		},
		{
			name:            "spectral:oas with empty modifier",
			extends:         []ExtendsEntry{{Name: "spectral:oas"}},
			expectedExtends: []string{"all"},
		},
		{
			name:            "speakeasy-recommended",
			extends:         []ExtendsEntry{{Name: "speakeasy-recommended"}},
			expectedExtends: []string{"recommended"},
		},
		{
			name:            "speakeasy-generation",
			extends:         []ExtendsEntry{{Name: "speakeasy-generation"}},
			expectedExtends: []string{"all"},
		},
		{
			name:            "spectral:oas off",
			extends:         []ExtendsEntry{{Name: "spectral:oas", Modifier: "off"}},
			expectedExtends: nil,
			expectWarnings:  true,
		},
		{
			name:            "unknown extends",
			extends:         []ExtendsEntry{{Name: "my-custom-extends"}},
			expectedExtends: []string{"my-custom-extends"},
			expectWarnings:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var warnings []Warning
			result := mapExtends(tt.extends, &warnings)
			assert.Equal(t, tt.expectedExtends, result, "mapped extends")
			if tt.expectWarnings {
				assert.NotEmpty(t, warnings, "should have warnings")
			} else {
				assert.Empty(t, warnings, "should have no warnings")
			}
		})
	}
}

func TestGenerate_UnmappedOverrideRule(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Rules: []Rule{
			{ID: "some-unknown-spectral-rule", Severity: "error"},
		},
	}

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	// Unmapped rule should be disabled with warning
	entry := findRuleEntry(result.Config.Rules, "unmapped-some-unknown-spectral-rule")
	require.NotNil(t, entry, "should have unmapped rule entry")
	require.NotNil(t, entry.Disabled, "should be disabled")
	assert.True(t, *entry.Disabled, "should be disabled")

	hasWarning := false
	for _, w := range result.Warnings {
		if w.RuleID == "some-unknown-spectral-rule" && w.Phase == "generate" {
			hasWarning = true
			break
		}
	}
	assert.True(t, hasWarning, "should have warning about unmapped rule")
}

func TestGenerate_DeduplicateRules(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Rules: []Rule{
			{ID: "operation-tags", Severity: "warn"},
			{ID: "operation-tags", Severity: "error"}, // second occurrence wins
		},
	}

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	// Only one entry for the native rule
	entries := findAllRuleEntries(result.Config.Rules, "style-operation-tags")
	require.Len(t, entries, 1, "should have exactly one entry after dedup")
	require.NotNil(t, entries[0].Severity, "should have severity")
	assert.Equal(t, validation.SeverityError, *entries[0].Severity, "last occurrence (error) wins")

	// Should have dedup warning
	hasDedup := false
	for _, w := range result.Warnings {
		if w.Phase == "generate" && w.RuleID == "operation-tags" {
			hasDedup = true
		}
	}
	assert.True(t, hasDedup, "should have dedup warning")
}

func TestGenerate_ResolvedFalseWarning(t *testing.T) {
	t.Parallel()

	resolved := false
	ir := &IntermediateConfig{
		Rules: []Rule{
			{
				ID:       "test-rule",
				Severity: "warn",
				Resolved: &resolved,
				Given:    []string{"$"},
				Then:     []RuleCheck{{Function: "truthy"}},
			},
		},
	}

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	hasResolvedWarning := false
	for _, w := range result.Warnings {
		if w.RuleID == "test-rule" && w.Phase == "generate" {
			hasResolvedWarning = true
		}
	}
	assert.True(t, hasResolvedWarning, "should warn about resolved: false")
}

func TestGenerate_CustomRulesDir(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Rules: []Rule{
			{
				ID:       "test-rule",
				Severity: "warn",
				Given:    []string{"$.info"},
				Then:     []RuleCheck{{Field: "description", Function: "truthy"}},
			},
		},
	}

	result, err := Generate(ir, WithRulesDir("./my-rules"), WithRulePrefix("my-"))
	require.NoError(t, err, "should generate")

	assert.Contains(t, result.GeneratedRules, "my-test-rule", "should use custom prefix")
	assert.Equal(t, "my-rules/*.ts", result.Config.CustomRules.Paths[0], "should use custom rules dir")
}

func TestGenerate_AdidasEndToEnd(t *testing.T) {
	t.Parallel()

	ir, err := ParseFile("testdata/spectral-adidas.yaml")
	require.NoError(t, err, "should parse Adidas fixture")

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")
	require.NotNil(t, result, "result should not be nil")

	// Extends should be mapped correctly
	assert.Contains(t, result.Config.Extends, "recommended", "should map spectral:oas recommended")

	// Should have native rule overrides for the 30 built-in overrides
	assert.GreaterOrEqual(t, len(result.Config.Rules), 5, "should have rule entries")

	// Specific override checks
	opTagsEntry := findRuleEntry(result.Config.Rules, "style-operation-tags")
	require.NotNil(t, opTagsEntry, "should map operation-tags to style-operation-tags")
	require.NotNil(t, opTagsEntry.Severity, "should have severity")
	assert.Equal(t, validation.SeverityError, *opTagsEntry.Severity, "should be error severity")

	// Disabled rule
	descDupEntry := findRuleEntry(result.Config.Rules, "style-description-duplication")
	require.NotNil(t, descDupEntry, "should map description-duplication")
	require.NotNil(t, descDupEntry.Disabled, "should be disabled")
	assert.True(t, *descDupEntry.Disabled, "description-duplication should be disabled")

	// Should generate TypeScript for custom rules
	assert.Contains(t, result.GeneratedRules, "custom-adidas-operation-summary", "should generate summary rule")
	assert.Contains(t, result.GeneratedRules, "custom-adidas-version-semver", "should generate semver rule")
	assert.Contains(t, result.GeneratedRules, "custom-adidas-paths-kebab-case", "should generate casing rule")

	// Config should be loadable
	yamlData, err := marshalConfig(result.Config)
	require.NoError(t, err, "should marshal config")
	loaded, err := linter.LoadConfig(bytes.NewReader(yamlData))
	require.NoError(t, err, "generated config should be loadable")
	require.NotNil(t, loaded, "loaded config should not be nil")

	// Should have warnings (unsupported JSONPath, resolved: false, etc.)
	hasUnsupportedPathWarning := false
	hasResolvedWarning := false
	for _, w := range result.Warnings {
		if w.RuleID == "adidas-extension-check" && w.Phase == "generate" {
			hasUnsupportedPathWarning = true
		}
		if w.RuleID == "adidas-no-circular-refs" && w.Phase == "generate" {
			hasResolvedWarning = true
		}
	}
	assert.True(t, hasUnsupportedPathWarning, "should warn about unsupported JSONPath")
	assert.True(t, hasResolvedWarning, "should warn about resolved: false")
}

func TestGenerate_GoldenOutputConfig(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Extends: []ExtendsEntry{{Name: "spectral:oas", Modifier: "recommended"}},
		Rules: []Rule{
			{ID: "operation-tags", Severity: "error"},
			{ID: "operation-description", Severity: "off"},
			{
				ID:          "require-summary",
				Description: "Operations need summaries",
				Severity:    "warn",
				Given:       []string{"$.paths[*][*]"},
				Then:        []RuleCheck{{Field: "summary", Function: "truthy"}},
			},
		},
	}

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	yamlData, err := marshalConfig(result.Config)
	require.NoError(t, err, "should marshal config")

	configStr := string(yamlData)

	// Extends should be "recommended"
	assert.Contains(t, configStr, "recommended", "config should contain recommended extends")

	// Should have rule entries
	assert.Contains(t, configStr, "style-operation-tags", "config should have mapped rule ID")
	assert.Contains(t, configStr, "style-operation-description", "config should have mapped disabled rule")
	assert.Contains(t, configStr, "custom_rules", "config should have custom_rules section")
	assert.Contains(t, configStr, "rules/*.ts", "config should reference rules glob")

	// Round-trip: should be loadable
	loaded, err := linter.LoadConfig(bytes.NewReader(yamlData))
	require.NoError(t, err, "output config should be loadable")

	// Verify extends survived round-trip
	assert.Contains(t, loaded.Extends, "recommended", "loaded extends should include recommended")

	// Verify disabled rule survived round-trip
	opDescEntry := findRuleEntry(loaded.Rules, "style-operation-description")
	require.NotNil(t, opDescEntry, "should have disabled rule after round-trip")
	require.NotNil(t, opDescEntry.Disabled, "should still be disabled")
	assert.True(t, *opDescEntry.Disabled, "should still be disabled after round-trip")
}

func TestGenerate_BoolTrueOverrideSkipped(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Rules: []Rule{
			// bool true -> empty severity, meaning "use default" -> no override generated
			{ID: "operation-tags", Severity: ""},
		},
	}

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")

	// Should NOT have a rule entry for operation-tags since severity is empty
	entry := findRuleEntry(result.Config.Rules, "style-operation-tags")
	assert.Nil(t, entry, "bool true override should not generate a rule entry")
}

func TestGenerate_NoCustomRulesConfigWhenNoRules(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Rules: []Rule{
			{ID: "operation-tags", Severity: "error"}, // override only
		},
	}

	result, err := Generate(ir)
	require.NoError(t, err, "should generate")
	assert.Nil(t, result.Config.CustomRules, "no custom rules config when no .ts generated")
}

// --- helpers ---

func findRuleEntry(entries []linter.RuleEntry, id string) *linter.RuleEntry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}

func TestGenerate_WriteFilesCustomRulesDir(t *testing.T) {
	t.Parallel()

	ir := &IntermediateConfig{
		Rules: []Rule{
			{
				ID:          "my-rule",
				Description: "test rule",
				Severity:    "warn",
				Given:       []string{"$.info"},
				Then:        []RuleCheck{{Field: "description", Function: "truthy"}},
			},
		},
	}

	result, err := Generate(ir, WithRulesDir("./custom-rules"))
	require.NoError(t, err, "should generate")

	// Config should reference custom-rules/*.ts
	require.NotNil(t, result.Config.CustomRules, "should have custom rules config")
	require.Len(t, result.Config.CustomRules.Paths, 1, "should have one path")
	assert.Contains(t, result.Config.CustomRules.Paths[0], "custom-rules", "config should reference custom-rules dir")

	// WriteFiles should write to custom-rules/, not rules/
	tmpDir := t.TempDir()
	err = result.WriteFiles(tmpDir)
	require.NoError(t, err, "should write files")

	// Check files land in custom-rules/
	customDir := filepath.Join(tmpDir, "custom-rules")
	_, err = os.Stat(customDir)
	require.NoError(t, err, "custom-rules directory should exist")

	rulePath := filepath.Join(customDir, "custom-my-rule.ts")
	_, err = os.Stat(rulePath)
	require.NoError(t, err, "rule file should exist in custom-rules dir")

	// Verify rules/ was NOT created
	defaultDir := filepath.Join(tmpDir, "rules")
	_, err = os.Stat(defaultDir)
	assert.True(t, os.IsNotExist(err), "default rules/ directory should not exist")
}

func findAllRuleEntries(entries []linter.RuleEntry, id string) []linter.RuleEntry {
	var result []linter.RuleEntry
	for _, e := range entries {
		if e.ID == id {
			result = append(result, e)
		}
	}
	return result
}

func marshalConfig(cfg *linter.Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}
