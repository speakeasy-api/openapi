package linter_test

import (
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

func TestConfig_ValidateMissingRuleID(t *testing.T) {
	t.Parallel()

	config := &linter.Config{
		Rules: []linter.RuleEntry{{}},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rule entry missing id")
}
