package arazzo

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/validation"
)

// Workflows provides a list of Workflow objects that describe the orchestration of API calls.
type Workflows []*Workflow

// Find will return the first workflow with the matching workflowId.
func (w Workflows) Find(id string) *Workflow {
	for _, workflow := range w {
		if workflow.WorkflowID == id {
			return workflow
		}
	}
	return nil
}

// Workflow represents a set of steps that orchestrates the execution of API calls.
type Workflow struct {
	// WorkflowID is a unique identifier for the workflow.
	WorkflowID string
	// Summary is a short description of the purpose of the workflow.
	Summary *string
	// Description is a longer description of the purpose of the workflow. May contain CommonMark syntax.
	Description *string
	// Parameters is a list of Parameters that will be passed to the referenced operation or workflow.
	Parameters []*ReusableParameter
	// Inputs is a JSON Schema containing a set of inputs that will be passed to the referenced workflow.
	Inputs oas31.JSONSchema
	// DependsOn is a list of workflowIds (or expressions to workflows) that must succeed before this workflow can be executed.
	DependsOn []expression.Expression
	// Steps is a list of steps that will be executed in the order they are listed.
	Steps Steps
	// SuccessActions is a list of actions that will be executed by each step in the workflow if the step succeeds. Can be overridden by the step.
	SuccessActions []*ReusableSuccessAction
	// FailureActions is a list of actions that will be executed by each step in the workflow if the step fails. Can be overridden by the step.
	FailureActions []*ReusableFailureAction
	// Outputs is a set of outputs that will be returned by the workflow.
	Outputs Outputs
	// Extensions provides a list of extensions to the Workflow object.
	Extensions *extensions.Extensions

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.Workflow
}

var _ model[core.Workflow] = (*Workflow)(nil)

// GetCore will return the low level representation of the workflow object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (w *Workflow) GetCore() *core.Workflow {
	return &w.core
}

var outputNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\.\-_]+$`)

// Validate will validate the workflow object against the Arazzo specification.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
func (w *Workflow) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			errors.New("An Arazzo object must be passed via validation options to validate a Workflow"),
		}
	}

	opts = append(opts, validation.WithContextObject(w))

	errs := []error{}

	if w.core.WorkflowID.Present && w.WorkflowID == "" {
		errs = append(errs, &validation.Error{
			Message: "workflowId is required",
			Line:    w.core.WorkflowID.GetValueNodeOrRoot(w.core.RootNode).Line,
			Column:  w.core.WorkflowID.GetValueNodeOrRoot(w.core.RootNode).Column,
		})
	}

	if w.Inputs != nil {
		inputsValNode := w.core.Inputs.GetValueNodeOrRoot(w.core.RootNode)
		errs = append(errs, validateJSONSchema(ctx, w.Inputs, inputsValNode.Line, inputsValNode.Column, opts...)...)
	}

	for i, dependsOn := range w.DependsOn {
		if err := dependsOn.Validate(false); err != nil {
			errs = append(errs, &validation.Error{
				Message: err.Error(),
				Line:    w.core.DependsOn.GetSliceValueNodeOrRoot(i, w.core.RootNode).Line,
				Column:  w.core.DependsOn.GetSliceValueNodeOrRoot(i, w.core.RootNode).Column,
			})
		}

		if dependsOn.IsExpression() {
			typ, sourceDescriptionName, _, _ := dependsOn.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("dependsOn must be a sourceDescriptions expression if not a workflowId, got %s", typ),
					Line:    w.core.DependsOn.GetValueNodeOrRoot(w.core.RootNode).Content[i].Line,
					Column:  w.core.DependsOn.GetValueNodeOrRoot(w.core.RootNode).Content[i].Column,
				})
			}

			if a.SourceDescriptions.Find(sourceDescriptionName) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("dependsOn sourceDescription %s not found", sourceDescriptionName),
					Line:    w.core.DependsOn.GetValueNodeOrRoot(w.core.RootNode).Content[i].Line,
					Column:  w.core.DependsOn.GetValueNodeOrRoot(w.core.RootNode).Content[i].Column,
				})
			}
		} else {
			if a.Workflows.Find(string(dependsOn)) == nil {
				errs = append(errs, &validation.Error{
					Message: fmt.Sprintf("dependsOn workflowId %s not found", dependsOn),
					Line:    w.core.DependsOn.GetValueNodeOrRoot(w.core.RootNode).Content[i].Line,
					Column:  w.core.DependsOn.GetValueNodeOrRoot(w.core.RootNode).Content[i].Column,
				})
			}
		}
	}

	for _, step := range w.Steps {
		errs = append(errs, step.Validate(ctx, opts...)...)
	}

	for _, successAction := range w.SuccessActions {
		errs = append(errs, successAction.Validate(ctx, opts...)...)
	}

	for _, failureAction := range w.FailureActions {
		errs = append(errs, failureAction.Validate(ctx, opts...)...)
	}

	for name, output := range w.Outputs.All() {
		if !outputNameRegex.MatchString(name) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("output name must be a valid name [%s]: %s", outputNameRegex.String(), name),
				Line:    w.core.Outputs.GetMapKeyNodeOrRoot(name, w.core.RootNode).Line,
				Column:  w.core.Outputs.GetMapKeyNodeOrRoot(name, w.core.RootNode).Column,
			})
		}

		if err := output.Validate(true); err != nil {
			errs = append(errs, &validation.Error{
				Message: err.Error(),
				Line:    w.core.Outputs.GetMapValueNodeOrRoot(name, w.core.RootNode).Line,
				Column:  w.core.Outputs.GetMapValueNodeOrRoot(name, w.core.RootNode).Column,
			})
		}
	}

	for _, parameter := range w.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}

	if len(errs) == 0 {
		w.Valid = true
	}

	return errs
}
