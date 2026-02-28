package arazzo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	walkpkg "github.com/speakeasy-api/openapi/walk"
	"go.yaml.in/yaml/v4"
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
	marshaller.Model[core.SuccessAction]

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
}

var _ interfaces.Model[core.SuccessAction] = (*SuccessAction)(nil)

// Validate will validate the success action object against the Arazzo specification.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
func (s *SuccessAction) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			errors.New("an Arazzo object must be passed via validation options to validate a SuccessAction"),
		}
	}

	core := s.GetCore()
	errs := []error{}

	if core.Name.Present && s.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("successAction.name is required"), core, core.Name))
	}

	switch s.Type {
	case SuccessActionTypeEnd:
		if s.WorkflowID != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("successAction.workflowId is not allowed when type: end is specified"), core, core.WorkflowID))
		}
		if s.StepID != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("successAction.stepId is not allowed when type: end is specified"), core, core.StepID))
		}
	case SuccessActionTypeGoto:
		workflowIDNode := core.WorkflowID.GetKeyNodeOrRoot(core.RootNode)

		errs = append(errs, validationActionWorkflowIDAndStepID(ctx, "successAction", validationActionWorkflowStepIDParams{
			parentType:     "successAction",
			workflowID:     s.WorkflowID,
			workflowIDNode: workflowIDNode,
			stepID:         s.StepID,
			stepIDLine:     core.StepID.GetKeyNodeOrRoot(core.RootNode).Line,
			stepIDColumn:   core.StepID.GetKeyNodeOrRoot(core.RootNode).Column,
			arazzo:         a,
			workflow:       validation.GetContextObject[Workflow](o),
			required:       true,
		}, opts...)...)
	default:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("successAction.type must be one of [`%s`]", strings.Join([]string{string(SuccessActionTypeEnd), string(SuccessActionTypeGoto)}, ", ")), core, core.Type))
	}

	for i := range s.Criteria {
		errs = append(errs, s.Criteria[i].Validate(opts...)...)
	}

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

type validationActionWorkflowStepIDParams struct {
	parentType     string
	workflowID     *expression.Expression
	workflowIDNode *yaml.Node
	stepID         *string
	stepIDLine     int
	stepIDColumn   int
	arazzo         *Arazzo
	workflow       *Workflow
	required       bool
}

func validationActionWorkflowIDAndStepID(ctx context.Context, parentName string, params validationActionWorkflowStepIDParams, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	errs := []error{}

	if params.required && params.workflowID == nil && params.stepID == nil {
		errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("`%s`.workflowId or stepId is required", parentName), params.workflowIDNode))
	}
	if params.workflowID != nil && params.stepID != nil {
		errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, fmt.Errorf("`%s`.workflowId and stepId are mutually exclusive, only one can be specified", parentName), params.workflowIDNode))
	}
	if params.workflowID != nil {
		if params.workflowID.IsExpression() {
			if err := params.workflowID.Validate(); err != nil {
				errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("`%s`.workflowId expression is invalid: %w", parentName, err), params.workflowIDNode))
			}

			typ, sourceDescriptionName, _, _ := params.workflowID.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("`%s`.workflowId must be a sourceDescriptions expression, got `%s`", parentName, typ), params.workflowIDNode))
			}

			if params.arazzo.SourceDescriptions.Find(sourceDescriptionName) == nil {
				errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("`%s`.sourceDescription value `%s` not found", parentName, sourceDescriptionName), params.workflowIDNode))
			}
		} else if params.arazzo.Workflows.Find(pointer.Value(params.workflowID).String()) == nil {
			errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("`%s`.workflowId value `%s` does not exist", parentName, *params.workflowID), params.workflowIDNode))
		}
	}
	if params.stepID != nil {
		w := params.workflow
		if w == nil {
			key := validation.GetContextObject[componentKey](o)
			if key != nil {
				foundStepId := false

				for item := range Walk(ctx, params.arazzo) {
					// Check if we have a parent location context
					if len(item.Location) == 0 {
						continue
					}

					// Get the parent match function from the location
					parentLoc := item.Location[len(item.Location)-1]

					err := parentLoc.ParentMatchFunc(Matcher{
						Workflow: func(workflow *Workflow) error {
							return item.Match(Matcher{
								Step: func(step *Step) error {
									switch params.parentType {
									case "successAction":
										for _, onSuccess := range step.OnSuccess {
											if onSuccess.Reference == nil {
												continue
											}

											_, _, expressionParts, _ := onSuccess.Reference.GetParts()
											if len(expressionParts) > 0 && expressionParts[0] == key.name {
												if workflow.Steps.Find(pointer.Value(params.stepID)) != nil {
													foundStepId = true
													return walkpkg.ErrTerminate
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
												if workflow.Steps.Find(pointer.Value(params.stepID)) != nil {
													foundStepId = true
													return walkpkg.ErrTerminate
												}
											}
										}
									}
									return nil
								},
							})
						},
					})

					if err != nil && errors.Is(err, walkpkg.ErrTerminate) {
						break
					}
				}

				if !foundStepId {
					errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("`%s`.stepId value `%s` does not exist in any parent workflows", parentName, pointer.Value(params.stepID)), params.workflowIDNode))
				}
			}
		} else if w.Steps.Find(pointer.Value(params.stepID)) == nil {
			errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("`%s`.stepId value `%s` does not exist in workflow `%s`", parentName, pointer.Value(params.stepID), w.WorkflowID), params.workflowIDNode))
		}
	}

	return errs
}
