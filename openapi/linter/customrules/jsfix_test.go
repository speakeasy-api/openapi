package customrules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi/linter/customrules"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSFix_NonInteractive(t *testing.T) {
	t.Parallel()

	rt, err := customrules.NewRuntime(&testLogger{}, nil)
	require.NoError(t, err, "creating runtime should succeed")

	// Create a non-interactive fix via JS
	_, err = rt.RunScript("test", `
		var fix = createFix({
			description: "remove trailing slash",
			apply: function(doc) {
				// no-op for test
			}
		});
	`)
	require.NoError(t, err, "creating fix should succeed")

	// Create an error with fix via JS
	result, err := rt.RunScript("test2", `
		var err = createValidationErrorWithFix("warning", "test-rule", "has trailing slash", null, fix);
		err;
	`)
	require.NoError(t, err, "creating error with fix should succeed")

	exported := result.Export()
	vErr, ok := exported.(*validation.Error)
	require.True(t, ok, "result should be a validation.Error")
	require.NotNil(t, vErr.Fix, "error should have a fix attached")
	assert.Equal(t, "remove trailing slash", vErr.Fix.Description(), "fix description should match")
	assert.False(t, vErr.Fix.Interactive(), "fix should be non-interactive")
	assert.Nil(t, vErr.Fix.Prompts(), "non-interactive fix should have no prompts")
}

func TestJSFix_Interactive(t *testing.T) {
	t.Parallel()

	rt, err := customrules.NewRuntime(&testLogger{}, nil)
	require.NoError(t, err, "creating runtime should succeed")

	_, err = rt.RunScript("test", `
		var fix = createFix({
			description: "add description",
			interactive: true,
			prompts: [
				{ type: "text", message: "Enter a description", "default": "A sample API" },
				{ type: "choice", message: "Pick a tag", choices: ["users", "admin"] }
			],
			apply: function(doc, inputs) {
				// no-op for test
			}
		});
	`)
	require.NoError(t, err, "creating interactive fix should succeed")

	result, err := rt.RunScript("test2", `
		var err = createValidationErrorWithFix("warning", "test-rule", "missing description", null, fix);
		err;
	`)
	require.NoError(t, err, "creating error with interactive fix should succeed")

	exported := result.Export()
	vErr, ok := exported.(*validation.Error)
	require.True(t, ok, "result should be a validation.Error")
	require.NotNil(t, vErr.Fix, "error should have a fix attached")

	fix := vErr.Fix
	assert.True(t, fix.Interactive(), "fix should be interactive")
	assert.Len(t, fix.Prompts(), 2, "fix should have 2 prompts")

	prompts := fix.Prompts()
	assert.Equal(t, validation.PromptFreeText, prompts[0].Type, "first prompt should be free text")
	assert.Equal(t, "Enter a description", prompts[0].Message, "first prompt message should match")
	assert.Equal(t, "A sample API", prompts[0].Default, "first prompt default should match")
	assert.Equal(t, validation.PromptChoice, prompts[1].Type, "second prompt should be choice")
	assert.Equal(t, []string{"users", "admin"}, prompts[1].Choices, "second prompt choices should match")
}

func TestJSFix_SetInput(t *testing.T) {
	t.Parallel()

	rt, err := customrules.NewRuntime(&testLogger{}, nil)
	require.NoError(t, err, "creating runtime should succeed")

	_, err = rt.RunScript("test", `
		var fix = createFix({
			description: "needs input",
			interactive: true,
			prompts: [{ type: "text", message: "Enter value" }],
			apply: function(doc, inputs) {}
		});
	`)
	require.NoError(t, err, "creating fix should succeed")

	result, err := rt.RunScript("test2", `
		createValidationErrorWithFix("warning", "test-rule", "msg", null, fix);
	`)
	require.NoError(t, err)

	vErr := result.Export().(*validation.Error)
	fix := vErr.Fix

	// Wrong number of inputs
	require.Error(t, fix.SetInput([]string{"a", "b"}), "SetInput with wrong count should fail")

	// Correct number of inputs
	require.NoError(t, fix.SetInput([]string{"my value"}), "SetInput with correct count should succeed")
}

func TestJSFix_CreateFix_MissingDescription(t *testing.T) {
	t.Parallel()

	rt, err := customrules.NewRuntime(&testLogger{}, nil)
	require.NoError(t, err, "creating runtime should succeed")

	_, err = rt.RunScript("test", `
		createFix({ apply: function(doc) {} });
	`)
	require.Error(t, err, "createFix without description should fail")
}

func TestJSFix_CreateFix_MissingApply(t *testing.T) {
	t.Parallel()

	rt, err := customrules.NewRuntime(&testLogger{}, nil)
	require.NoError(t, err, "creating runtime should succeed")

	_, err = rt.RunScript("test", `
		createFix({ description: "test" });
	`)
	require.Error(t, err, "createFix without apply should fail")
}

// testLogger is a simple logger for testing.
type testLogger struct{}

func (l *testLogger) Log(args ...any)   {}
func (l *testLogger) Warn(args ...any)  {}
func (l *testLogger) Error(args ...any) {}
