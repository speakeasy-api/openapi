package cmdutil

import (
	"bytes"
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
)

type stubFix struct{}

func (stubFix) Description() string          { return "stub fix" }
func (stubFix) Interactive() bool            { return false }
func (stubFix) Prompts() []validation.Prompt { return nil }
func (stubFix) SetInput([]string) error      { return nil }
func (stubFix) Apply(any) error              { return nil }

func TestReportFixResults_Success(t *testing.T) {
	t.Parallel()

	vErr := &validation.Error{UnderlyingError: errors.New("issue"), Rule: "some-rule", Node: &yaml.Node{Line: 3, Column: 5}}
	result := &fix.Result{
		Applied: []fix.AppliedFix{{Error: vErr, Fix: stubFix{}, Before: "old", After: "new"}},
		Skipped: []fix.SkippedFix{
			{Error: vErr, Fix: stubFix{}, Reason: fix.SkipInteractive},
			{Error: vErr, Fix: stubFix{}, Reason: fix.SkipConflict},
			{Error: vErr, Fix: stubFix{}, Reason: fix.SkipUser},
			{Error: vErr, Fix: stubFix{}, Reason: fix.SkipReason(99)},
		},
		Failed: []fix.FailedFix{{Error: vErr, Fix: stubFix{}, FixError: errors.New("boom")}},
	}

	var buf bytes.Buffer
	ReportFixResults(&buf, result, true)

	expected := `
[dry-run] Fixed:
  [3:5] some-rule - stub fix
    old -> new

[dry-run] Skipped:
  [3:5] some-rule - stub fix (requires interactive input)
  [3:5] some-rule - stub fix (conflict with previous fix)
  [3:5] some-rule - stub fix (skipped by user)
  [3:5] some-rule - stub fix (unknown)

[dry-run] Failed:
  [3:5] some-rule - stub fix: boom
`
	assert.Equal(t, expected, buf.String(), "fix results should be reported")
}
