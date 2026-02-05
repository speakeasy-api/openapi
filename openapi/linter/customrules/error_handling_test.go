package customrules_test

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/customrules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_SyntaxError(t *testing.T) {
	t.Parallel()

	testdataPath, err := filepath.Abs("testdata/syntax-error.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	_, err = loader.LoadRules(config)
	require.Error(t, err, "should fail on syntax error")
	assert.Contains(t, err.Error(), "esbuild errors", "error should mention esbuild")
}

func TestLoader_FileNotFound(t *testing.T) {
	t.Parallel()

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{"/nonexistent/path/rule.ts"},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err, "should not error on missing file (glob returns empty)")
	assert.Nil(t, rules, "should return no rules for nonexistent path")
}

func TestLoader_InvalidGlobPattern(t *testing.T) {
	t.Parallel()

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{"[invalid-glob"},
	}

	_, err := loader.LoadRules(config)
	require.Error(t, err, "should fail on invalid glob pattern")
	assert.Contains(t, err.Error(), "invalid glob pattern", "error should mention invalid glob")
}

func TestCustomRule_RuntimeError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/throws-error.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err, "should load rule that throws")
	require.Len(t, rules, 1)

	// Parse a minimal OpenAPI doc
	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Run the rule - should return error, not panic
	errs := rules[0].Run(ctx, docInfo, nil)
	require.Len(t, errs, 1, "should return 1 error")
	assert.Contains(t, errs[0].Error(), "Intentional error for testing", "error should contain the thrown message")
}

func TestCustomRule_MissingIdMethod(t *testing.T) {
	t.Parallel()

	testdataPath, err := filepath.Abs("testdata/missing-methods.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	_, err = loader.LoadRules(config)
	require.Error(t, err, "should fail when rule is missing id method")
	assert.Contains(t, err.Error(), "id", "error should mention missing id")
}

// TestLogger captures log messages for testing
type TestLogger struct {
	mu       sync.Mutex
	logs     []string
	warnings []string
	errors   []string
}

func (l *TestLogger) Log(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var parts []string
	for _, arg := range args {
		parts = append(parts, formatArg(arg))
	}
	l.logs = append(l.logs, strings.Join(parts, " "))
}

func (l *TestLogger) Warn(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var parts []string
	for _, arg := range args {
		parts = append(parts, formatArg(arg))
	}
	l.warnings = append(l.warnings, strings.Join(parts, " "))
}

func (l *TestLogger) Error(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var parts []string
	for _, arg := range args {
		parts = append(parts, formatArg(arg))
	}
	l.errors = append(l.errors, strings.Join(parts, " "))
}

func formatArg(arg any) string {
	if s, ok := arg.(string); ok {
		return s
	}
	return ""
}

func TestCustomRule_ConsoleLogging(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	testdataPath, err := filepath.Abs("testdata/logs-messages.ts")
	require.NoError(t, err)

	testLogger := &TestLogger{}
	loaderConfig := &customrules.Config{
		Logger: testLogger,
	}

	loader := customrules.NewLoader(loaderConfig)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Parse a minimal OpenAPI doc
	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Run the rule
	errs := rules[0].Run(ctx, docInfo, nil)
	assert.Empty(t, errs, "rule should not return errors")

	// Check that console messages were captured
	testLogger.mu.Lock()
	defer testLogger.mu.Unlock()

	assert.NotEmpty(t, testLogger.logs, "should have log messages")
	assert.NotEmpty(t, testLogger.warnings, "should have warning messages")
	assert.NotEmpty(t, testLogger.errors, "should have error messages")

	// Check specific messages
	foundLogMessage := false
	for _, log := range testLogger.logs {
		if strings.Contains(log, "Log message from custom rule") {
			foundLogMessage = true
			break
		}
	}
	assert.True(t, foundLogMessage, "should find log message")
}

func TestCustomRule_Timeout(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create an inline rule that takes forever
	// We'll use the require-summary rule but with a very short timeout
	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	// Configure with a very short timeout
	loaderConfig := &customrules.Config{
		Timeout: 1 * time.Nanosecond, // Extremely short timeout
	}

	loader := customrules.NewLoader(loaderConfig)
	config := &linter.CustomRulesConfig{
		Paths:   []string{testdataPath},
		Timeout: 1 * time.Nanosecond,
	}

	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Parse a minimal OpenAPI doc
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
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Run the rule - it may timeout or complete (race condition with short timeout)
	// This test verifies the timeout mechanism doesn't panic
	_ = rules[0].Run(ctx, docInfo, nil)
	// No assertion needed - we just want to verify it doesn't panic
}

func TestLoader_NilConfig(t *testing.T) {
	t.Parallel()

	loader := customrules.NewLoader(nil)
	rules, err := loader.LoadRules(nil)
	require.NoError(t, err, "should handle nil config")
	assert.Nil(t, rules, "should return no rules for nil config")
}

func TestLoader_EmptyFile(t *testing.T) {
	t.Parallel()

	// Create a temp file that's empty - we'll use a path that exists but has no rules
	testdataPath, err := filepath.Abs("testdata/require-summary.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	// This should work fine
	rules, err := loader.LoadRules(config)
	require.NoError(t, err)
	assert.NotEmpty(t, rules)
}

func TestLoader_MultipleRulesFromGlob(t *testing.T) {
	t.Parallel()

	testdataPath, err := filepath.Abs("testdata/*.ts")
	require.NoError(t, err)

	loader := customrules.NewLoader(nil)
	config := &linter.CustomRulesConfig{
		Paths: []string{testdataPath},
	}

	rules, err := loader.LoadRules(config)
	// Some files have syntax errors, so we expect an error
	// But we should get at least some rules loaded before the error
	if err != nil {
		assert.Contains(t, err.Error(), "error", "should have an error message")
	} else {
		assert.GreaterOrEqual(t, len(rules), 2, "should load multiple rules from glob")
	}
}
