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
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// SuccessActionType represents the type of action to take on success.
type SuccessActionType string

const (
	// SuccessActionTypeEnd indicates that the workflow/step should end.
	SuccessActionTypeEnd SuccessActionType = "end"
	// SuccessActionTypeGoto indicates that the workflow/step should go to another workflow/step on success.
	SuccessActionTypeGoto SuccessActionType = "goto"
)

// SuccessAction represents an action to take on success of a workflow/step.
type SuccessAction struct {
	// Name is a case-sensitive name for the SuccessAction.
	Name string
	// Type is the type of action to take on success.
	Type SuccessActionType
	// WorkflowID is the workflow/step to go to on success. Mutually exclusive to StepID.
	WorkflowID *expression.Expression
	// StepID is the workflow/step to go to on success. Mutually exclusive to WorkflowID.
	StepID *string
	// Criteria is a list of criteria that must be met for the action to be taken.
	Criteria []criterion.Criterion
	// Extensions provides a list of extensions to the SuccessAction object.
	Extensions *extensions.Extensions

	marshaller.Model[core.SuccessAction]
}

var _ interfaces.Model[core.SuccessAction] = (*SuccessAction)(nil)

// Validate will validate the success action object against the Arazzo specification.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
func (s *SuccessAction) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			errors.New("An Arazzo object must be passed via validation options to validate a SuccessAction"),
		}
	}

	errs := []error{}

	if s.GetCore().Name.Present && s.Name == "" {
		errs = append(errs, validation.NewValueError("name is required", s.GetCore(), s.GetCore().Name))
	}

	switch s.Type {
	case SuccessActionTypeEnd:
		if s.WorkflowID != nil {
			errs = append(errs, validation.NewValueError("workflowId is not allowed when type: end is specified", s.GetCore(), s.GetCore().WorkflowID))
		}
		if s.StepID != nil {
			errs = append(errs, validation.NewValueError("stepId is not allowed when type: end is specified", s.GetCore(), s.GetCore().StepID))
		}
	case SuccessActionTypeGoto:
		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, validationActionWorkflowStepIDParams{
			parentType:       "successAction",
			workflowID:       s.WorkflowID,
			workflowIDLine:   s.GetCore().WorkflowID.GetKeyNodeOrRoot(s.GetCore().RootNode).Line,
			workflowIDColumn: s.GetCore().WorkflowID.GetKeyNodeOrRoot(s.GetCore().RootNode).Column,
			stepID:           s.StepID,
			stepIDLine:       s.GetCore().StepID.GetKeyNodeOrRoot(s.GetCore().RootNode).Line,
			stepIDColumn:     s.GetCore().StepID.GetKeyNodeOrRoot(s.GetCore().RootNode).Column,
			arazzo:           a,
			workflow:         validation.GetContextObject[Workflow](o),
			required:         true,
		}, opts...)...)
	default:
		errs = append(errs, validation.NewValueError(fmt.Sprintf("type must be one of [%s]", strings.Join([]string{string(SuccessActionTypeEnd), string(SuccessActionTypeGoto)}, ", ")), s.GetCore(), s.GetCore().Type))
	}

	for _, criterion := range s.Criteria {
		errs = append(errs, criterion.Validate(opts...)...)
	}

	s.Valid = len(errs) == 0 && s.GetCore().GetValid()

	return errs
}

type validationActionWorkflowStepIDParams struct {
	parentType       string
	workflowID       *expression.Expression
	workflowIDLine   int
	workflowIDColumn int
	stepID           *string
	stepIDLine       int
	stepIDColumn     int
	arazzo           *Arazzo
	workflow         *Workflow
	required         bool
}

func validationActionWorkflowIDAndStepID(ctx context.Context, params validationActionWorkflowStepIDParams, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	errs := []error{}

	if params.required && params.workflowID == nil && params.stepID == nil {
		errs = append(errs, &validation.Error{
			Message: "workflowId or stepId is required",
			Line:    params.workflowIDLine,
			Column:  params.workflowIDColumn,
		})
	}
	if params.workflowID != nil && params.stepID != nil {
		errs = append(errs, &validation.Error{
			Message: "workflowId and stepId are mutually exclusive, only one can be specified",
			Line:    params.workflowIDLine,
			Column:  params.workflowIDColumn,
		})
	}
	if params.workflowID != nil {
		if params.workflowID.IsExpression() {
			if err := params.workflowID.Validate(false); err != nil {
				errs = append(errs, &validation.Error{
					Message: err.Error(),
					Line:    params.workflowIDLine,
					Column:  params.workflowIDColumn,
				})
			}

			typ, sourceDescriptionName, _, _ := params.workflowID.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("workflowId must be a sourceDescriptions expression, got %s", typ),
					Line:    params.workflowIDLine,
					Column:  params.workflowIDColumn,
				})
			}

			if params.arazzo.SourceDescriptions.Find(string(sourceDescriptionName)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("sourceDescription %s not found", sourceDescriptionName),
					Line:    params.workflowIDLine,
					Column:  params.workflowIDColumn,
				})
			}
		} else {
			if params.arazzo.Workflows.Find(string(*params.workflowID)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("workflowId %s does not exist", *params.workflowID),
					Line:    params.workflowIDLine,
					Column:  params.workflowIDColumn,
				})
			}
		}
	}
	if params.stepID != nil {
		w := params.workflow
		if w == nil {
			key := validation.GetContextObject[componentKey](o)
			if key != nil {
				foundStepId := false

				_ = Walk(ctx, params.arazzo, func(ctx context.Context, node, parent MatchFunc, arazzo *Arazzo) error {
					if parent == nil {
						return nil
					}

					return parent(Matcher{
						Workflow: func(workflow *Workflow) error {
							return node(Matcher{
								Step: func(step *Step) error {
									switch params.parentType {
									case "successAction":
										for _, onSuccess := range step.OnSuccess {
											if onSuccess.Reference == nil {
												continue
											}

											_, _, expressionParts, _ := onSuccess.Reference.GetParts()
											if len(expressionParts) > 0 && expressionParts[0] == key.name {
												if workflow.Steps.Find(string(*params.stepID)) != nil {
													foundStepId = true
													return ErrTerminate
												}
											}
										}
									case "failureAction":
										for _, onFailure := range step.OnFailure {
											if onFailure.Reference == nil {
												continue
											}

											_, _, expressionParts, _ := onFailure.Reference.GetParts()
											if len(expressionParts) > 0 && expressionParts[0] == key.name {
												if workflow.Steps.Find(string(*params.stepID)) != nil {
													foundStepId = true
													return ErrTerminate
												}
											}
										}
									}
									return nil
								},
							})
						},
					})
				})

				if !foundStepId {
					errs = append(errs, &validation.Error{
						Message: fmt.Sprintf("stepId %s does not exist in any parent workflows", *params.stepID),
						Line:    params.stepIDLine,
						Column:  params.stepIDColumn,
					})
				}
			}
		} else {
			if w.Steps.Find(string(*params.stepID)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("stepId %s does not exist in workflow %s", *params.stepID, w.WorkflowID),
					Line:    params.stepIDLine,
					Column:  params.stepIDColumn,
				})
			}
		}
	}

	return errs
}
