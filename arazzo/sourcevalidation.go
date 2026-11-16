package arazzo

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

var (
	operationPathRegex      = regexp.MustCompile(`^\{?\$sourceDescriptions\.([^.{}]+)\.url\}?#(.*)$`)
	remoteDependsOnRegex    = regexp.MustCompile(`^\$sourceDescriptions\.([^.]+)\.(.+)\.steps\.([^.]+)$`)
	parameterLocationsError = map[openapi.ParameterIn]bool{
		openapi.ParameterInPath:        true,
		openapi.ParameterInQuery:       true,
		openapi.ParameterInQueryString: true,
	}
)

// ValidateSources validates the Arazzo document against the documents loaded for its sourceDescriptions (see
// LoadSourceDescriptions). It checks that:
//
//   - operationId, operationPath and workflowId targets exist in the referenced source
//   - step parameters match a parameter of the target operation by name and location
//   - required operation parameters and required request bodies are supplied by the step
//   - parameters passed to a workflow in an Arazzo source match the workflow's inputs
//
// Sources that weren't loaded (unavailable, or AsyncAPI) are skipped. Problems that may still work at runtime, such
// as a header parameter the operation doesn't declare, are reported as warnings.
func (a *Arazzo) ValidateSources(ctx context.Context, sources *SourceDocuments) []error {
	if a == nil || sources == nil {
		return nil
	}

	errs := []error{}

	for _, w := range a.Workflows {
		if w == nil {
			continue
		}

		wCore := w.GetCore()
		for i, dependsOn := range w.DependsOn {
			if err := a.checkRemoteWorkflowExists(sources, dependsOn); err != nil {
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("workflow.dependsOn %w", err), wCore, wCore.DependsOn, i))
			}
		}

		for _, s := range w.Steps {
			if s == nil {
				continue
			}
			errs = append(errs, a.validateStepSources(sources, w, s)...)
		}

		errs = append(errs, a.validateSuccessActionsSources(sources, w.SuccessActions)...)
		errs = append(errs, a.validateFailureActionsSources(sources, w.FailureActions)...)
	}

	if a.Components != nil {
		for _, action := range a.Components.SuccessActions.All() {
			if action != nil {
				core := action.GetCore()
				errs = append(errs, a.validateRemoteWorkflowTarget(sources, "successAction", action.WorkflowID, action.Parameters, core, core.Parameters, core.WorkflowID)...)
			}
		}
		for _, action := range a.Components.FailureActions.All() {
			if action != nil {
				core := action.GetCore()
				errs = append(errs, a.validateRemoteWorkflowTarget(sources, "failureAction", action.WorkflowID, action.Parameters, core, core.Parameters, core.WorkflowID)...)
			}
		}
	}

	return errs
}

func (a *Arazzo) validateStepSources(sources *SourceDocuments, w *Workflow, s *Step) []error {
	core := s.GetCore()
	errs := []error{}

	if s.ChannelPath != nil {
		_, name, _, _ := s.ChannelPath.GetParts()
		// Sources declared as another type are reported by Validate, and AsyncAPI sources aren't loaded, so a loaded
		// source with an omitted type is a channelPath into a document that isn't AsyncAPI.
		doc := sources.Get(name)
		if sd := a.SourceDescriptions.Find(name); sd != nil && sd.Type == "" && doc != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.channelPath references sourceDescription `%s` which is an %s document, not an AsyncAPI document", name, doc.Type), core, core.ChannelPath))
		}
	}

	switch {
	case s.OperationID != nil:
		errs = append(errs, a.validateOperationStep(sources, w, s, core.OperationID)...)
	case s.OperationPath != nil:
		errs = append(errs, a.validateOperationStep(sources, w, s, core.OperationPath)...)
	case s.WorkflowID != nil:
		errs = append(errs, a.validateRemoteWorkflowTarget(sources, "step", s.WorkflowID, s.Parameters, core, core.Parameters, core.WorkflowID)...)
	}

	for i, dependsOn := range s.DependsOn {
		matches := remoteDependsOnRegex.FindStringSubmatch(dependsOn)
		if matches == nil {
			continue
		}
		doc := sources.Get(matches[1])
		if doc == nil {
			continue
		}
		target := remoteArazzoWorkflow(sources, matches[1], matches[2])
		switch {
		case doc.Arazzo == nil:
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn %w", notArazzoSourceError(doc)), core, core.DependsOn, i))
		case target == nil:
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn workflowId `%s` not found in sourceDescription `%s`", matches[2], matches[1]), core, core.DependsOn, i))
		case target.Steps.Find(matches[3]) == nil:
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn stepId `%s` not found in workflow `%s` of sourceDescription `%s`", matches[3], matches[2], matches[1]), core, core.DependsOn, i))
		}
	}

	errs = append(errs, a.validateSuccessActionsSources(sources, s.OnSuccess)...)
	errs = append(errs, a.validateFailureActionsSources(sources, s.OnFailure)...)

	return errs
}

func (a *Arazzo) validateSuccessActionsSources(sources *SourceDocuments, actions []*ReusableSuccessAction) []error {
	errs := []error{}
	for _, action := range actions {
		// Referenced component actions are checked once with the components.
		if action == nil || action.Object == nil {
			continue
		}
		core := action.Object.GetCore()
		errs = append(errs, a.validateRemoteWorkflowTarget(sources, "successAction", action.Object.WorkflowID, action.Object.Parameters, core, core.Parameters, core.WorkflowID)...)
	}
	return errs
}

func (a *Arazzo) validateFailureActionsSources(sources *SourceDocuments, actions []*ReusableFailureAction) []error {
	errs := []error{}
	for _, action := range actions {
		// Referenced component actions are checked once with the components.
		if action == nil || action.Object == nil {
			continue
		}
		core := action.Object.GetCore()
		errs = append(errs, a.validateRemoteWorkflowTarget(sources, "failureAction", action.Object.WorkflowID, action.Object.Parameters, core, core.Parameters, core.WorkflowID)...)
	}
	return errs
}

// checkRemoteWorkflowExists checks a `$sourceDescriptions.<name>.<workflowId>` expression references a workflow
// that exists in a loaded Arazzo source. Other values are not checked.
func (a *Arazzo) checkRemoteWorkflowExists(sources *SourceDocuments, workflowID expression.Expression) error {
	name, id, ok := remoteWorkflowReference(workflowID)
	if !ok {
		return nil
	}

	doc := sources.Get(name)
	switch {
	case doc == nil:
		return nil
	case doc.Arazzo == nil:
		return notArazzoSourceError(doc)
	case doc.Arazzo.Workflows.Find(id) == nil:
		return fmt.Errorf("references workflowId `%s` which doesn't exist in sourceDescription `%s`", id, name)
	}

	return nil
}

func notArazzoSourceError(doc *SourceDocument) error {
	return fmt.Errorf("references sourceDescription `%s` which is an %s document, not an Arazzo document", doc.Name, doc.Type)
}

// validateRemoteWorkflowTarget checks a workflowId targeting a workflow in an Arazzo source exists and that the
// parameters passed to it match its inputs.
func (a *Arazzo) validateRemoteWorkflowTarget(sources *SourceDocuments, parentName string, workflowID *expression.Expression, parameters []*ReusableParameter, core validation.CoreModeler, parametersNode validation.SliceNodeGetter, workflowIDNode validation.ValueNodeGetter) []error {
	if workflowID == nil {
		return nil
	}

	if err := a.checkRemoteWorkflowExists(sources, *workflowID); err != nil {
		return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("%s.workflowId %w", parentName, err), core, workflowIDNode)}
	}

	name, id, ok := remoteWorkflowReference(*workflowID)
	if !ok {
		return nil
	}

	target := remoteArazzoWorkflow(sources, name, id)
	if target == nil {
		return nil
	}

	return inputParameterValidationErrors(checkWorkflowInputParameters(parentName, a, parameters, target), core, parametersNode, workflowIDNode)
}

// remoteWorkflowReference parses a `$sourceDescriptions.<name>.<workflowId>` expression.
func remoteWorkflowReference(workflowID expression.Expression) (string, string, bool) {
	if !workflowID.IsExpression(expression.WithArazzo11Grammar()) {
		return "", "", false
	}
	typ, name, parts, _ := workflowID.GetParts()
	if typ != expression.ExpressionTypeSourceDescriptions || len(parts) == 0 {
		return "", "", false
	}
	return name, strings.Join(parts, "."), true
}

func remoteArazzoWorkflow(sources *SourceDocuments, name, workflowID string) *Workflow {
	doc := sources.Get(name)
	if doc == nil || doc.Arazzo == nil {
		return nil
	}
	return doc.Arazzo.Workflows.Find(workflowID)
}

// targetOperation is the OpenAPI operation targeted by a step.
type targetOperation struct {
	operation *openapi.Operation
	pathItem  *openapi.PathItem
	// label identifies the operation in messages.
	label string
}

func (a *Arazzo) validateOperationStep(sources *SourceDocuments, w *Workflow, s *Step, operationNode validation.ValueNodeGetter) []error {
	core := s.GetCore()

	target, err := a.findTargetOperation(sources, s)
	if err != nil {
		return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationOperationNotFound, err, core, operationNode)}
	}
	if target == nil {
		return nil
	}

	errs := []error{}

	opParams, complete := collectOperationParameters(target)

	supplied := map[string]bool{}
	for _, parameter := range w.Parameters {
		if p := a.resolveParameter(parameter); p != nil && p.In != nil {
			supplied[parameterKey(p.Name, openapi.ParameterIn(*p.In))] = true
		}
	}

	for i, parameter := range s.Parameters {
		p := a.resolveParameter(parameter)
		if p == nil || p.In == nil {
			continue
		}

		in := openapi.ParameterIn(*p.In)
		key := parameterKey(p.Name, in)
		supplied[key] = true

		if _, ok := opParams.parameters[key]; !ok && complete {
			severity := validation.SeverityWarning
			if in == openapi.ParameterInPath {
				severity = validation.SeverityError
			}
			errs = append(errs, validation.NewSliceError(severity, validation.RuleValidationOperationParameters, fmt.Errorf("step.parameters `%s` in `%s` is not a parameter of operation %s", p.Name, in, target.label), core, core.Parameters, i))
		}
	}

	for _, key := range opParams.keys {
		p := opParams.parameters[key]
		in := p.GetIn()
		if (!p.GetRequired() && in != openapi.ParameterInPath) || supplied[key] {
			continue
		}

		severity := validation.SeverityWarning
		if parameterLocationsError[in] {
			severity = validation.SeverityError
		}
		errs = append(errs, validation.NewValueError(severity, validation.RuleValidationOperationParameters, fmt.Errorf("step is missing required parameter `%s` in `%s` of operation %s", p.GetName(), in, target.label), core, operationNode))
	}

	if target.operation.GetRequestBody().GetObject().GetRequired() && s.RequestBody == nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationOperationParameters, fmt.Errorf("step.requestBody is required by operation %s", target.label), core, operationNode))
	}

	return errs
}

func (a *Arazzo) resolveParameter(parameter *ReusableParameter) *Parameter {
	if parameter == nil {
		return nil
	}
	return parameter.Get(a.Components)
}

// findTargetOperation finds the OpenAPI operation targeted by the step's operationId or operationPath. Returns nil
// without an error if the target source wasn't loaded or can't be determined (reported by structural validation).
func (a *Arazzo) findTargetOperation(sources *SourceDocuments, s *Step) (*targetOperation, error) {
	if s.OperationPath != nil {
		matches := operationPathRegex.FindStringSubmatch(s.OperationPath.String())
		if matches == nil {
			return nil, nil
		}
		doc := sources.Get(matches[1])
		if doc == nil {
			return nil, nil
		}
		if doc.OpenAPI == nil {
			return nil, fmt.Errorf("step.operationPath references sourceDescription `%s` which is an %s document, not an OpenAPI document", matches[1], doc.Type)
		}
		target := findOperationByPath(doc.OpenAPI, matches[2])
		if target == nil {
			return nil, fmt.Errorf("step.operationPath `%s` doesn't reference an operation in sourceDescription `%s`", s.OperationPath, matches[1])
		}
		return target, nil
	}

	sd := s.operationSourceDescription(a)
	if sd == nil {
		return nil, nil
	}
	doc := sources.Get(sd.Name)
	if doc == nil {
		return nil, nil
	}
	if doc.OpenAPI == nil {
		return nil, fmt.Errorf("step.operationId references sourceDescription `%s` which is an %s document, not an OpenAPI document", sd.Name, doc.Type)
	}

	operationID := s.OperationID.String()
	if s.OperationID.IsExpression(expression.WithArazzo11Grammar()) {
		_, _, parts, _ := s.OperationID.GetParts()
		operationID = strings.Join(parts, ".")
	}

	target := findOperationByID(doc.OpenAPI, operationID)
	if target == nil {
		return nil, fmt.Errorf("step.operationId `%s` not found in sourceDescription `%s`", operationID, sd.Name)
	}
	return target, nil
}

func findOperationByID(doc *openapi.OpenAPI, operationID string) *targetOperation {
	find := func(pathItem *openapi.PathItem) *targetOperation {
		if pathItem == nil {
			return nil
		}
		for method, operation := range pathItem.All() {
			if operation.GetOperationID() == operationID {
				return &targetOperation{operation: operation, pathItem: pathItem, label: fmt.Sprintf("`%s` (%s)", operationID, strings.ToUpper(string(method)))}
			}
		}
		for method, operation := range pathItem.GetAdditionalOperations().All() {
			if operation.GetOperationID() == operationID {
				return &targetOperation{operation: operation, pathItem: pathItem, label: fmt.Sprintf("`%s` (%s)", operationID, method)}
			}
		}
		return nil
	}

	if doc.Paths != nil {
		for _, pathItem := range doc.Paths.All() {
			if target := find(pathItem.GetObject()); target != nil {
				return target
			}
		}
	}
	for _, pathItem := range doc.Webhooks.All() {
		if target := find(pathItem.GetObject()); target != nil {
			return target
		}
	}

	return nil
}

// findOperationByPath finds the operation referenced by a JSON Pointer such as `/paths/~1pets~1{petId}/get`,
// `/webhooks/newPet/post` or `/paths/~1pets/additionalOperations/COPY`.
func findOperationByPath(doc *openapi.OpenAPI, pointer string) *targetOperation {
	if !strings.HasPrefix(pointer, "/") {
		return nil
	}

	tokens := strings.Split(pointer[1:], "/")
	for i, token := range tokens {
		tokens[i] = jsonpointer.UnescapeString(token)
	}

	if len(tokens) < 3 {
		return nil
	}

	var referencedPathItem *openapi.ReferencedPathItem
	switch tokens[0] {
	case "paths":
		if doc.Paths != nil {
			referencedPathItem, _ = doc.Paths.Get(tokens[1])
		}
	case "webhooks":
		referencedPathItem, _ = doc.Webhooks.Get(tokens[1])
	}

	pathItem := referencedPathItem.GetObject()
	if pathItem == nil {
		return nil
	}

	var operation *openapi.Operation
	switch {
	case len(tokens) == 3:
		operation = pathItem.GetOperation(openapi.HTTPMethod(strings.ToLower(tokens[2])))
	case len(tokens) == 4 && tokens[2] == "additionalOperations":
		operation, _ = pathItem.GetAdditionalOperations().Get(tokens[3])
	}

	if operation == nil {
		return nil
	}

	return &targetOperation{operation: operation, pathItem: pathItem, label: "`" + pointer + "`"}
}

// operationParameters are the parameters of an operation keyed by location and name, in declaration order.
type operationParameters struct {
	keys       []string
	parameters map[string]*openapi.Parameter
}

// collectOperationParameters returns the parameters of the operation, including those declared on its path item.
// complete is false if any parameter reference couldn't be resolved, in which case the parameters are partial.
func collectOperationParameters(target *targetOperation) (operationParameters, bool) {
	params := operationParameters{parameters: map[string]*openapi.Parameter{}}
	complete := true

	add := func(referenced []*openapi.ReferencedParameter) {
		for _, rp := range referenced {
			p := rp.GetObject()
			if p == nil {
				complete = false
				continue
			}
			key := parameterKey(p.Name, p.In)
			if _, ok := params.parameters[key]; !ok {
				params.keys = append(params.keys, key)
			}
			// Operation parameters override path item parameters with the same name and location.
			params.parameters[key] = p
		}
	}

	add(target.pathItem.GetParameters())
	add(target.operation.GetParameters())

	return params, complete
}

// parameterKey identifies a parameter by location and name. Header names are case-insensitive.
func parameterKey(name string, in openapi.ParameterIn) string {
	if in == openapi.ParameterInHeader {
		name = strings.ToLower(name)
	}
	return string(in) + ":" + name
}
