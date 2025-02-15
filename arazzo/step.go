package arazzo

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/validation"
)

// Steps represents a list of Step objects that describe the operations to be performed in the workflow.
type Steps []*Step

// Find will return the first Step object with the provided id.
func (s Steps) Find(id string) *Step {
	for _, step := range s {
		if step.StepID == id {
			return step
		}
	}
	return nil
}

// Step represents a step in a workflow that describes the operation to be performed.
type Step struct {
	// StepID is a unique identifier for the step within a workflow.
	StepID string
	// Description is a description of the step.
	Description *string
	// OperationID is an operationId or expression to an operation in a SourceDescription that the step relates to. Mutually exclusive with OperationPath & WorkflowID.
	OperationID *expression.Expression
	// OperationPath is an expression to an operation in a SourceDescription that the step relates to. Mutually exclusive with OperationID & WorkflowID.
	OperationPath *expression.Expression
	// WorkflowID is a workflowId or expression to a workflow in a SourceDescription that the step relates to. Mutually exclusive with OperationID & OperationPath.
	WorkflowID *expression.Expression
	// Parameters is a list of Parameters that will be passed to the referenced operation or workflow. These will override any matching parameters defined at the workflow level.
	Parameters []*ReusableParameter
	// RequestBody is the request body to be passed to the referenced operation.
	RequestBody *RequestBody
	// SuccessCriteria is a list of criteria that must be met for the step to be considered successful.
	SuccessCriteria []*criterion.Criterion
	// OnSuccess is a list of SuccessActions that will be executed if the step is successful.
	OnSuccess []*ReusableSuccessAction
	// OnFailure is a list of FailureActions that will be executed if the step is unsuccessful.
	OnFailure []*ReusableFailureAction
	// Outputs is a list of outputs that will be returned by the step.
	Outputs Outputs
	// Extensions provides a list of extensions to the Step object.
	Extensions *extensions.Extensions

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.Step
}

var _ model[core.Step] = (*Step)(nil)

// GetCore will return the low level representation of the step object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (s *Step) GetCore() *core.Step {
	return &s.core
}

var stepIDRegex = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

// Validate will validate the step object against the Arazzo specification.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
// Requires a Workflow object to be passed via validation options with validation.WithContextObject().
func (s *Step) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)
	w := validation.GetContextObject[Workflow](o)

	if a == nil {
		return []error{
			errors.New("An Arazzo object must be passed via validation options to validate a Step"),
		}
	}

	if w == nil {
		return []error{
			errors.New("A Workflow object must be passed via validation options to validate a Step"),
		}
	}

	opts = append(opts, validation.WithContextObject(s))

	errs := []error{}

	if s.core.StepID.Present && s.StepID == "" {
		errs = append(errs, &validation.Error{
			Message: "stepId is required",
			Line:    s.core.StepID.GetValueNodeOrRoot(s.core.RootNode).Line,
			Column:  s.core.StepID.GetValueNodeOrRoot(s.core.RootNode).Column,
		})
	} else if s.StepID != "" {
		if !stepIDRegex.MatchString(s.StepID) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("stepId must be a valid name [%s]: %s", stepIDRegex.String(), s.StepID),
				Line:    s.core.StepID.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.StepID.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}

		numStepsWithID := 0
		for _, step := range w.Steps {
			if step.StepID == s.StepID {
				numStepsWithID++
			}
		}
		if numStepsWithID > 1 {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("stepId must be unique within the workflow, found %d steps with the same stepId", numStepsWithID),
				Line:    s.core.StepID.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.StepID.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}
	}

	targetExpressions := []*expression.Expression{
		s.OperationID,
		s.OperationPath,
		s.WorkflowID,
	}

	numSet := 0
	for _, expression := range targetExpressions {
		if expression != nil {
			numSet++
		}
	}
	switch numSet {
	case 0:
		errs = append(errs, &validation.Error{
			Message: "at least one of operationId, operationPath or workflowId must be set",
			Line:    s.core.RootNode.Line,
			Column:  s.core.RootNode.Column,
		})
	case 1:
	default:
		errs = append(errs, &validation.Error{
			Message: "only one of operationId, operationPath or workflowId can be set",
			Line:    s.core.RootNode.Line,
			Column:  s.core.RootNode.Column,
		})
	}

	if s.OperationID != nil {
		numOpenAPISourceDescriptions := 0
		for _, sourceDescription := range a.SourceDescriptions {
			if sourceDescription.Type == SourceDescriptionTypeOpenAPI {
				numOpenAPISourceDescriptions++
			}
		}
		if numOpenAPISourceDescriptions > 1 && !s.OperationID.IsExpression() {
			errs = append(errs, &validation.Error{
				Message: "operationId must be a valid expression if there are multiple OpenAPI source descriptions",
				Line:    s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}
		if s.OperationID.IsExpression() {
			if err := s.OperationID.Validate(false); err != nil {
				errs = append(errs, &validation.Error{
					Message: err.Error(),
					Line:    s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}

			typ, sourceDescriptionName, _, _ := s.OperationID.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("operationId must be a sourceDescriptions expression, got %s", typ),
					Line:    s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}

			if a.SourceDescriptions.Find(string(sourceDescriptionName)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("sourceDescription %s not found", sourceDescriptionName),
					Line:    s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.OperationID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}
		}
	}

	if s.OperationPath != nil {
		if err := s.OperationPath.Validate(true); err != nil {
			errs = append(errs, &validation.Error{
				Message: err.Error(),
				Line:    s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}

		typ, sourceDescriptionName, expressionParts, jp := s.OperationPath.GetParts()

		if typ != expression.ExpressionTypeSourceDescriptions {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("operationPath must be a sourceDescriptions expression, got %s", typ),
				Line:    s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}

		if a.SourceDescriptions.Find(string(sourceDescriptionName)) == nil {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("sourceDescription %s not found", sourceDescriptionName),
				Line:    s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}

		if len(expressionParts) != 1 || expressionParts[0] != "url" {
			errs = append(errs, &validation.Error{
				Message: "operationPath must reference the url of a sourceDescription",
				Line:    s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}
		if jp == "" {
			errs = append(errs, &validation.Error{
				Message: "operationPath must contain a json pointer to the operation path within the sourceDescription",
				Line:    s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.OperationPath.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}
	}

	if s.WorkflowID != nil {
		if s.WorkflowID.IsExpression() {
			if err := s.WorkflowID.Validate(false); err != nil {
				errs = append(errs, &validation.Error{
					Message: err.Error(),
					Line:    s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}

			typ, sourceDescriptionName, _, _ := s.WorkflowID.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("workflowId must be a sourceDescriptions expression, got %s", typ),
					Line:    s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}

			if a.SourceDescriptions.Find(string(sourceDescriptionName)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("sourceDescription %s not found", sourceDescriptionName),
					Line:    s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}
		} else {
			if a.Workflows.Find(string(*s.WorkflowID)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("workflow %s not found", *s.WorkflowID),
					Line:    s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Line,
					Column:  s.core.WorkflowID.GetValueNodeOrRoot(s.core.RootNode).Column,
				})
			}
		}
	}

	parameterRefs := make(map[string]bool)
	parameters := make(map[string]bool)

	for i, parameter := range s.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)

		if parameter.Reference != nil {
			_, ok := parameterRefs[string(*parameter.Reference)]
			if ok {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("duplicate parameter found with reference %s", *parameter.Reference),
					Line:    s.core.Parameters.GetSliceValueNodeOrRoot(i, s.core.RootNode).Line,
					Column:  s.core.Parameters.GetSliceValueNodeOrRoot(i, s.core.RootNode).Column,
				})
			}
			parameterRefs[string(*parameter.Reference)] = true
		} else if parameter.Object != nil {
			id := fmt.Sprintf("%s.%v", parameter.Object.Name, parameter.Object.In)
			_, ok := parameters[id]
			if ok {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("duplicate parameter found with name %s and in %v", parameter.Object.Name, parameter.Object.In),
					Line:    s.core.Parameters.GetSliceValueNodeOrRoot(i, s.core.RootNode).Line,
					Column:  s.core.Parameters.GetSliceValueNodeOrRoot(i, s.core.RootNode).Column,
				})
			}
			parameters[id] = true
		}
	}

	if s.RequestBody != nil {
		if s.WorkflowID != nil {
			errs = append(errs, &validation.Error{
				Message: "requestBody should not be set when workflowId is set",
				Line:    s.core.RequestBody.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.RequestBody.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}

		errs = append(errs, s.RequestBody.Validate(ctx, opts...)...)
	}

	for _, successCriterion := range s.SuccessCriteria {
		errs = append(errs, successCriterion.Validate(opts...)...)
	}

	successActionRefs := make(map[string]bool)
	successActions := make(map[string]bool)

	for i, onSuccess := range s.OnSuccess {
		errs = append(errs, onSuccess.Validate(ctx, opts...)...)

		if onSuccess.Reference != nil {
			_, ok := successActionRefs[string(*onSuccess.Reference)]
			if ok {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("duplicate successAction found with reference %s", *onSuccess.Reference),
					Line:    s.core.OnSuccess.GetSliceValueNodeOrRoot(i, s.core.RootNode).Line,
					Column:  s.core.OnSuccess.GetSliceValueNodeOrRoot(i, s.core.RootNode).Column,
				})
			}
			successActionRefs[string(*onSuccess.Reference)] = true
		} else if onSuccess.Object != nil {
			id := fmt.Sprintf("%s.%v", onSuccess.Object.Name, onSuccess.Object.Type)
			_, ok := successActions[id]
			if ok {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("duplicate successAction found with name %s and type %v", onSuccess.Object.Name, onSuccess.Object.Type),
					Line:    s.core.OnSuccess.GetSliceValueNodeOrRoot(i, s.core.RootNode).Line,
					Column:  s.core.OnSuccess.GetSliceValueNodeOrRoot(i, s.core.RootNode).Column,
				})
			}
			successActions[id] = true
		}
	}

	failureActionRefs := make(map[string]bool)
	failureActions := make(map[string]bool)

	for i, onFailure := range s.OnFailure {
		errs = append(errs, onFailure.Validate(ctx, opts...)...)

		if onFailure.Reference != nil {
			_, ok := failureActionRefs[string(*onFailure.Reference)]
			if ok {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("duplicate failureAction found with reference %s", *onFailure.Reference),
					Line:    s.core.OnFailure.GetSliceValueNodeOrRoot(i, s.core.RootNode).Line,
					Column:  s.core.OnFailure.GetSliceValueNodeOrRoot(i, s.core.RootNode).Column,
				})
			}
			failureActionRefs[string(*onFailure.Reference)] = true
		} else if onFailure.Object != nil {
			id := fmt.Sprintf("%s.%v", onFailure.Object.Name, onFailure.Object.Type)
			_, ok := failureActions[id]
			if ok {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("duplicate failureAction found with name %s and type %v", onFailure.Object.Name, onFailure.Object.Type),
					Line:    s.core.OnFailure.GetSliceValueNodeOrRoot(i, s.core.RootNode).Line,
					Column:  s.core.OnFailure.GetSliceValueNodeOrRoot(i, s.core.RootNode).Column,
				})
			}
			failureActions[id] = true
		}
	}

	for name, output := range s.Outputs.All() {
		if !outputNameRegex.MatchString(name) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("output name must be a valid name [%s]: %s", outputNameRegex.String(), name),
				Line:    s.core.Outputs.GetMapKeyNodeOrRoot(name, s.core.RootNode).Line,
				Column:  s.core.Outputs.GetMapKeyNodeOrRoot(name, s.core.RootNode).Column,
			})
		}

		if err := output.Validate(true); err != nil {
			errs = append(errs, &validation.Error{
				Message: err.Error(),
				Line:    s.core.Outputs.GetMapValueNodeOrRoot(name, s.core.RootNode).Line,
				Column:  s.core.Outputs.GetMapValueNodeOrRoot(name, s.core.RootNode).Column,
			})
		}
	}

	if len(errs) == 0 {
		s.Valid = true
	}

	return errs
}
