package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/validation"
)

// PayloadReplacement represents a replacement of a value within a payload such as a request body.
type PayloadReplacement struct {
	// Target is a JSON pointer of XPath expression to the value to be replaced.
	Target jsonpointer.JSONPointer // TODO also support XPath
	// Value represents either the static value of the replacem	ent or an expression that will be evaluated to produce the value.
	Value ValueOrExpression
	// Extensions provides a list of extensions to the PayloadReplacement object.
	Extensions *extensions.Extensions

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.PayloadReplacement
}

var _ interfaces.Model[core.PayloadReplacement] = (*PayloadReplacement)(nil)

// GetCore will return the low level representation of the payload replacement object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (p *PayloadReplacement) GetCore() *core.PayloadReplacement {
	return &p.core
}

// Validate will validate the payload replacement object against the Arazzo specification.
func (p *PayloadReplacement) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}

	if p.core.Target.Present && p.Target == "" {
		errs = append(errs, &validation.Error{
			Message: "target is required",
			Line:    p.core.Target.GetValueNodeOrRoot(p.core.RootNode).Line,
			Column:  p.core.Target.GetValueNodeOrRoot(p.core.RootNode).Column,
		})
	}

	if err := p.Target.Validate(); err != nil {
		errs = append(errs, &validation.Error{
			Message: err.Error(),
			Line:    p.core.Target.GetValueNodeOrRoot(p.core.RootNode).Line,
			Column:  p.core.Target.GetValueNodeOrRoot(p.core.RootNode).Column,
		})
	}

	if p.core.Value.Present && p.Value == nil {
		errs = append(errs, &validation.Error{
			Message: "value is required",
			Line:    p.core.Value.GetValueNodeOrRoot(p.core.RootNode).Line,
			Column:  p.core.Value.GetValueNodeOrRoot(p.core.RootNode).Column,
		})
	} else if p.Value != nil {
		_, expression, err := GetValueOrExpressionValue(p.Value)
		if err != nil {
			errs = append(errs, &validation.Error{
				Message: err.Error(),
				Line:    p.core.Value.GetValueNodeOrRoot(p.core.RootNode).Line,
				Column:  p.core.Value.GetValueNodeOrRoot(p.core.RootNode).Column,
			})
		}
		if expression != nil {
			if err := expression.Validate(true); err != nil {
				errs = append(errs, &validation.Error{
					Message: err.Error(),
					Line:    p.core.Value.GetValueNodeOrRoot(p.core.RootNode).Line,
					Column:  p.core.Value.GetValueNodeOrRoot(p.core.RootNode).Column,
				})
			}
		}
	}

	if len(errs) == 0 {
		p.Valid = true
	}

	return errs
}
