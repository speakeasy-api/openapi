package fix_test

import (
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// mockFix is a non-interactive fix for testing.
type mockFix struct {
	description string
	applied     bool
	applyErr    error
}

func (f *mockFix) Description() string          { return f.description }
func (f *mockFix) Interactive() bool            { return false }
func (f *mockFix) Prompts() []validation.Prompt { return nil }
func (f *mockFix) SetInput([]string) error      { return nil }
func (f *mockFix) Apply(doc any) error {
	f.applied = true
	return f.applyErr
}

// mockInteractiveFix is an interactive fix for testing.
type mockInteractiveFix struct {
	description string
	prompts     []validation.Prompt
	applied     bool
	inputs      []string
}

func (f *mockInteractiveFix) Description() string          { return f.description }
func (f *mockInteractiveFix) Interactive() bool            { return true }
func (f *mockInteractiveFix) Prompts() []validation.Prompt { return f.prompts }
func (f *mockInteractiveFix) SetInput(responses []string) error {
	f.inputs = responses
	return nil
}
func (f *mockInteractiveFix) Apply(doc any) error {
	f.applied = true
	return nil
}

// mockNodeFix implements NodeFix for testing.
type mockNodeFix struct {
	mockFix
	nodeApplied bool
}

func (f *mockNodeFix) ApplyNode(rootNode *yaml.Node) error {
	f.nodeApplied = true
	return nil
}

// mockPrompter is a test prompter that returns predefined responses.
type mockPrompter struct {
	responses []string
	err       error
	called    bool
}

func (p *mockPrompter) PromptFix(_ *validation.Error, _ validation.Fix) ([]string, error) {
	p.called = true
	return p.responses, p.err
}

func (p *mockPrompter) Confirm(_ string) (bool, error) {
	return true, nil
}

func makeError(rule string, line, col int, msg string, f validation.Fix) error {
	return &validation.Error{
		UnderlyingError: errors.New(msg),
		Node:            &yaml.Node{Line: line, Column: col},
		Severity:        validation.SeverityWarning,
		Rule:            rule,
		Fix:             f,
	}
}

func TestEngine_ModeNone(t *testing.T) {
	t.Parallel()

	engine := fix.NewEngine(fix.Options{Mode: fix.ModeNone}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "some error", &mockFix{description: "fix it"}),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should not apply any fixes in ModeNone")
	assert.Empty(t, result.Skipped, "should not skip any fixes in ModeNone")
	assert.Empty(t, result.Failed, "should not fail any fixes in ModeNone")
}

func TestEngine_ModeAuto_NonInteractive(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "auto fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should apply the non-interactive fix")
	assert.True(t, f.applied, "fix should have been applied")
}

func TestEngine_ModeAuto_SkipsInteractive(t *testing.T) {
	t.Parallel()

	f := &mockInteractiveFix{
		description: "needs input",
		prompts:     []validation.Prompt{{Type: validation.PromptFreeText, Message: "enter value"}},
	}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should not apply interactive fix in auto mode")
	assert.Len(t, result.Skipped, 1, "should skip the interactive fix")
	assert.False(t, f.applied, "fix should not have been applied")
}

func TestEngine_ModeInteractive_PromptsUser(t *testing.T) {
	t.Parallel()

	f := &mockInteractiveFix{
		description: "needs input",
		prompts:     []validation.Prompt{{Type: validation.PromptFreeText, Message: "enter value"}},
	}
	prompter := &mockPrompter{responses: []string{"user answer"}}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeInteractive}, prompter, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.True(t, prompter.called, "prompter should have been called")
	assert.Len(t, result.Applied, 1, "should apply the fix after prompting")
	assert.True(t, f.applied, "fix should have been applied")
	assert.Equal(t, []string{"user answer"}, f.inputs, "fix should have received user input")
}

func TestEngine_ModeInteractive_UserSkips(t *testing.T) {
	t.Parallel()

	f := &mockInteractiveFix{
		description: "needs input",
		prompts:     []validation.Prompt{{Type: validation.PromptFreeText, Message: "enter value"}},
	}
	prompter := &mockPrompter{err: validation.ErrSkipFix}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeInteractive}, prompter, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should not apply skipped fix")
	assert.Len(t, result.Skipped, 1, "should record skipped fix")
	assert.False(t, f.applied, "fix should not have been applied")
}

func TestEngine_DryRun(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "auto fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto, DryRun: true}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should record the fix as would-apply")
	assert.False(t, f.applied, "fix should NOT have been actually applied in dry-run")
}

func TestEngine_ConflictDetection_SameRule(t *testing.T) {
	t.Parallel()

	f1 := &mockFix{description: "first fix"}
	f2 := &mockFix{description: "second fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("same-rule", 5, 3, "issue 1", f1),
		makeError("same-rule", 5, 3, "issue 2", f2),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should apply the first fix")
	assert.Len(t, result.Skipped, 1, "should skip the second fix as conflict")
	assert.True(t, f1.applied, "first fix should have been applied")
	assert.False(t, f2.applied, "second fix should not have been applied")
}

func TestEngine_ConflictDetection_DifferentRules(t *testing.T) {
	t.Parallel()

	f1 := &mockFix{description: "first fix"}
	f2 := &mockFix{description: "second fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("rule-a", 5, 3, "issue 1", f1),
		makeError("rule-b", 5, 3, "issue 2", f2),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 2, "should apply both fixes from different rules at the same location")
	assert.Empty(t, result.Skipped, "should not skip fixes from different rules")
	assert.True(t, f1.applied, "first fix should have been applied")
	assert.True(t, f2.applied, "second fix should have been applied")
}

func TestEngine_FailedFix(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "broken fix", applyErr: errors.New("fix failed")}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail even when fixes fail")
	assert.Empty(t, result.Applied, "should not record failed fix as applied")
	assert.Len(t, result.Failed, 1, "should record the failed fix")
}

func TestEngine_NodeFix_FallsBackToApply(t *testing.T) {
	t.Parallel()

	// When doc.GetRootNode() returns nil, NodeFix falls back to Apply()
	f := &mockNodeFix{mockFix: mockFix{description: "node fix"}}
	doc := &openapi.OpenAPI{}

	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), doc, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should apply the fix")
	assert.False(t, f.nodeApplied, "ApplyNode should not be called when root node is nil")
	assert.True(t, f.applied, "Apply should be called as fallback")
}

func TestEngine_RegistryFix(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "registry fix"}
	registry := fix.NewFixRegistry()
	registry.Register("validation-empty-value", func(_ *validation.Error) validation.Fix {
		return f
	})

	// Error without a fix attached, but registry provides one
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, registry)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("validation-empty-value", 1, 1, "empty value", nil),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should apply the registry-provided fix")
	assert.True(t, f.applied, "fix should have been applied")
}

func TestEngine_NoFixableErrors(t *testing.T) {
	t.Parallel()

	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue without fix", nil),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should have no applied fixes")
	assert.Empty(t, result.Skipped, "should have no skipped fixes")
	assert.Empty(t, result.Failed, "should have no failed fixes")
}

func TestEngine_ModeInteractive_SkipsWhenNoPrompter(t *testing.T) {
	t.Parallel()

	f := &mockInteractiveFix{
		description: "needs input",
		prompts:     []validation.Prompt{{Type: validation.PromptFreeText, Message: "enter value"}},
	}
	// Interactive mode but nil prompter — interactive fixes should be skipped
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeInteractive}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should not apply interactive fix without prompter")
	assert.Len(t, result.Skipped, 1, "should skip the interactive fix")
	assert.Equal(t, fix.SkipInteractive, result.Skipped[0].Reason, "skip reason should be SkipInteractive")
	assert.False(t, f.applied, "fix should not have been applied")
}

func TestEngine_ModeInteractive_NonInteractiveFixAppliesWithoutPrompter(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "auto fix"}
	// Interactive mode but nil prompter — non-interactive fixes should still apply
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeInteractive}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should apply non-interactive fix even without prompter")
	assert.True(t, f.applied, "fix should have been applied")
}

func TestEngine_SortsByLocation(t *testing.T) {
	t.Parallel()

	var order []string
	makeFix := func(name string) *mockFix {
		return &mockFix{description: name}
	}

	f1 := makeFix("fix-line10")
	f2 := makeFix("fix-line2")
	f3 := makeFix("fix-line5")

	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("rule", 10, 1, "issue at line 10", f1),
		makeError("rule", 2, 1, "issue at line 2", f2),
		makeError("rule", 5, 1, "issue at line 5", f3),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 3, "should apply all fixes")

	for _, af := range result.Applied {
		order = append(order, af.Fix.Description())
	}
	assert.Equal(t, []string{"fix-line2", "fix-line5", "fix-line10"}, order, "fixes should be applied in location order")
}
