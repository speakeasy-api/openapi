package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/speakeasy-api/openapi/jsonschema/oas31"
)

const (
	// ErrTerminate is a sentinel error that can be returned from a Walk function to terminate the walk.
	ErrTerminate = errors.Error("terminate")
)

// Matcher is a struct that can be used to match specific nodes in the Arazzo document.
type Matcher struct {
	Arazzo                func(*Arazzo) error
	Info                  func(*Info) error
	SourceDescription     func(*SourceDescription) error
	Workflow              func(*Workflow) error
	ReusableParameter     func(*ReusableParameter) error
	JSONSchema            func(oas31.JSONSchema) error
	Step                  func(*Step) error
	ReusableSuccessAction func(*ReusableSuccessAction) error
	ReusableFailureAction func(*ReusableFailureAction) error
	Components            func(*Components) error
	Parameter             func(*Parameter) error
	SuccessAction         func(*SuccessAction) error
	FailureAction         func(*FailureAction) error
}

// MatchFunc represents a particular node in the Arazzo document that can be matched.
// Pass it a Matcher with the appropriate functions to populated to match the node type you are interested in.
type MatchFunc func(Matcher) error

// VisitFunc represents a function that will be called for each node in the Arazzo document.
// The functions receives the current node, any parent nodes, and the Arazzo document.
// TODO would this benefit from a locator type argument that contains the key or index it is located in within a slice or map?
type VisitFunc func(context.Context, MatchFunc, MatchFunc, *Arazzo) error

// Walk will walk the Arazzo document and call the provided VisitFunc for each node in the document.
func Walk(ctx context.Context, arazzo *Arazzo, visit VisitFunc) error {
	if arazzo == nil {
		return nil
	}

	if err := visit(ctx, getArazzoMatchFunc(arazzo), nil, arazzo); err != nil {
		if errors.Is(err, ErrTerminate) {
			return nil
		}
		return err
	}

	if err := visit(ctx, getInfoMatchFunc(&arazzo.Info), getArazzoMatchFunc(arazzo), arazzo); err != nil {
		if errors.Is(err, ErrTerminate) {
			return nil
		}
		return err
	}

	for _, sd := range arazzo.SourceDescriptions {
		if err := visit(ctx, getSourceDescriptionMatchFunc(sd), getArazzoMatchFunc(arazzo), arazzo); err != nil {
			if errors.Is(err, ErrTerminate) {
				return nil
			}
			return err
		}
	}

	for _, wf := range arazzo.Workflows {
		if err := walkWorkflow(ctx, wf, getArazzoMatchFunc(arazzo), arazzo, visit); err != nil {
			if errors.Is(err, ErrTerminate) {
				return nil
			}
			return err
		}
	}

	if err := walkComponents(ctx, arazzo.Components, getArazzoMatchFunc(arazzo), arazzo, visit); err != nil {
		if errors.Is(err, ErrTerminate) {
			return nil
		}
		return err
	}

	return nil
}

func walkWorkflow(ctx context.Context, workflow *Workflow, parent MatchFunc, arazzo *Arazzo, visit VisitFunc) error {
	if workflow == nil {
		return nil
	}

	if err := visit(ctx, getWorkflowMatchFunc(workflow), parent, arazzo); err != nil {
		return err
	}

	for _, parameter := range workflow.Parameters {
		if err := visit(ctx, getReusableParameterMatchFunc(parameter), getWorkflowMatchFunc(workflow), arazzo); err != nil {
			return err
		}
	}

	if err := visit(ctx, getJSONSchemaMatchFunc(workflow.Inputs), parent, arazzo); err != nil {
		return err
	}

	for _, step := range workflow.Steps {
		if err := walkStep(ctx, step, getWorkflowMatchFunc(workflow), arazzo, visit); err != nil {
			return err
		}
	}

	for _, successAction := range workflow.SuccessActions {
		if err := visit(ctx, getReusableSuccessActionMatchFunc(successAction), getWorkflowMatchFunc(workflow), arazzo); err != nil {
			return err
		}
	}

	for _, failureAction := range workflow.FailureActions {
		if err := visit(ctx, getReusableFailureActionMatchFunc(failureAction), getWorkflowMatchFunc(workflow), arazzo); err != nil {
			return err
		}
	}

	return nil
}

func walkStep(ctx context.Context, step *Step, parent MatchFunc, arazzo *Arazzo, visit VisitFunc) error {
	if step == nil {
		return nil
	}

	if err := visit(ctx, getStepMatchFunc(step), parent, arazzo); err != nil {
		return err
	}

	for _, parameter := range step.Parameters {
		if err := visit(ctx, getReusableParameterMatchFunc(parameter), getStepMatchFunc(step), arazzo); err != nil {
			return err
		}
	}

	for _, successAction := range step.OnSuccess {
		if err := visit(ctx, getReusableSuccessActionMatchFunc(successAction), getStepMatchFunc(step), arazzo); err != nil {
			return err
		}
	}

	for _, failureAction := range step.OnFailure {
		if err := visit(ctx, getReusableFailureActionMatchFunc(failureAction), getStepMatchFunc(step), arazzo); err != nil {
			return err
		}
	}

	return nil
}

func walkComponents(ctx context.Context, components *Components, parent MatchFunc, arazzo *Arazzo, visit VisitFunc) error {
	if components == nil {
		return nil
	}

	if err := visit(ctx, getComponentsMatchFunc(components), parent, arazzo); err != nil {
		return err
	}

	for _, inputs := range components.Inputs.All() {
		if err := visit(ctx, getJSONSchemaMatchFunc(inputs), getComponentsMatchFunc(components), arazzo); err != nil {
			return err
		}
	}

	for _, parameter := range components.Parameters.All() {
		if err := visit(ctx, getParameterMatchFunc(parameter), getComponentsMatchFunc(components), arazzo); err != nil {
			return err
		}
	}

	for _, successAction := range components.SuccessActions.All() {
		if err := visit(ctx, getSuccessActionMatchFunc(successAction), getComponentsMatchFunc(components), arazzo); err != nil {
			return err
		}
	}

	for _, failureAction := range components.FailureActions.All() {
		if err := visit(ctx, getFailureActionMatchFunc(failureAction), getComponentsMatchFunc(components), arazzo); err != nil {
			return err
		}
	}

	return nil
}

func getArazzoMatchFunc(Arazzo *Arazzo) MatchFunc {
	return func(m Matcher) error {
		if m.Arazzo != nil {
			return m.Arazzo(Arazzo)
		}
		return nil
	}
}

func getInfoMatchFunc(info *Info) MatchFunc {
	return func(m Matcher) error {
		if m.Info != nil {
			return m.Info(info)
		}
		return nil
	}
}

func getSourceDescriptionMatchFunc(sd *SourceDescription) MatchFunc {
	return func(m Matcher) error {
		if m.SourceDescription != nil {
			return m.SourceDescription(sd)
		}
		return nil
	}
}

func getWorkflowMatchFunc(workflow *Workflow) MatchFunc {
	return func(m Matcher) error {
		if m.Workflow != nil {
			return m.Workflow(workflow)
		}
		return nil
	}
}

func getReusableParameterMatchFunc(reusable *ReusableParameter) MatchFunc {
	return func(m Matcher) error {
		if m.ReusableParameter != nil {
			return m.ReusableParameter(reusable)
		}
		return nil
	}
}

func getJSONSchemaMatchFunc(jsonSchema oas31.JSONSchema) MatchFunc {
	return func(m Matcher) error {
		if m.JSONSchema != nil {
			return m.JSONSchema(jsonSchema)
		}
		return nil
	}
}

func getStepMatchFunc(step *Step) MatchFunc {
	return func(m Matcher) error {
		if m.Step != nil {
			return m.Step(step)
		}
		return nil
	}
}

func getReusableSuccessActionMatchFunc(successAction *ReusableSuccessAction) MatchFunc {
	return func(m Matcher) error {
		if m.ReusableSuccessAction != nil {
			return m.ReusableSuccessAction(successAction)
		}
		return nil
	}
}

func getReusableFailureActionMatchFunc(failureAction *ReusableFailureAction) MatchFunc {
	return func(m Matcher) error {
		if m.ReusableFailureAction != nil {
			return m.ReusableFailureAction(failureAction)
		}
		return nil
	}
}

func getComponentsMatchFunc(components *Components) MatchFunc {
	return func(m Matcher) error {
		if m.Components != nil {
			return m.Components(components)
		}
		return nil
	}
}

func getParameterMatchFunc(parameter *Parameter) MatchFunc {
	return func(m Matcher) error {
		if m.Parameter != nil {
			return m.Parameter(parameter)
		}
		return nil
	}
}

func getSuccessActionMatchFunc(successAction *SuccessAction) MatchFunc {
	return func(m Matcher) error {
		if m.SuccessAction != nil {
			return m.SuccessAction(successAction)
		}
		return nil
	}
}

func getFailureActionMatchFunc(failureAction *FailureAction) MatchFunc {
	return func(m Matcher) error {
		if m.FailureAction != nil {
			return m.FailureAction(failureAction)
		}
		return nil
	}
}
