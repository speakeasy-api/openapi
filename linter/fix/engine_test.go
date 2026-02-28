package fix_test

import (
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
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

// mockInteractiveFixWithSetInputErr is an interactive fix that fails on SetInput.
type mockInteractiveFixWithSetInputErr struct {
	mockInteractiveFix
	setInputErr error
}

func (f *mockInteractiveFixWithSetInputErr) SetInput(responses []string) error {
	return f.setInputErr
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

func TestEngine_SkipsNonValidationErrors(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		errors.New("plain error, not a validation.Error"),
		makeError("test-rule", 1, 1, "real issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should only apply fix for the validation error")
	assert.True(t, f.applied, "fix for validation error should have been applied")
}

func TestEngine_ModeInteractive_PrompterError(t *testing.T) {
	t.Parallel()

	f := &mockInteractiveFix{
		description: "needs input",
		prompts:     []validation.Prompt{{Type: validation.PromptFreeText, Message: "enter value"}},
	}
	prompter := &mockPrompter{err: errors.New("terminal closed")}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeInteractive}, prompter, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should not apply fix when prompter fails")
	assert.Len(t, result.Failed, 1, "should record the fix as failed")
	assert.False(t, f.applied, "fix should not have been applied")
}

func TestEngine_ModeInteractive_SetInputError(t *testing.T) {
	t.Parallel()

	f := &mockInteractiveFixWithSetInputErr{
		mockInteractiveFix: mockInteractiveFix{
			description: "needs input",
			prompts:     []validation.Prompt{{Type: validation.PromptFreeText, Message: "enter value"}},
		},
		setInputErr: errors.New("invalid input"),
	}
	prompter := &mockPrompter{responses: []string{"bad value"}}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeInteractive}, prompter, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Empty(t, result.Applied, "should not apply fix when SetInput fails")
	assert.Len(t, result.Failed, 1, "should record the fix as failed")
}

func TestEngine_NodeFix_UsesApplyNode(t *testing.T) {
	t.Parallel()

	f := &mockNodeFix{mockFix: mockFix{description: "node fix"}}
	rootNode := &yaml.Node{Kind: yaml.MappingNode}
	doc := &openapi.OpenAPI{}
	doc.GetCore().SetRootNode(rootNode)

	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), doc, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "should apply the fix")
	assert.True(t, f.nodeApplied, "ApplyNode should be called when root node is present")
	assert.False(t, f.applied, "Apply should not be called when ApplyNode succeeds")
}

func TestEngine_DryRun_ConflictDetection(t *testing.T) {
	t.Parallel()

	f1 := &mockFix{description: "first fix"}
	f2 := &mockFix{description: "second fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto, DryRun: true}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("same-rule", 5, 3, "issue 1", f1),
		makeError("same-rule", 5, 3, "issue 2", f2),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	assert.Len(t, result.Applied, 1, "dry-run should record first fix as would-apply")
	assert.Len(t, result.Skipped, 1, "dry-run should skip second fix as conflict")
	assert.False(t, f1.applied, "fix should NOT have been actually applied in dry-run")
	assert.False(t, f2.applied, "fix should NOT have been actually applied in dry-run")
}

func TestApplyNodeFix_WithNodeFix(t *testing.T) {
	t.Parallel()

	f := &mockNodeFix{mockFix: mockFix{description: "node fix"}}
	rootNode := &yaml.Node{Kind: yaml.MappingNode}
	doc := &openapi.OpenAPI{}

	err := fix.ApplyNodeFix(f, doc, rootNode)
	require.NoError(t, err, "ApplyNodeFix should not fail")
	assert.True(t, f.nodeApplied, "ApplyNode should be called")
	assert.False(t, f.applied, "Apply should not be called")
}

func TestApplyNodeFix_NilRootNode(t *testing.T) {
	t.Parallel()

	f := &mockNodeFix{mockFix: mockFix{description: "node fix"}}
	doc := &openapi.OpenAPI{}

	err := fix.ApplyNodeFix(f, doc, nil)
	require.NoError(t, err, "ApplyNodeFix should not fail")
	assert.False(t, f.nodeApplied, "ApplyNode should not be called with nil root")
	assert.True(t, f.applied, "Apply should be called as fallback")
}

func TestApplyNodeFix_RegularFix(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "regular fix"}
	rootNode := &yaml.Node{Kind: yaml.MappingNode}
	doc := &openapi.OpenAPI{}

	err := fix.ApplyNodeFix(f, doc, rootNode)
	require.NoError(t, err, "ApplyNodeFix should not fail")
	assert.True(t, f.applied, "Apply should be called for non-NodeFix")
}

// mockChangeDescriberFix is a fix that implements ChangeDescriber.
type mockChangeDescriberFix struct {
	mockFix
	before string
	after  string
}

func (f *mockChangeDescriberFix) DescribeChange() (string, string) {
	return f.before, f.after
}

func TestEngine_ChangeDescriber_PopulatesBeforeAfter(t *testing.T) {
	t.Parallel()

	f := &mockChangeDescriberFix{
		mockFix: mockFix{description: "trim slash"},
		before:  "/users/",
		after:   "/users",
	}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "trailing slash", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	require.Len(t, result.Applied, 1, "should apply the fix")
	assert.Equal(t, "/users/", result.Applied[0].Before, "should populate Before from ChangeDescriber")
	assert.Equal(t, "/users", result.Applied[0].After, "should populate After from ChangeDescriber")
}

func TestEngine_ChangeDescriber_DryRun(t *testing.T) {
	t.Parallel()

	f := &mockChangeDescriberFix{
		mockFix: mockFix{description: "upgrade https"},
		before:  "http://api.example.com",
		after:   "https://api.example.com",
	}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto, DryRun: true}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "use https", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	require.Len(t, result.Applied, 1, "should record the fix as would-apply")
	assert.Equal(t, "http://api.example.com", result.Applied[0].Before, "dry-run should populate Before")
	assert.Equal(t, "https://api.example.com", result.Applied[0].After, "dry-run should populate After")
	assert.False(t, f.applied, "fix should NOT have been actually applied in dry-run")
}

func TestEngine_NoChangeDescriber_EmptyBeforeAfter(t *testing.T) {
	t.Parallel()

	f := &mockFix{description: "simple fix"}
	engine := fix.NewEngine(fix.Options{Mode: fix.ModeAuto}, nil, nil)
	result, err := engine.ProcessErrors(t.Context(), &openapi.OpenAPI{}, []error{
		makeError("test-rule", 1, 1, "issue", f),
	})

	require.NoError(t, err, "ProcessErrors should not fail")
	require.Len(t, result.Applied, 1, "should apply the fix")
	assert.Empty(t, result.Applied[0].Before, "Before should be empty without ChangeDescriber")
	assert.Empty(t, result.Applied[0].After, "After should be empty without ChangeDescriber")
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
