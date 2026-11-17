package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleWorkflowDescription = "style-workflow-description"

type WorkflowDescriptionRule struct{}

func (r *WorkflowDescriptionRule) ID() string       { return RuleStyleWorkflowDescription }
func (r *WorkflowDescriptionRule) Category() string { return CategoryStyle }
func (r *WorkflowDescriptionRule) Description() string {
	return "Workflows should have a summary or description that explains the outcome they achieve. Workflow IDs are often terse, and a short explanation helps people and tools pick the right workflow."
}
func (r *WorkflowDescriptionRule) Summary() string {
	return "Workflows should have a summary or description."
}
func (r *WorkflowDescriptionRule) HowToFix() string {
	return "Add a summary or description to the workflow."
}
func (r *WorkflowDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#style-workflow-description"
}
func (r *WorkflowDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *WorkflowDescriptionRule) Versions() []string { return nil }

func (r *WorkflowDescriptionRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	for _, w := range docInfo.Document.Workflows {
		if w == nil || hasText(w.Summary) || hasText(w.Description) {
			continue
		}
		core := w.GetCore()
		errs = append(errs, validation.NewValueError(config.GetSeverity(r.DefaultSeverity()), RuleStyleWorkflowDescription, fmt.Errorf("workflow `%s` has no summary or description", w.WorkflowID), core, core.WorkflowID))
	}
	return errs
}
