package linter_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleConfig_GetSeverity(t *testing.T) {
	t.Parallel()

	t.Run("returns configured severity when set", func(t *testing.T) {
		t.Parallel()

		warningSeverity := validation.SeverityWarning
		config := linter.RuleConfig{
			Severity: &warningSeverity,
		}

		assert.Equal(t, validation.SeverityWarning, config.GetSeverity(validation.SeverityError))
	})

	t.Run("returns default severity when not set", func(t *testing.T) {
		t.Parallel()

		config := linter.RuleConfig{}

		assert.Equal(t, validation.SeverityError, config.GetSeverity(validation.SeverityError))
	})

	t.Run("returns configured severity overriding different default", func(t *testing.T) {
		t.Parallel()

		hintSeverity := validation.SeverityHint
		config := linter.RuleConfig{
			Severity: &hintSeverity,
		}

		assert.Equal(t, validation.SeverityHint, config.GetSeverity(validation.SeverityWarning))
	})
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	config := linter.NewConfig()
	assert.NotNil(t, config)
	assert.Equal(t, linter.OutputFormatText, config.OutputFormat)
	assert.NotNil(t, config.Rules)
	assert.NotNil(t, config.Categories)
	assert.NotNil(t, config.Extends)
}

func TestLoadConfig_ExtendsString(t *testing.T) {
	t.Parallel()

	configYAML := `extends: recommended`
	config, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)
	assert.Equal(t, []string{"recommended"}, config.Extends)
}

func TestLoadConfig_ExtendsList(t *testing.T) {
	t.Parallel()

	configYAML := `extends:
  - recommended
  - strict`
	config, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)
	assert.Equal(t, []string{"recommended", "strict"}, config.Extends)
}

func TestLoadConfig_MatchRegex(t *testing.T) {
	t.Parallel()

	configYAML := `rules:
  - id: validation-required
    match: ".*title.*"`
	config, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)
	require.Len(t, config.Rules, 1)
	require.NotNil(t, config.Rules[0].Match)
	assert.Equal(t, regexp.MustCompile(".*title.*").String(), config.Rules[0].Match.String())
}

func TestLoadConfig_CustomRulesRoundTrip(t *testing.T) {
	t.Parallel()

	configYAML := `extends: all
custom_rules:
  paths:
    - "./rules/*.ts"
    - "./extra/*.ts"`
	config, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err, "should load config with custom_rules")
	require.NotNil(t, config.CustomRules, "custom_rules should survive UnmarshalYAML round-trip")
	assert.Equal(t, []string{"./rules/*.ts", "./extra/*.ts"}, config.CustomRules.Paths, "custom_rules.paths should be preserved")
}

func TestLoadConfig_CategorySeverityAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		yaml             string
		expectedSeverity validation.Severity
	}{
		{
			name: "error severity",
			yaml: `categories:
  style:
    severity: error`,
			expectedSeverity: validation.SeverityError,
		},
		{
			name: "warn alias for warning",
			yaml: `categories:
  style:
    severity: warn`,
			expectedSeverity: validation.SeverityWarning,
		},
		{
			name: "warning severity",
			yaml: `categories:
  style:
    severity: warning`,
			expectedSeverity: validation.SeverityWarning,
		},
		{
			name: "hint severity",
			yaml: `categories:
  style:
    severity: hint`,
			expectedSeverity: validation.SeverityHint,
		},
		{
			name: "info alias for hint",
			yaml: `categories:
  style:
    severity: info`,
			expectedSeverity: validation.SeverityHint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := linter.LoadConfig(strings.NewReader(tt.yaml))
			require.NoError(t, err)
			require.NotNil(t, config.Categories["style"].Severity, "severity should be set")
			assert.Equal(t, tt.expectedSeverity, *config.Categories["style"].Severity, "severity should match expected")
		})
	}
}

func TestLoadConfig_CategoryEnabled(t *testing.T) {
	t.Parallel()

	configYAML := `categories:
  security:
    enabled: false`
	config, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)
	require.NotNil(t, config.Categories["security"].Enabled, "enabled should be set")
	assert.False(t, *config.Categories["security"].Enabled, "security category should be disabled")
}

func TestLoadConfig_CategoryInvalidSeverity(t *testing.T) {
	t.Parallel()

	configYAML := `categories:
  style:
    severity: critical`
	_, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown severity")
}

func TestLoadConfig_RuleSeverityAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		yaml             string
		expectedSeverity validation.Severity
	}{
		{
			name: "warn alias",
			yaml: `rules:
  - id: test-rule
    severity: warn`,
			expectedSeverity: validation.SeverityWarning,
		},
		{
			name: "info alias",
			yaml: `rules:
  - id: test-rule
    severity: info`,
			expectedSeverity: validation.SeverityHint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := linter.LoadConfig(strings.NewReader(tt.yaml))
			require.NoError(t, err)
			require.Len(t, config.Rules, 1)
			require.NotNil(t, config.Rules[0].Severity, "severity should be set")
			assert.Equal(t, tt.expectedSeverity, *config.Rules[0].Severity, "severity should match expected")
		})
	}
}

func TestLoadConfig_RuleInvalidSeverity(t *testing.T) {
	t.Parallel()

	configYAML := `rules:
  - id: test-rule
    severity: critical`
	_, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown severity")
}

func TestLoadConfig_ExtendsInvalidType(t *testing.T) {
	t.Parallel()

	configYAML := `extends:
  key: value`
	_, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extends must be a string or list of strings")
}

func TestLoadConfig_ExtendsNull(t *testing.T) {
	t.Parallel()

	configYAML := `extends: null`
	config, err := linter.LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)
	assert.Equal(t, []string{"all"}, config.Extends, "null extends should default to all")
}

func TestLoadConfigFromFile_Success(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/lint.yaml"
	err := os.WriteFile(tmpFile, []byte("extends: recommended\n"), 0644)
	require.NoError(t, err)

	config, err := linter.LoadConfigFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, []string{"recommended"}, config.Extends)
}

func TestLoadConfigFromFile_Error(t *testing.T) {
	t.Parallel()

	_, err := linter.LoadConfigFromFile("/nonexistent/path/lint.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open config file")
}

func TestConfig_ValidateMissingRuleID(t *testing.T) {
	t.Parallel()

	config := &linter.Config{
		Rules: []linter.RuleEntry{{}},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rule entry missing id")
}
