package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticUnreachableStep = "semantic-unreachable-step"

type UnreachableStepRule struct{}

func (r *UnreachableStepRule) ID() string       { return RuleSemanticUnreachableStep }
func (r *UnreachableStepRule) Category() string { return CategorySemantic }
func (r *UnreachableStepRule) Description() string {
	return "Every step should be reachable. Steps run in order unless a success or failure action ends the workflow or goes to another step, so a step is unreachable when every path to it is cut off by an unconditional `end` or `goto` and no action goes to it. Unreachable steps are usually left over from a refactor, or a sign that an action's criteria are missing."
}
func (r *UnreachableStepRule) Summary() string {
	return "Steps should be reachable from the start of the workflow."
}
func (r *UnreachableStepRule) HowToFix() string {
	return "Remove the step, add criteria to the action that skips it, or add a goto action that targets it."
}
func (r *UnreachableStepRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#semantic-unreachable-step"
}
func (r *UnreachableStepRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *UnreachableStepRule) Versions() []string { return nil }

func (r *UnreachableStepRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	a := docInfo.Document
	var errs []error
	for _, w := range a.Workflows {
		if w == nil {
			continue
		}
		reachable := reachableSteps(a, w)
		for i, s := range w.Steps {
			if s == nil || reachable[i] {
				continue
			}
			core := s.GetCore()
			errs = append(errs, validation.NewValueError(config.GetSeverity(r.DefaultSeverity()), RuleSemanticUnreachableStep, fmt.Errorf("step `%s` in workflow `%s` can never run", s.StepID, w.WorkflowID), core, core.StepID))
		}
	}
	return errs
}

// reachableSteps returns which of the workflow's steps can run, starting from the first step.
//
// After a step succeeds, its success actions are considered in order: a goto to a step makes that step reachable,
// and the first action without criteria always applies, so later actions and the next step are not reached through
// it. A step without such an action carries on to the next step. After a step fails, the workflow ends unless a
// failure action goes to a step.
func reachableSteps(a *arazzo.Arazzo, w *arazzo.Workflow) []bool {
	reachable := make([]bool, len(w.Steps))
	if len(w.Steps) == 0 {
		return reachable
	}

	index := map[string]int{}
	for i, s := range w.Steps {
		if s != nil {
			index[s.StepID] = i
		}
	}

	queue := []int{0}
	reachable[0] = true
	mark := func(stepID *string) {
		if stepID == nil {
			return
		}
		if j, ok := index[*stepID]; ok && !reachable[j] {
			reachable[j] = true
			queue = append(queue, j)
		}
	}

	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		s := w.Steps[i]

		next := i + 1
		continues := true
		for _, action := range successActions(a, w, s) {
			if action.Type == arazzo.SuccessActionTypeGoto {
				mark(action.StepID)
			}
			if len(action.Criteria) == 0 {
				continues = false
				break
			}
		}
		for _, action := range failureActions(a, w, s) {
			if action.Type == arazzo.FailureActionTypeGoto {
				mark(action.StepID)
			}
			if len(action.Criteria) == 0 {
				break
			}
		}

		if continues && next < len(w.Steps) && w.Steps[next] != nil {
			mark(&w.Steps[next].StepID)
		}
	}

	return reachable
}
