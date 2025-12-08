package arazzo

import (
	"context"
	"errors"
	"regexp"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
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
	marshaller.Model[core.Workflow]

	// WorkflowID is a unique identifier for the workflow.
	WorkflowID string
	// Summary is a short description of the purpose of the workflow.
	Summary *string
	// Description is a longer description of the purpose of the workflow. May contain CommonMark syntax.
	Description *string
	// Parameters is a list of Parameters that will be passed to the referenced operation or workflow.
	Parameters []*ReusableParameter
	// Inputs is a JSON Schema containing a set of inputs that will be passed to the referenced workflow.
	Inputs *oas3.JSONSchema[oas3.Referenceable]
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
}

var _ interfaces.Model[core.Workflow] = (*Workflow)(nil)

var outputNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\.\-_]+$`)

// Validate will validate the workflow object against the Arazzo specification.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
func (w *Workflow) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			errors.New("an Arazzo object must be passed via validation options to validate a Workflow"),
		}
	}

	opts = append(opts, validation.WithContextObject(w))

	core := w.GetCore()
	errs := []error{}

	if core.WorkflowID.Present && w.WorkflowID == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("workflow.workflowId is required"), core, core.WorkflowID))
	}

	if w.Inputs != nil {
		errs = append(errs, w.Inputs.Validate(ctx, opts...)...)
	}

	for i, dependsOn := range w.DependsOn {
		if dependsOn.IsExpression() {
			if err := dependsOn.Validate(); err != nil {
				errs = append(errs, validation.NewSliceError(validation.NewValueValidationError("workflow.dependsOn expression is invalid: %s", err.Error()), core, core.DependsOn, i))
			}

			typ, sourceDescriptionName, _, _ := dependsOn.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, validation.NewSliceError(validation.NewValueValidationError("workflow.dependsOn must be a sourceDescriptions expression if not a workflowId, got %s", typ), core, core.DependsOn, i))
			}

			if a.SourceDescriptions.Find(sourceDescriptionName) == nil {
				errs = append(errs, validation.NewSliceError(validation.NewValueValidationError("workflow.dependsOn sourceDescription %s not found", sourceDescriptionName), core, core.DependsOn, i))
			}
		} else if a.Workflows.Find(string(dependsOn)) == nil {
			errs = append(errs, validation.NewSliceError(validation.NewValueValidationError("workflow.dependsOn workflowId %s not found", dependsOn), core, core.DependsOn, i))
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
			errs = append(errs, validation.NewMapKeyError(validation.NewValueValidationError("workflow.outputs name must be a valid name [%s]: %s", outputNameRegex.String(), name), core, core.Outputs, name))
		}

		if err := output.Validate(); err != nil {
			errs = append(errs, validation.NewMapValueError(validation.NewValueValidationError("workflow.outputs expression is invalid: %s", err.Error()), core, core.Outputs, name))
		}
	}

	for _, parameter := range w.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}

	w.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
