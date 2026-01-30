package linter_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
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
