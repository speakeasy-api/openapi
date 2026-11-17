package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

const RuleSemanticUnusedStepOutput = "semantic-unused-step-output"

// stepOutputsExpressionRegex matches `$steps.<stepId>.outputs` expressions, capturing the step and the (possibly
// dotted) output name if there is one.
var stepOutputsExpressionRegex = regexp.MustCompile(`\$steps\.([A-Za-z0-9_\-]+)\.outputs(?:\.([A-Za-z0-9_.\-]+))?`)

type UnusedStepOutputRule struct{}

func (r *UnusedStepOutputRule) ID() string       { return RuleSemanticUnusedStepOutput }
func (r *UnusedStepOutputRule) Category() string { return CategorySemantic }
func (r *UnusedStepOutputRule) Description() string {
	return "Step outputs should be used. A step's outputs can only be read within its workflow, by later steps or by the workflow's outputs, so an output that is never read has no effect."
}
func (r *UnusedStepOutputRule) Summary() string {
	return "Step outputs should be read with a $steps expression."
}
func (r *UnusedStepOutputRule) HowToFix() string {
	return "Remove the output, or use it with a `$steps.<stepId>.outputs.<name>` expression."
}
func (r *UnusedStepOutputRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#semantic-unused-step-output"
}
func (r *UnusedStepOutputRule) DefaultSeverity() validation.Severity { return validation.SeverityHint }
func (r *UnusedStepOutputRule) Versions() []string                   { return nil }

func (r *UnusedStepOutputRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	severity := config.GetSeverity(r.DefaultSeverity())

	var errs []error
	for _, w := range docInfo.Document.Workflows {
		if w == nil {
			continue
		}

		used, all := usedStepOutputs(docInfo.Document, w)
		for _, s := range w.Steps {
			if s == nil || all[s.StepID] {
				continue
			}
			core := s.GetCore()
			for name := range s.Outputs.Keys() {
				if used[s.StepID][name] {
					continue
				}
				errs = append(errs, validation.NewMapKeyError(severity, RuleSemanticUnusedStepOutput, fmt.Errorf("output `%s` of step `%s` is never used", name, s.StepID), core, core.Outputs, name))
			}
		}
	}
	return errs
}

// usedStepOutputs returns the step outputs read with `$steps` expressions within the workflow, keyed by stepId.
// all records the steps whose outputs are used as a whole. Expressions in reusable components count too, as the
// components can be referenced from the workflow.
func usedStepOutputs(a *arazzo.Arazzo, w *arazzo.Workflow) (map[string]map[string]bool, map[string]bool) {
	used := map[string]map[string]bool{}
	all := map[string]bool{}

	visit := func(scalar *yaml.Node) {
		for _, m := range stepOutputsExpressionRegex.FindAllStringSubmatch(scalar.Value, -1) {
			if m[2] == "" {
				all[m[1]] = true
				continue
			}
			if used[m[1]] == nil {
				used[m[1]] = map[string]bool{}
			}
			markDottedName(used[m[1]], m[2])
		}
	}
	visitScalars(w.GetRootNode(), visit)
	for _, node := range reusableComponentNodes(a) {
		visitScalars(node, visit)
	}

	return used, all
}
