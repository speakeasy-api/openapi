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

	core core.FailureAction
}

var _ model[core.FailureAction] = (*FailureAction)(nil)

// GetCore will return the low level representation of the failure action object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (f *FailureAction) GetCore() *core.FailureAction {
	return &f.core
}

// Validate will validate the failure action object.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
// If a Workflow object is provided via validation options with validation.WithContextObject() then
// the FailureAction will be validated with the context of the workflow.
func (f FailureAction) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			errors.New("An Arazzo object must be passed via validation options to validate a FailureAction"),
		}
	}

	errs := []error{}

	if f.core.Name.Present && f.Name == "" {
		errs = append(errs, &validation.Error{
			Message: "name is required",
			Line:    f.core.Name.GetValueNodeOrRoot(f.core.RootNode).Line,
			Column:  f.core.Name.GetValueNodeOrRoot(f.core.RootNode).Column,
		})
	}

	switch f.Type {
	case FailureActionTypeEnd:
		if f.WorkflowID != nil {
			errs = append(errs, &validation.Error{
				Message: "workflowId is not allowed when type: end is specified",
				Line:    f.core.WorkflowID.GetKeyNodeOrRoot(f.core.RootNode).Line,
				Column:  f.core.WorkflowID.GetKeyNodeOrRoot(f.core.RootNode).Column,
			})
		}
		if f.StepID != nil {
			errs = append(errs, &validation.Error{
				Message: "stepId is not allowed when type: end is specified",
				Line:    f.core.StepID.GetKeyNodeOrRoot(f.core.RootNode).Line,
				Column:  f.core.StepID.GetKeyNodeOrRoot(f.core.RootNode).Column,
			})
		}
		if f.RetryAfter != nil {
			errs = append(errs, &validation.Error{
				Message: "retryAfter is not allowed when type: end is specified",
				Line:    f.core.RetryAfter.GetKeyNodeOrRoot(f.core.RootNode).Line,
				Column:  f.core.RetryAfter.GetKeyNodeOrRoot(f.core.RootNode).Column,
			})
		}
		if f.RetryLimit != nil {
			errs = append(errs, &validation.Error{
				Message: "retryLimit is not allowed when type: end is specified",
				Line:    f.core.RetryLimit.GetKeyNodeOrRoot(f.core.RootNode).Line,
				Column:  f.core.RetryLimit.GetKeyNodeOrRoot(f.core.RootNode).Column,
			})
		}
	case FailureActionTypeGoto:
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, validationActionWorkflowStepIDParams{
			parentType:       "failureAction",
			workflowID:       f.WorkflowID,
			workflowIDLine:   f.core.WorkflowID.GetKeyNodeOrRoot(f.core.RootNode).Line,
			workflowIDColumn: f.core.WorkflowID.GetKeyNodeOrRoot(f.core.RootNode).Column,
			stepID:           f.StepID,
			stepIDLine:       f.core.StepID.GetKeyNodeOrRoot(f.core.RootNode).Line,
			stepIDColumn:     f.core.StepID.GetKeyNodeOrRoot(f.core.RootNode).Column,
			arazzo:           a,
			workflow:         validation.GetContextObject[Workflow](o),
			required:         true,
		}, opts...)...)
		if f.RetryAfter != nil {
			errs = append(errs, &validation.Error{
				Message: "retryAfter is not allowed when type: goto is specified",
				Line:    f.core.RetryAfter.GetKeyNodeOrRoot(f.core.RootNode).Line,
				Column:  f.core.RetryAfter.GetKeyNodeOrRoot(f.core.RootNode).Column,
			})
		}
		if f.RetryLimit != nil {
			errs = append(errs, &validation.Error{
				Message: "retryLimit is not allowed when type: goto is specified",
				Line:    f.core.RetryLimit.GetKeyNodeOrRoot(f.core.RootNode).Line,
				Column:  f.core.RetryLimit.GetKeyNodeOrRoot(f.core.RootNode).Column,
			})
		}
	case FailureActionTypeRetry:
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, validationActionWorkflowStepIDParams{
			parentType:       "failureAction",
			workflowID:       f.WorkflowID,
			workflowIDLine:   f.core.WorkflowID.GetKeyNodeOrRoot(f.core.RootNode).Line,
			workflowIDColumn: f.core.WorkflowID.GetKeyNodeOrRoot(f.core.RootNode).Column,
			stepID:           f.StepID,
			stepIDLine:       f.core.StepID.GetKeyNodeOrRoot(f.core.RootNode).Line,
			stepIDColumn:     f.core.StepID.GetKeyNodeOrRoot(f.core.RootNode).Column,
			arazzo:           a,
			workflow:         validation.GetContextObject[Workflow](o),
			required:         false,
		}, opts...)...)
		if f.RetryAfter != nil {
			if *f.RetryAfter < 0 {
				errs = append(errs, &validation.Error{
					Message: "retryAfter must be greater than or equal to 0",
					Line:    f.core.RetryAfter.GetValueNodeOrRoot(f.core.RootNode).Line,
					Column:  f.core.RetryAfter.GetValueNodeOrRoot(f.core.RootNode).Column,
				})
			}
		}
		if f.RetryLimit != nil {
			if *f.RetryLimit < 0 {
				errs = append(errs, &validation.Error{
					Message: "retryLimit must be greater than or equal to 0",
					Line:    f.core.RetryLimit.GetValueNodeOrRoot(f.core.RootNode).Line,
					Column:  f.core.RetryLimit.GetValueNodeOrRoot(f.core.RootNode).Column,
				})
			}
		}
	default:
		errs = append(errs, &validation.Error{
			Message: fmt.Sprintf("type must be one of [%s]", strings.Join([]string{string(FailureActionTypeEnd), string(FailureActionTypeGoto), string(FailureActionTypeRetry)}, ", ")),
			Line:    f.core.Type.GetValueNodeOrRoot(f.core.RootNode).Line,
			Column:  f.core.Type.GetValueNodeOrRoot(f.core.RootNode).Column,
		})
	}

	for _, criterion := range f.Criteria {
		errs = append(errs, criterion.Validate(opts...)...)
	}

	return errs
}
