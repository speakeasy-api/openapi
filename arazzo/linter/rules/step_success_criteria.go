package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleStepSuccessCriteria = "style-step-success-criteria"

type StepSuccessCriteriaRule struct{}

func (r *StepSuccessCriteriaRule) ID() string       { return RuleStyleStepSuccessCriteria }
func (r *StepSuccessCriteriaRule) Category() string { return CategoryStyle }
func (r *StepSuccessCriteriaRule) Description() string {
	return "Steps that call an operation or channel should declare successCriteria. Without them, any response is treated as success, so a workflow can carry on after an error response and fail later in a way that is harder to diagnose. Steps that run another workflow are not checked, as their success is decided by that workflow."
}
func (r *StepSuccessCriteriaRule) Summary() string {
	return "Operation and channel steps should declare successCriteria."
}
func (r *StepSuccessCriteriaRule) HowToFix() string {
	return "Add successCriteria to the step, for example `- condition: $statusCode == 200`."
}
func (r *StepSuccessCriteriaRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#style-step-success-criteria"
}
func (r *StepSuccessCriteriaRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *StepSuccessCriteriaRule) Versions() []string { return nil }

func (r *StepSuccessCriteriaRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	for _, w := range docInfo.Document.Workflows {
		if w == nil {
			continue
		}
		for _, s := range w.Steps {
			if s == nil || s.WorkflowID != nil || len(s.SuccessCriteria) > 0 {
				continue
			}
			core := s.GetCore()
			errs = append(errs, validation.NewValueError(config.GetSeverity(r.DefaultSeverity()), RuleStyleStepSuccessCriteria, fmt.Errorf("step `%s` in workflow `%s` has no successCriteria", s.StepID, w.WorkflowID), core, core.StepID))
		}
	}
	return errs
}
