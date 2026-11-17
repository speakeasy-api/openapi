package rules_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allRules returns every built-in rule.
func allRules() []linter.RuleRunner[*arazzo.Arazzo] {
	return []linter.RuleRunner[*arazzo.Arazzo]{
		&rules.UnreachableStepRule{},
		&rules.DependencyCycleRule{},
		&rules.UnusedComponentRule{},
		&rules.UnusedWorkflowInputRule{},
		&rules.UnusedStepOutputRule{},
		&rules.StepSuccessCriteriaRule{},
		&rules.SourceDescriptionTypeRule{},
		&rules.InfoDescriptionRule{},
		&rules.WorkflowDescriptionRule{},
		&rules.RetryAfterRule{},
		&rules.SourceDescriptionHTTPSRule{},
	}
}

// parseDoc parses an Arazzo document without validation.
func parseDoc(t *testing.T, doc string) *arazzo.Arazzo {
	t.Helper()

	a, _, err := arazzo.Unmarshal(t.Context(), bytes.NewBufferString(doc), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")
	return a
}

// findings formats errors as `<severity>: <message>`.
func findings(errs []error) []string {
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		var vErr *validation.Error
		if !errors.As(err, &vErr) {
			out = append(out, err.Error())
			continue
		}
		out = append(out, vErr.Severity.String()+": "+vErr.UnderlyingError.Error())
	}
	return out
}

func runRule(t *testing.T, rule linter.RuleRunner[*arazzo.Arazzo], doc string) []string {
	t.Helper()

	return findings(rule.Run(t.Context(), linter.NewDocumentInfo(parseDoc(t, doc), "test.arazzo.yaml"), &linter.RuleConfig{}))
}

func TestRules_Metadata_Success(t *testing.T) {
	t.Parallel()

	for _, rule := range allRules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, rule.Category(), "category should be set")
			assert.NotEmpty(t, rule.Summary(), "summary should be set")
			assert.NotEmpty(t, rule.Description(), "description should be set")
			assert.Equal(t, "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#"+rule.ID(), rule.Link(), "link should point at the rule docs")
			assert.Nil(t, rule.Versions(), "rules should apply to all versions")
			assert.NotEmpty(t, rule.DefaultSeverity().String(), "default severity should be set")

			fixer, ok := rule.(interface{ HowToFix() string })
			require.True(t, ok, "rule should explain how to fix it")
			assert.NotEmpty(t, fixer.HowToFix(), "how to fix should be set")
		})
	}
}

func TestRules_NilDocument_Success(t *testing.T) {
	t.Parallel()

	for _, rule := range allRules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			assert.Empty(t, rule.Run(ctx, nil, &linter.RuleConfig{}), "nil document info should have no findings")
			assert.Empty(t, rule.Run(ctx, linter.NewDocumentInfo[*arazzo.Arazzo](nil, ""), &linter.RuleConfig{}), "nil document should have no findings")
		})
	}
}

func TestRules_NilEntries_Success(t *testing.T) {
	t.Parallel()

	a := &arazzo.Arazzo{
		Arazzo:             "1.1.0",
		Info:               arazzo.Info{Description: pointerTo("described")},
		SourceDescriptions: arazzo.SourceDescriptions{nil},
		Workflows: arazzo.Workflows{nil, {
			WorkflowID:     "programmatic",
			Summary:        pointerTo("summary"),
			Steps:          arazzo.Steps{nil},
			SuccessActions: []*arazzo.ReusableSuccessAction{nil},
			FailureActions: []*arazzo.ReusableFailureAction{nil},
		}},
	}

	for _, rule := range allRules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, rule.Run(t.Context(), linter.NewDocumentInfo(a, ""), &linter.RuleConfig{}), "nil entries should be skipped")
		})
	}
}

func TestRules_SeverityOverride_Success(t *testing.T) {
	t.Parallel()

	severity := validation.SeverityError
	errs := (&rules.WorkflowDescriptionRule{}).Run(t.Context(), linter.NewDocumentInfo(parseDoc(t, minimalDoc), ""), &linter.RuleConfig{Severity: &severity})
	require.Len(t, errs, 1, "should report the workflow")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "should be a validation error")
	assert.Equal(t, validation.SeverityError, vErr.Severity, "configured severity should be used")
}

func pointerTo[T any](v T) *T {
	return &v
}

// minimalDoc is a document with a single undocumented workflow and step.
const minimalDoc = `arazzo: 1.1.0
info:
  title: Minimal
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    steps:
      - stepId: first
        operationId: getPet
`
