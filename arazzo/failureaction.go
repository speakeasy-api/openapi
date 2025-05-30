package arazzo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/models"
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
	models.Model[core.FailureAction]

	// Name is the case sensitive name of the failure action.
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
			errors.New("An Arazzo object must be passed via validation options to validate a FailureAction"),
		}
	}

	errs := []error{}

	if f.GetCore().Name.Present && f.Name == "" {
		errs = append(errs, validation.NewValueError("name is required", f.GetCore(), f.GetCore().Name))
	}

	switch f.Type {
	case FailureActionTypeEnd:
		if f.WorkflowID != nil {
			errs = append(errs, validation.NewValueError("workflowId is not allowed when type: end is specified", f.GetCore(), f.GetCore().WorkflowID))
		}
		if f.StepID != nil {
			errs = append(errs, validation.NewValueError("stepId is not allowed when type: end is specified", f.GetCore(), f.GetCore().StepID))
		}
		if f.RetryAfter != nil {
			errs = append(errs, validation.NewValueError("retryAfter is not allowed when type: end is specified", f.GetCore(), f.GetCore().RetryAfter))
		}
		if f.RetryLimit != nil {
			errs = append(errs, validation.NewValueError("retryLimit is not allowed when type: end is specified", f.GetCore(), f.GetCore().RetryLimit))
		}
	case FailureActionTypeGoto:
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, validationActionWorkflowStepIDParams{
			parentType:       "failureAction",
			workflowID:       f.WorkflowID,
			workflowIDLine:   f.GetCore().WorkflowID.GetKeyNodeOrRoot(f.GetCore().RootNode).Line,
			workflowIDColumn: f.GetCore().WorkflowID.GetKeyNodeOrRoot(f.GetCore().RootNode).Column,
			stepID:           f.StepID,
			stepIDLine:       f.GetCore().StepID.GetKeyNodeOrRoot(f.GetCore().RootNode).Line,
			stepIDColumn:     f.GetCore().StepID.GetKeyNodeOrRoot(f.GetCore().RootNode).Column,
			arazzo:           a,
			workflow:         validation.GetContextObject[Workflow](o),
			required:         true,
		}, opts...)...)
		if f.RetryAfter != nil {
			errs = append(errs, validation.NewValueError("retryAfter is not allowed when type: goto is specified", f.GetCore(), f.GetCore().RetryAfter))
		}
		if f.RetryLimit != nil {
			errs = append(errs, validation.NewValueError("retryLimit is not allowed when type: goto is specified", f.GetCore(), f.GetCore().RetryLimit))
		}
	case FailureActionTypeRetry:
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, validationActionWorkflowStepIDParams{
			parentType:       "failureAction",
			workflowID:       f.WorkflowID,
			workflowIDLine:   f.GetCore().WorkflowID.GetKeyNodeOrRoot(f.GetCore().RootNode).Line,
			workflowIDColumn: f.GetCore().WorkflowID.GetKeyNodeOrRoot(f.GetCore().RootNode).Column,
			stepID:           f.StepID,
			stepIDLine:       f.GetCore().StepID.GetKeyNodeOrRoot(f.GetCore().RootNode).Line,
			stepIDColumn:     f.GetCore().StepID.GetKeyNodeOrRoot(f.GetCore().RootNode).Column,
			arazzo:           a,
			workflow:         validation.GetContextObject[Workflow](o),
			required:         false,
		}, opts...)...)
		if f.RetryAfter != nil {
			if *f.RetryAfter < 0 {
				errs = append(errs, validation.NewValueError("retryAfter must be greater than or equal to 0", f.GetCore(), f.GetCore().RetryAfter))
			}
		}
		if f.RetryLimit != nil {
			if *f.RetryLimit < 0 {
				errs = append(errs, validation.NewValueError("retryLimit must be greater than or equal to 0", f.GetCore(), f.GetCore().RetryLimit))
			}
		}
	default:
		errs = append(errs, validation.NewValueError(fmt.Sprintf("type must be one of [%s]", strings.Join([]string{string(FailureActionTypeEnd), string(FailureActionTypeGoto), string(FailureActionTypeRetry)}, ", ")), f.GetCore(), f.GetCore().Type))
	}

	for _, criterion := range f.Criteria {
		errs = append(errs, criterion.Validate(opts...)...)
	}

	f.Valid = len(errs) == 0 && f.GetCore().GetValid()

	return errs
}
