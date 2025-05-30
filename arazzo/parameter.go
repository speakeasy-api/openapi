package arazzo

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
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
)

// Parameter represents parameters that will be passed to a workflow or operation referenced by a step.
type Parameter struct {
	marshaller.Model[core.Parameter]

	// Name is the case sensitive name of the parameter.
	Name string
	// In is the location of the parameter within an operation.
	In *In
	// Value represents either the static value of the parameter or an expression that will be evaluated to produce the value.
	Value ValueOrExpression
	// Extensions provides a list of extensions to the Parameter object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Parameter] = (*Parameter)(nil)

// Validate will validate the parameter object against the Arazzo specification.
// If an Workflow or Step object is provided via validation options with validation.WithContextObject() then
// it will be validated in the context of that object.
func (p *Parameter) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := p.GetCore()
	errs := core.GetValidationErrors()

	o := validation.NewOptions(opts...)

	w := validation.GetContextObject[Workflow](o)
	s := validation.GetContextObject[Step](o)

	if core.Name.Present && p.Name == "" {
		errs = append(errs, validation.NewValueError("name is required", core, core.Name))
	}

	in := In("")
	if p.In != nil {
		in = *p.In
	}

	switch in {
	case InPath:
	case InQuery:
	case InHeader:
	case InCookie:
	default:
		if p.In == nil || in == "" {
			if w == nil && s != nil && s.WorkflowID == nil {
				errs = append(errs, validation.NewValueError("in is required within a step when workflowId is not set", core, core.In))
			}
		}

		if in != "" {
			errs = append(errs, validation.NewValueError(fmt.Sprintf("in must be one of [%s] but was %s", strings.Join([]string{string(InPath), string(InQuery), string(InHeader), string(InCookie)}, ", "), in), core, core.In))
		}
	}

	if core.Value.Present && p.Value == nil {
		errs = append(errs, validation.NewValueError("value is required", core, core.Value))
	} else if p.Value != nil {
		_, expression, err := GetValueOrExpressionValue(p.Value)
		if err != nil {
			errs = append(errs, validation.NewValueError(err.Error(), core, core.Value))
		}
		if expression != nil {
			if err := expression.Validate(true); err != nil {
				errs = append(errs, validation.NewValueError(err.Error(), core, core.Value))
			}
		}
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
