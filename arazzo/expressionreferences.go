package arazzo

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

// expressionScope is the kind of value being searched for runtime expressions, which determines the keys skipped.
type expressionScope int

const (
	// expressionScopeObject is an Arazzo object within a workflow, whose extensions are skipped.
	expressionScopeObject expressionScope = iota
	// expressionScopeWorkflow is a workflow object.
	expressionScopeWorkflow
	// expressionScopeStep is a step object.
	expressionScopeStep
	// expressionScopeLiteral is user data, such as a payload or parameter value, where no keys are skipped.
	expressionScopeLiteral
)

// expressionReferenceSkippedKeys are workflow and step fields whose values aren't runtime expressions evaluated
// against the workflow, or that are validated separately (dependsOn), so they aren't checked for references.
var expressionReferenceSkippedKeys = map[expressionScope]map[string]bool{
	expressionScopeWorkflow: {"inputs": true, "description": true, "summary": true, "dependsOn": true},
	expressionScopeStep:     {"description": true, "dependsOn": true},
}

// validateReferences checks references between objects within the document that runtime expressions and
// workflow targets make: `$steps`, `$workflows` and `$inputs` expressions must point at steps, outputs and inputs
// that exist, and parameters passed to a workflow in this document must match its inputs.
func (a *Arazzo) validateReferences() []error {
	errs := []error{}

	for _, w := range a.Workflows {
		if w == nil {
			continue
		}

		errs = append(errs, a.validateWorkflowExpressionReferences(w)...)

		for _, s := range w.Steps {
			if s == nil {
				continue
			}

			core := s.GetCore()
			if target := a.localWorkflow(s.WorkflowID); target != nil {
				errs = append(errs, inputParameterValidationErrors(checkWorkflowInputParameters("step", a, s.Parameters, target), core, core.Parameters, core.WorkflowID)...)
			}

			errs = append(errs, a.validateSuccessActionsInputParameters(s.OnSuccess)...)
			errs = append(errs, a.validateFailureActionsInputParameters(s.OnFailure)...)
		}

		errs = append(errs, a.validateSuccessActionsInputParameters(w.SuccessActions)...)
		errs = append(errs, a.validateFailureActionsInputParameters(w.FailureActions)...)
	}

	errs = append(errs, a.validateComponentActionsInputParameters()...)

	return errs
}

// localWorkflow returns the workflow in this document targeted by a workflowId, or nil if the workflowId is an
// expression referencing another document or the workflow doesn't exist (reported elsewhere).
func (a *Arazzo) localWorkflow(workflowID *expression.Expression) *Workflow {
	if workflowID == nil || workflowID.IsExpression(expression.WithArazzo11Grammar()) {
		return nil
	}
	return a.Workflows.Find(workflowID.String())
}

func (a *Arazzo) validateSuccessActionsInputParameters(actions []*ReusableSuccessAction) []error {
	errs := []error{}
	for _, action := range actions {
		// Referenced component actions are checked once with the components.
		if action == nil || action.Object == nil {
			continue
		}
		core := action.Object.GetCore()
		if target := a.localWorkflow(action.Object.WorkflowID); target != nil {
			errs = append(errs, inputParameterValidationErrors(checkWorkflowInputParameters("successAction", a, action.Object.Parameters, target), core, core.Parameters, core.WorkflowID)...)
		}
	}
	return errs
}

func (a *Arazzo) validateFailureActionsInputParameters(actions []*ReusableFailureAction) []error {
	errs := []error{}
	for _, action := range actions {
		// Referenced component actions are checked once with the components.
		if action == nil || action.Object == nil {
			continue
		}
		core := action.Object.GetCore()
		if target := a.localWorkflow(action.Object.WorkflowID); target != nil {
			errs = append(errs, inputParameterValidationErrors(checkWorkflowInputParameters("failureAction", a, action.Object.Parameters, target), core, core.Parameters, core.WorkflowID)...)
		}
	}
	return errs
}

func (a *Arazzo) validateComponentActionsInputParameters() []error {
	errs := []error{}
	if a.Components == nil {
		return errs
	}

	for _, action := range a.Components.SuccessActions.All() {
		if action == nil {
			continue
		}
		core := action.GetCore()
		if target := a.localWorkflow(action.WorkflowID); target != nil {
			errs = append(errs, inputParameterValidationErrors(checkWorkflowInputParameters("successAction", a, action.Parameters, target), core, core.Parameters, core.WorkflowID)...)
		}
	}
	for _, action := range a.Components.FailureActions.All() {
		if action == nil {
			continue
		}
		core := action.GetCore()
		if target := a.localWorkflow(action.WorkflowID); target != nil {
			errs = append(errs, inputParameterValidationErrors(checkWorkflowInputParameters("failureAction", a, action.Parameters, target), core, core.Parameters, core.WorkflowID)...)
		}
	}

	return errs
}

// inputParameterValidationErrors converts the result of checkWorkflowInputParameters into validation errors,
// reporting unknown parameters against the parameter and missing inputs against the workflowId.
func inputParameterValidationErrors(result inputParameterErrors, core validation.CoreModeler, parametersNode validation.SliceNodeGetter, workflowIDNode validation.ValueNodeGetter) []error {
	errs := []error{}
	for _, unknown := range result.unknown {
		errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, unknown.err, core, parametersNode, unknown.index))
	}
	for _, err := range result.missing {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, err, core, workflowIDNode))
	}
	return errs
}

// validateWorkflowExpressionReferences checks the `$steps`, `$workflows` and `$inputs` runtime expressions used
// anywhere within the workflow, including those embedded in payloads and criteria conditions.
func (a *Arazzo) validateWorkflowExpressionReferences(w *Workflow) []error {
	errs := []error{}

	visitExpressionScalars(w.GetRootNode(), expressionScopeWorkflow, map[*yaml.Node]bool{}, func(node *yaml.Node) {
		for _, e := range expression.ExtractExpressions(node.Value) {
			if err := a.checkExpressionReference(w, e); err != nil {
				errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidReference, err, node))
			}
		}
	})

	return errs
}

// visitExpressionScalars calls visit for every scalar value node within node that may contain runtime expressions.
func visitExpressionScalars(node *yaml.Node, scope expressionScope, seen map[*yaml.Node]bool, visit func(node *yaml.Node)) {
	if node == nil || seen[node] {
		return
	}
	seen[node] = true

	switch node.Kind {
	case yaml.ScalarNode:
		visit(node)
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if skipExpressionKey(scope, key) {
				continue
			}
			visitExpressionScalars(node.Content[i+1], childExpressionScope(scope, key), seen, visit)
		}
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			visitExpressionScalars(child, scope, seen, visit)
		}
	case yaml.AliasNode:
		visitExpressionScalars(node.Alias, scope, seen, visit)
	}
}

// skipExpressionKey returns true if the value of key isn't searched for runtime expressions within scope.
func skipExpressionKey(scope expressionScope, key string) bool {
	if scope == expressionScopeLiteral {
		return false
	}
	return strings.HasPrefix(key, "x-") || expressionReferenceSkippedKeys[scope][key]
}

// childExpressionScope returns the scope of the value of key within scope.
func childExpressionScope(scope expressionScope, key string) expressionScope {
	switch {
	case scope == expressionScopeLiteral, key == "payload", key == "value":
		return expressionScopeLiteral
	case scope == expressionScopeWorkflow && key == "steps":
		return expressionScopeStep
	default:
		return expressionScopeObject
	}
}

// checkExpressionReference checks that a `$steps`, `$workflows` or `$inputs` expression evaluated within the
// workflow references a step, workflow, output or input that exists. Other expressions are not checked here.
func (a *Arazzo) checkExpressionReference(w *Workflow, e expression.Expression) error {
	typ, reference, parts, _ := e.GetParts()

	switch typ {
	case expression.ExpressionTypeSteps:
		step := w.Steps.Find(reference)
		if step == nil {
			return fmt.Errorf("expression `%s` references step `%s` which doesn't exist in workflow `%s`", e, reference, w.WorkflowID)
		}
		if len(parts) > 1 && parts[0] == "outputs" && !hasOutput(step.Outputs, strings.Join(parts[1:], ".")) {
			return fmt.Errorf("expression `%s` references output `%s` which isn't defined by step `%s`", e, strings.Join(parts[1:], "."), reference)
		}
	case expression.ExpressionTypeWorkflows:
		target := a.Workflows.Find(reference)
		if target == nil {
			return fmt.Errorf("expression `%s` references workflow `%s` which doesn't exist", e, reference)
		}
		if len(parts) > 1 {
			name := strings.Join(parts[1:], ".")
			if parts[0] == "outputs" && !hasOutput(target.Outputs, name) {
				return fmt.Errorf("expression `%s` references output `%s` which isn't defined by workflow `%s`", e, name, reference)
			}
			if parts[0] == "inputs" {
				if inputs := getWorkflowInputs(target); inputs.known && !inputs.has(name) {
					return fmt.Errorf("expression `%s` references input `%s` which isn't defined by workflow `%s`", e, name, reference)
				}
			}
		}
	case expression.ExpressionTypeInputs:
		name := strings.Join(append([]string{reference}, parts...), ".")
		if inputs := getWorkflowInputs(w); inputs.known && !inputs.has(name) {
			return fmt.Errorf("expression `%s` references input `%s` which isn't defined by workflow `%s`", e, name, w.WorkflowID)
		}
	case expression.ExpressionTypeURL, expression.ExpressionTypeMethod, expression.ExpressionTypeStatusCode, expression.ExpressionTypeRequest,
		expression.ExpressionTypeResponse, expression.ExpressionTypeOutputs, expression.ExpressionTypeSourceDescriptions,
		expression.ExpressionTypeComponents, expression.ExpressionTypeMessage, expression.ExpressionTypeSelf:
		// Not references to objects within the workflow, checked elsewhere if at all.
	}

	return nil
}

// hasOutput returns true if the outputs define the named output, or a dotted prefix of it (for example
// `pet.name` referring to a property of the `pet` output).
func hasOutput(outputs Outputs, name string) bool {
	names := map[string]bool{}
	for key := range outputs.All() {
		names[key] = true
	}
	return hasDottedName(names, name)
}
