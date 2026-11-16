package arazzo

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// In represents the location of a parameter.
type In string

const (
	// InPath indicates that the parameter is in the path of the request.
	InPath In = "path"
	// InQuery indicates that the parameter is in the query of the request.
	InQuery In = "query"
	// InHeader indicates that the parameter is in the header of the request.
	InHeader In = "header"
	// InCookie indicates that the parameter is in the cookie of the request.
	InCookie In = "cookie"
	// InQuerystring indicates that the parameter represents the entire query string of the request. Arazzo 1.1+ only.
	InQuerystring In = "querystring"
)

// actionParameterContext is passed as a validation context object when validating the parameters of a
// success or failure action, where parameters map to workflow inputs and `in` MUST NOT be used.
type actionParameterContext struct {
	parentName string
	// inline is true for parameters declared inline in the action, `in` is only rejected for inline parameters as
	// referenced component parameters may be shared with steps that require `in`.
	inline bool
}

// Parameter represents parameters that will be passed to a workflow or operation referenced by a step.
type Parameter struct {
	marshaller.Model[core.Parameter]

	// Name is the case-sensitive name of the parameter.
	Name string
	// In is the location of the parameter within an operation.
	In *In
	// Value represents either the static value of the parameter or an expression that will be evaluated to produce the value.
	Value expression.ValueOrExpression
	// Extensions provides a list of extensions to the Parameter object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Parameter] = (*Parameter)(nil)

// Validate will validate the parameter object against the Arazzo specification.
// If an Workflow or Step object is provided via validation options with validation.WithContextObject() then
// it will be validated in the context of that object.
func (p *Parameter) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := p.GetCore()
	errs := []error{}

	o := validation.NewOptions(opts...)

	s := validation.GetContextObject[Step](o)
	ap := validation.GetContextObject[actionParameterContext](o)
	is11 := specversion.IsAtLeast11(o)
	exprOpts := specversion.ExpressionOptions(o)

	if core.Name.Present && p.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter fieldname is required"), core, core.Name))
	}

	in := In("")
	if p.In != nil {
		in = *p.In
	}

	allowedIns := []string{string(InPath), string(InQuery), string(InHeader), string(InCookie)}
	if is11 {
		allowedIns = []string{string(InPath), string(InQuery), string(InQuerystring), string(InHeader), string(InCookie)}
	}

	switch {
	case ap != nil:
		if ap.inline && p.In != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, fmt.Errorf("parameter.in must not be used for %s parameters", ap.parentName), core, core.In))
		}
	case in == "":
		// Step parameters are validated with both the Step and its parent Workflow as context objects, while
		// workflow level parameters only have the Workflow, so a Step context means this is a step parameter.
		if s != nil && s.WorkflowID == nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.in is required within a step when workflowId is not set"), core, core.In))
		}
	case in == InQuerystring && !is11:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`parameter.in` `querystring`"), core, core.In))
	case !slices.Contains(allowedIns, string(in)):
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.in must be one of [`%s`] but was `%s`", strings.Join(allowedIns, ", "), in), core, core.In))
	}

	switch {
	case core.Value.Present && p.Value == nil:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`parameter.value` is required"), core, core.Value))
	case is11 && isSelectorNode(p.Value):
		errs = append(errs, validateSelectorNode(ctx, p.Value, opts...)...)
	case p.Value != nil:
		_, expression, err := expression.GetValueOrExpressionValue(p.Value, exprOpts...)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("%s", err.Error()), core, core.Value))
		}
		if expression != nil {
			if err := expression.Validate(exprOpts...); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("%s", err.Error()), core, core.Value))
			}
		}
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// validateParametersUnique reports duplicate parameters, a parameter is unique by the combination of its name
// and location (`in`), or by its reference for reusable parameters.
func validateParametersUnique(parentName string, parameters []*ReusableParameter, parentCore validation.CoreModeler, node validation.SliceNodeGetter) []error {
	errs := []error{}

	seenRefs := map[string]bool{}
	seen := map[string]bool{}

	for i, parameter := range parameters {
		switch {
		case parameter == nil:
		case parameter.Reference != nil:
			ref := string(*parameter.Reference)
			if seenRefs[ref] {
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("%s.parameters duplicate parameter found with reference `%s`", parentName, ref), parentCore, node, i))
			}
			seenRefs[ref] = true
		case parameter.Object != nil:
			in := ""
			if parameter.Object.In != nil {
				in = string(*parameter.Object.In)
			}
			id := parameter.Object.Name + "." + in
			if seen[id] {
				errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("%s.parameters duplicate parameter found with name `%s` and in `%s`", parentName, parameter.Object.Name, in), parentCore, node, i))
			}
			seen[id] = true
		}
	}

	return errs
}
