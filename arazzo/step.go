package arazzo

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
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
	marshaller.Model[core.Step]

	// StepID is a unique identifier for the step within a workflow.
	StepID string
	// Description is a description of the step.
	Description *string
	// OperationID is an operationId or expression to an operation in a SourceDescription that the step relates to. Mutually exclusive with OperationPath & WorkflowID.
	OperationID *expression.Expression
	// OperationPath is an expression to an operation in a SourceDescription that the step relates to. Mutually exclusive with OperationID & WorkflowID.
	OperationPath *expression.Expression
	// WorkflowID is a workflowId or expression to a workflow in a SourceDescription that the step relates to. Mutually exclusive with OperationID, OperationPath & ChannelPath.
	WorkflowID *expression.Expression
	// ChannelPath is an expression to an AsyncAPI channel in a SourceDescription that the step relates to. Mutually exclusive with OperationID, OperationPath & WorkflowID. Arazzo 1.1+ only.
	ChannelPath *expression.Expression
	// Action indicates whether an AsyncAPI step sends or receives a message. Required for AsyncAPI steps. Arazzo 1.1+ only.
	Action *StepAction
	// CorrelationID is the correlation identifier used to match received messages. Only applicable to AsyncAPI steps with action `receive`. Arazzo 1.1+ only.
	CorrelationID expression.ValueOrExpression
	// Timeout is the maximum number of milliseconds to wait for the step to complete before failing the step. Arazzo 1.1+ only.
	Timeout *int
	// DependsOn is a list of steps that must complete before this step can be processed. Each value is a stepId in the current
	// workflow, a `$workflows.<workflowId>.steps.<stepId>` reference or a `$sourceDescriptions.<name>.<workflowId>.steps.<stepId>` reference. Arazzo 1.1+ only.
	DependsOn []string
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
}

var _ interfaces.Model[core.Step] = (*Step)(nil)

// StepAction indicates whether an AsyncAPI step sends or receives a message.
type StepAction string

const (
	// StepActionSend indicates the step sends (publishes) a message.
	StepActionSend StepAction = "send"
	// StepActionReceive indicates the step receives (subscribes to) a message.
	StepActionReceive StepAction = "receive"
)

var (
	stepIDRegex                     = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)
	dependsOnWorkflowStepRegex      = regexp.MustCompile(`^\$workflows\.([^.]+)\.steps\.([^.]+)$`)
	dependsOnSourceDescriptionRegex = regexp.MustCompile(`^\$sourceDescriptions\.([^.]+)\.([^.]+)\.steps\.([^.]+)$`)
)

// Validate will validate the step object against the Arazzo specification.
// Requires an Arazzo object to be passed via validation options with validation.WithContextObject().
// Requires a Workflow object to be passed via validation options with validation.WithContextObject().
func (s *Step) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)
	w := validation.GetContextObject[Workflow](o)

	if a == nil {
		return []error{
			errors.New("an Arazzo object must be passed via validation options to validate a Step"),
		}
	}

	if w == nil {
		return []error{
			errors.New("a Workflow object must be passed via validation options to validate a Step"),
		}
	}

	opts = append(opts, validation.WithContextObject(s))

	core := s.GetCore()
	errs := []error{}

	if core.StepID.Present && s.StepID == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("step.stepId is required"), core, core.StepID))
	} else if s.StepID != "" {
		if !stepIDRegex.MatchString(s.StepID) {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("step.stepId must be a valid name [`%s`]: `%s`", stepIDRegex.String(), s.StepID), core, core.StepID))
		}

		numStepsWithID := 0
		for _, step := range w.Steps {
			if step.StepID == s.StepID {
				numStepsWithID++
			}
		}
		if numStepsWithID > 1 {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("step.stepId must be unique within the workflow, found `%d` steps with the same stepId", numStepsWithID), core, core.StepID))
		}
	}

	is11 := specversion.IsAtLeast11(o)
	exprOpts := specversion.ExpressionOptions(o)

	errs = append(errs, s.validateVersionedFields(is11)...)

	targetExpressions := []*expression.Expression{
		s.OperationID,
		s.OperationPath,
		s.WorkflowID,
		s.ChannelPath,
	}
	targetFields := "operationId, operationPath or workflowId"
	if is11 {
		targetFields = "operationId, operationPath, channelPath or workflowId"
	}

	numSet := 0
	for _, expression := range targetExpressions {
		if expression != nil {
			numSet++
		}
	}
	switch numSet {
	case 0:
		errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("step at least one of %s fields must be set", targetFields), core.RootNode))
	case 1:
	default:
		errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, fmt.Errorf("step only one of %s can be set", targetFields), core.RootNode))
	}

	if s.OperationID != nil {
		numAPISourceDescriptions := 0
		for _, sourceDescription := range a.SourceDescriptions {
			if sourceDescription.Type == SourceDescriptionTypeOpenAPI || sourceDescription.Type == SourceDescriptionTypeAsyncAPI {
				numAPISourceDescriptions++
			}
		}
		if numAPISourceDescriptions > 1 && !s.OperationID.IsExpression(exprOpts...) {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, errors.New("step.operationId must be a valid expression if there are multiple OpenAPI or AsyncAPI source descriptions"), core, core.OperationID))
		}
		if s.OperationID.IsExpression(exprOpts...) {
			if err := s.OperationID.Validate(exprOpts...); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.operationId expression is invalid: %w", err), core, core.OperationID))
			}

			typ, sourceDescriptionName, _, _ := s.OperationID.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.operationId must be a sourceDescriptions expression, got `%s`", typ), core, core.OperationID))
			}

			if a.SourceDescriptions.Find(sourceDescriptionName) == nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.operationId referencing sourceDescription `%s` not found", sourceDescriptionName), core, core.OperationID))
			}
		}
	}

	if s.OperationPath != nil {
		errs = append(errs, validatePathTarget(a, "operationPath", "operation path", *s.OperationPath, core, core.OperationPath, exprOpts)...)
	}

	if s.ChannelPath != nil && is11 {
		errs = append(errs, validatePathTarget(a, "channelPath", "channel", *s.ChannelPath, core, core.ChannelPath, exprOpts)...)

		_, sourceDescriptionName, _, _ := s.ChannelPath.GetParts()
		if sd := a.SourceDescriptions.Find(sourceDescriptionName); sd != nil && sd.Type != SourceDescriptionTypeAsyncAPI {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.channelPath must reference an `asyncapi` sourceDescription, `%s` is of type `%s`", sourceDescriptionName, sd.Type), core, core.ChannelPath))
		}
	}

	if is11 {
		errs = append(errs, s.validateAsyncFields(a)...)
		errs = append(errs, s.validateDependsOn(a, w)...)
	}

	if s.WorkflowID != nil {
		if s.WorkflowID.IsExpression(exprOpts...) {
			if err := s.WorkflowID.Validate(exprOpts...); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.workflowId expression is invalid: %w", err), core, core.WorkflowID))
			}

			typ, sourceDescriptionName, _, _ := s.WorkflowID.GetParts()

			if typ != expression.ExpressionTypeSourceDescriptions {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.workflowId must be a sourceDescriptions expression, got `%s`", typ), core, core.WorkflowID))
			}

			if a.SourceDescriptions.Find((sourceDescriptionName)) == nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.workflowId referencing sourceDescription `%s` not found", sourceDescriptionName), core, core.WorkflowID))
			}
		} else if a.Workflows.Find(pointer.Value(s.WorkflowID).String()) == nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.workflowId referencing workflow `%s` not found", *s.WorkflowID), core, core.WorkflowID))
		}
	}

	for _, parameter := range s.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}
	errs = append(errs, validateParametersUnique("step", s.Parameters, core, core.Parameters)...)

	if s.RequestBody != nil {
		if s.WorkflowID != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("step.requestBody should not be set when workflowId is set"), core, core.RequestBody))
		}

		errs = append(errs, s.RequestBody.Validate(ctx, opts...)...)
	}

	if is11 && core.SuccessCriteria.Present && len(s.SuccessCriteria) == 0 {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("step.successCriteria must contain at least one criterion if set"), core, core.SuccessCriteria))
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
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("step.onSuccess duplicate successAction found with reference `%s`", *onSuccess.Reference), core, core.OnSuccess, i))
			}
			successActionRefs[string(*onSuccess.Reference)] = true
		} else if onSuccess.Object != nil {
			id := fmt.Sprintf("%s.%v", onSuccess.Object.Name, onSuccess.Object.Type)
			_, ok := successActions[id]
			if ok {
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("step.onSuccess duplicate successAction found with name `%s` and type `%v`", onSuccess.Object.Name, onSuccess.Object.Type), core, core.OnSuccess, i))
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
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("step.onFailure duplicate failureAction found with reference `%s`", *onFailure.Reference), core, core.OnFailure, i))
			}
			failureActionRefs[string(*onFailure.Reference)] = true
		} else if onFailure.Object != nil {
			id := fmt.Sprintf("%s.%v", onFailure.Object.Name, onFailure.Object.Type)
			_, ok := failureActions[id]
			if ok {
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("step.onFailure duplicate failureAction found with name `%s` and type `%v`", onFailure.Object.Name, onFailure.Object.Type), core, core.OnFailure, i))
			}
			failureActions[id] = true
		}
	}

	errs = append(errs, validateOutputs(ctx, "step", s.Outputs, core, core.Outputs, opts...)...)

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// validateVersionedFields reports Arazzo 1.1 only fields used in documents declaring an earlier version.
func (s *Step) validateVersionedFields(is11 bool) []error {
	if is11 {
		return nil
	}

	core := s.GetCore()
	errs := []error{}

	if core.ChannelPath.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`step.channelPath`"), core, core.ChannelPath))
	}
	if core.Action.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`step.action`"), core, core.Action))
	}
	if core.CorrelationID.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`step.correlationId`"), core, core.CorrelationID))
	}
	if core.Timeout.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`step.timeout`"), core, core.Timeout))
	}
	if core.DependsOn.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`step.dependsOn`"), core, core.DependsOn))
	}

	return errs
}

// validateAsyncFields validates the AsyncAPI specific step fields (action, correlationId) and timeout.
func (s *Step) validateAsyncFields(a *Arazzo) []error {
	core := s.GetCore()
	errs := []error{}

	asyncOperationSourceDescription := ""
	if s.OperationID != nil && s.OperationID.IsExpression(expression.WithArazzo11Grammar()) {
		_, sourceDescriptionName, _, _ := s.OperationID.GetParts()
		if sd := a.SourceDescriptions.Find(sourceDescriptionName); sd != nil && sd.Type == SourceDescriptionTypeAsyncAPI {
			asyncOperationSourceDescription = sourceDescriptionName
		}
	}

	switch {
	case s.Action != nil:
		switch *s.Action {
		case StepActionSend, StepActionReceive:
		default:
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("step.action must be one of [`%s`, `%s`] but was `%s`", StepActionSend, StepActionReceive, *s.Action), core, core.Action))
		}

		if s.OperationPath != nil || s.WorkflowID != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("step.action is only applicable to asyncapi steps using operationId or channelPath"), core, core.Action))
		}

		if s.OperationID != nil && s.OperationID.IsExpression(expression.WithArazzo11Grammar()) {
			_, sourceDescriptionName, _, _ := s.OperationID.GetParts()
			if sd := a.SourceDescriptions.Find(sourceDescriptionName); sd != nil && sd.Type != SourceDescriptionTypeAsyncAPI {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.action is only applicable to asyncapi steps, sourceDescription `%s` is of type `%s`", sourceDescriptionName, sd.Type), core, core.Action))
			}
		}
	case s.ChannelPath != nil:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("step.action is required when channelPath is set"), core, core.ChannelPath))
	case asyncOperationSourceDescription != "":
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("step.action is required for asyncapi steps, operationId references asyncapi sourceDescription `%s`", asyncOperationSourceDescription), core, core.OperationID))
	}

	if s.CorrelationID != nil {
		if s.Action == nil || *s.Action != StepActionReceive {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("step.correlationId is only applicable to asyncapi steps with action `receive`"), core, core.CorrelationID))
		}

		_, exp, err := expression.GetValueOrExpressionValue(s.CorrelationID, expression.WithArazzo11Grammar())
		if err == nil && exp != nil {
			err = exp.Validate(expression.WithArazzo11Grammar())
		}
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.correlationId is invalid: %w", err), core, core.CorrelationID))
		}
	}

	if s.Timeout != nil && *s.Timeout < 0 {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("step.timeout must be greater than or equal to 0"), core, core.Timeout))
	}

	return errs
}

// validateDependsOn validates the step level dependsOn references.
func (s *Step) validateDependsOn(a *Arazzo, w *Workflow) []error {
	core := s.GetCore()
	errs := []error{}

	if core.DependsOn.Present && len(s.DependsOn) == 0 {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("step.dependsOn must contain at least one entry if set"), core, core.DependsOn))
	}

	seen := map[string]bool{}

	for i, dependsOn := range s.DependsOn {
		if seen[dependsOn] {
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("step.dependsOn duplicate entry `%s`", dependsOn), core, core.DependsOn, i))
			continue
		}
		seen[dependsOn] = true

		workflowStepMatches := dependsOnWorkflowStepRegex.FindStringSubmatch(dependsOn)
		sourceDescriptionMatches := dependsOnSourceDescriptionRegex.FindStringSubmatch(dependsOn)

		switch {
		case len(workflowStepMatches) == 3:
			workflowID, stepID := workflowStepMatches[1], workflowStepMatches[2]
			workflow := a.Workflows.Find(workflowID)
			switch {
			case workflow == nil:
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn workflowId `%s` not found", workflowID), core, core.DependsOn, i))
			case workflow.Steps.Find(stepID) == nil:
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn stepId `%s` not found in workflow `%s`", stepID, workflowID), core, core.DependsOn, i))
			}
		case len(sourceDescriptionMatches) == 4:
			sourceDescriptionName := sourceDescriptionMatches[1]
			sd := a.SourceDescriptions.Find(sourceDescriptionName)
			switch {
			case sd == nil:
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn sourceDescription `%s` not found", sourceDescriptionName), core, core.DependsOn, i))
			case sd.Type != SourceDescriptionTypeArazzo:
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn sourceDescription `%s` must be of type `arazzo`", sourceDescriptionName), core, core.DependsOn, i))
			}
		case strings.HasPrefix(dependsOn, "$"):
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.dependsOn must be a stepId, `$workflows.<workflowId>.steps.<stepId>` or `$sourceDescriptions.<name>.<workflowId>.steps.<stepId>`, got `%s`", dependsOn), core, core.DependsOn, i))
		case dependsOn == s.StepID:
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn must not reference the step itself `%s`", dependsOn), core, core.DependsOn, i))
		case w.Steps.Find(dependsOn) == nil:
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.dependsOn stepId `%s` not found in workflow `%s`", dependsOn, w.WorkflowID), core, core.DependsOn, i))
		}
	}

	return errs
}

// validatePathTarget validates an operationPath or channelPath expression, which must be a
// `{$sourceDescriptions.<name>.url}#/json/pointer` expression referencing a known sourceDescription.
func validatePathTarget(a *Arazzo, field, targetName string, target expression.Expression, core validation.CoreModeler, node validation.ValueNodeGetter, exprOpts []expression.ValidateOption) []error {
	errs := []error{}

	if err := target.Validate(exprOpts...); err != nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.%s expression is invalid: %w", field, err), core, node))
	}

	typ, sourceDescriptionName, expressionParts, jp := target.GetParts()

	if typ != expression.ExpressionTypeSourceDescriptions {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.%s must be a sourceDescriptions expression, got `%s`", field, typ), core, node))
	}

	if a.SourceDescriptions.Find(sourceDescriptionName) == nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.%s referencing sourceDescription `%s` not found", field, sourceDescriptionName), core, node))
	}

	if len(expressionParts) != 1 || expressionParts[0] != "url" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("step.%s must reference the url of a sourceDescription", field), core, node))
	}
	if jp == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("step.%s must contain a json pointer to the %s within the sourceDescription", field, targetName), core, node))
	}

	return errs
}
