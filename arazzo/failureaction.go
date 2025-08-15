package arazzo

import (
	"context"
	"errors"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// FailureActionType represents the type of action to take on failure.
type FailureActionType string

const (
	// FailureActionTypeEnd indicates that the workflow/step should end.
	FailureActionTypeEnd FailureActionType = "end"
	// FailureActionTypeGoto indicates that the workflow/step should go to another workflow/step on failure.
	FailureActionTypeGoto FailureActionType = "goto"
	// FailureActionTypeRetry indicates that the workflow/step should retry on failure.
	FailureActionTypeRetry FailureActionType = "retry"
)

// FailureAction represents an action to take on failure of a workflow/step.
type FailureAction struct {
	marshaller.Model[core.FailureAction]

	// Name is the case-sensitive name of the failure action.
	Name string
	// Type is the type of action to take on failure.
	Type FailureActionType
	// WorkflowID is the workflow ID of the workflow/step to go to on failure. Mutually exclusive to StepID.
	WorkflowID *expression.Expression
	// StepID is the step ID of the workflow/step to go to on failure. Mutually exclusive to WorkflowID.
	StepID *string
	// RetryAfter is the number of seconds to wait before retrying the workflow/step on failure.
	RetryAfter *float64
	// RetryLimit is the number of times to retry the workflow/step on failure. If not set a single retry will be attempted if type is retry.
	RetryLimit *int
	// A list of assertions to check before taking the action.
	Criteria []criterion.Criterion
	// Extensions provides a list of extensions to the FailureAction object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.FailureAction] = (*FailureAction)(nil)

// Validate will validate the failure action object.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
// If a Workflow object is provided via validation options with validation.WithContextObject() then
// the FailureAction will be validated with the context of the workflow.
func (f *FailureAction) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			errors.New("an Arazzo object must be passed via validation options to validate a FailureAction"),
		}
	}

	core := f.GetCore()
	errs := []error{}

	if core.Name.Present && f.Name == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("failureAction field name is required"), core, core.Name))
	}

	switch f.Type {
	case FailureActionTypeEnd:
		if f.WorkflowID != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field workflowId is not allowed when type: end is specified"), core, core.WorkflowID))
		}
		if f.StepID != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field stepId is not allowed when type: end is specified"), core, core.StepID))
		}
		if f.RetryAfter != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field retryAfter is not allowed when type: end is specified"), core, core.RetryAfter))
		}
		if f.RetryLimit != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field retryLimit is not allowed when type: end is specified"), core, core.RetryLimit))
		}
	case FailureActionTypeGoto:
		workflowIDNode := core.WorkflowID.GetKeyNodeOrRoot(core.RootNode)
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, "failureAction", validationActionWorkflowStepIDParams{
			parentType:     "failureAction",
			workflowID:     f.WorkflowID,
			workflowIDNode: workflowIDNode,
			stepID:         f.StepID,
			stepIDLine:     core.StepID.GetKeyNodeOrRoot(core.RootNode).Line,
			stepIDColumn:   core.StepID.GetKeyNodeOrRoot(core.RootNode).Column,
			arazzo:         a,
			workflow:       validation.GetContextObject[Workflow](o),
			required:       true,
		}, opts...)...)
		if f.RetryAfter != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field retryAfter is not allowed when type: goto is specified"), core, core.RetryAfter))
		}
		if f.RetryLimit != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field retryLimit is not allowed when type: goto is specified"), core, core.RetryLimit))
		}
	case FailureActionTypeRetry:
		workflowIDNode := core.WorkflowID.GetKeyNodeOrRoot(core.RootNode)
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, "failureAction", validationActionWorkflowStepIDParams{
			parentType:     "failureAction",
			workflowID:     f.WorkflowID,
			workflowIDNode: workflowIDNode,
			stepID:         f.StepID,
			stepIDLine:     core.StepID.GetKeyNodeOrRoot(core.RootNode).Line,
			stepIDColumn:   core.StepID.GetKeyNodeOrRoot(core.RootNode).Column,
			arazzo:         a,
			workflow:       validation.GetContextObject[Workflow](o),
			required:       false,
		}, opts...)...)
		if f.RetryAfter != nil {
			if *f.RetryAfter < 0 {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field retryAfter must be greater than or equal to 0"), core, core.RetryAfter))
			}
		}
		if f.RetryLimit != nil {
			if *f.RetryLimit < 0 {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field retryLimit must be greater than or equal to 0"), core, core.RetryLimit))
			}
		}
	default:
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError("failureAction field type must be one of [%s]", strings.Join([]string{string(FailureActionTypeEnd), string(FailureActionTypeGoto), string(FailureActionTypeRetry)}, ", ")), core, core.Type))
	}

	for i := range f.Criteria {
		errs = append(errs, f.Criteria[i].Validate(opts...)...)
	}

	f.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
