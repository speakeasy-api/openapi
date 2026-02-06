package customrules_test

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomRule_SeverityOverride(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	warningSeverity := validation.SeverityWarning
	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Rules: []linter.RuleEntry{
			{
				ID:       "custom-require-operation-summary",
				Severity: &warningSeverity,
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Find the custom rule error
	var customRuleErr *validation.Error
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) && vErr.Rule == "custom-require-operation-summary" {
			customRuleErr = vErr
			break
		}
	}

	require.NotNil(t, customRuleErr, "should find custom rule error")
	assert.Equal(t, validation.SeverityWarning, customRuleErr.Severity, "severity should be overridden to warning")
}

func TestCustomRule_Disabled(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Rules: []linter.RuleEntry{
			{
				ID:       "custom-require-operation-summary",
				Disabled: pointer.From(true),
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Check that custom rule error is NOT present
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) {
			assert.NotEqual(t, "custom-require-operation-summary", vErr.Rule, "disabled rule should not produce errors")
		}
	}
}

func TestCustomRule_MatchRegex_ChangeSeverity(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	hintSeverity := validation.SeverityHint
	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Rules: []linter.RuleEntry{
			{
				ID:       "custom-require-operation-summary",
				Match:    regexp.MustCompile(".*missing a summary.*"),
				Severity: &hintSeverity,
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Find the custom rule error
	var customRuleErr *validation.Error
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) && vErr.Rule == "custom-require-operation-summary" {
			customRuleErr = vErr
			break
		}
	}

	require.NotNil(t, customRuleErr, "should find custom rule error")
	assert.Equal(t, validation.SeverityHint, customRuleErr.Severity, "severity should be overridden to hint via match")
}

func TestCustomRule_MatchRegex_Disable(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Rules: []linter.RuleEntry{
			{
				ID:       "custom-require-operation-summary",
				Match:    regexp.MustCompile(".*missing a summary.*"),
				Disabled: pointer.From(true),
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Check that custom rule error is filtered out by match regex
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) {
			assert.NotEqual(t, "custom-require-operation-summary", vErr.Rule, "matched error should be disabled")
		}
	}
}

func TestCustomRule_MatchRegex_NoMatch_KeepsOriginal(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	hintSeverity := validation.SeverityHint
	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Rules: []linter.RuleEntry{
			{
				ID:       "custom-require-operation-summary",
				Match:    regexp.MustCompile(".*this will not match.*"),
				Severity: &hintSeverity,
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Find the custom rule error - should still exist with default severity
	var customRuleErr *validation.Error
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) && vErr.Rule == "custom-require-operation-summary" {
			customRuleErr = vErr
			break
		}
	}

	require.NotNil(t, customRuleErr, "should find custom rule error")
	assert.Equal(t, validation.SeverityWarning, customRuleErr.Severity, "non-matching error should keep default severity")
}

func TestCustomRule_CategorySeverityOverride(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	hintSeverity := validation.SeverityHint
	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Categories: map[string]linter.CategoryConfig{
			"style": {
				Severity: &hintSeverity,
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Find the custom rule error
	var customRuleErr *validation.Error
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) && vErr.Rule == "custom-require-operation-summary" {
			customRuleErr = vErr
			break
		}
	}

	require.NotNil(t, customRuleErr, "should find custom rule error")
	assert.Equal(t, validation.SeverityHint, customRuleErr.Severity, "category severity should override rule default")
}

func TestCustomRule_CategoryDisabled(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	config := &linter.Config{
		Extends: []string{"all"},
		CustomRules: &linter.CustomRulesConfig{
			Paths: []string{testdataPath},
		},
		Categories: map[string]linter.CategoryConfig{
			"style": {
				Enabled: pointer.From(false),
			},
		},
	}

	lint, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "test.yaml")
	output, err := lint.Lint(ctx, docInfo, validationErrs, nil)
	require.NoError(t, err)

	// Check that custom rule error is NOT present (since it's in the 'style' category)
	for _, result := range output.Results {
		var vErr *validation.Error
		if errors.As(result, &vErr) {
			assert.NotEqual(t, "custom-require-operation-summary", vErr.Rule, "category disabled rule should not produce errors")
		}
	}
}
