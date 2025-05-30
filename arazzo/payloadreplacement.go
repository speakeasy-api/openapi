package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/models"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/validation"
)

// PayloadReplacement represents a replacement of a value within a payload such as a request body.
type PayloadReplacement struct {
	models.Model[core.PayloadReplacement]

	// Target is a JSON pointer of XPath expression to the value to be replaced.
	Target jsonpointer.JSONPointer // TODO also support XPath
	// Value represents either the static value of the replacem	ent or an expression that will be evaluated to produce the value.
	Value ValueOrExpression
	// Extensions provides a list of extensions to the PayloadReplacement object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.PayloadReplacement] = (*PayloadReplacement)(nil)


// Validate will validate the payload replacement object against the Arazzo specification.
func (p *PayloadReplacement) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}

	if p.GetCore().Target.Present && p.Target == "" {
		errs = append(errs, validation.NewValueError("target is required", p.GetCore(), p.GetCore().Target))
	}

	if err := p.Target.Validate(); err != nil {
		errs = append(errs, validation.NewValueError(err.Error(), p.GetCore(), p.GetCore().Target))
	}

	if p.GetCore().Value.Present && p.Value == nil {
		errs = append(errs, validation.NewValueError("value is required", p.GetCore(), p.GetCore().Value))
	} else if p.Value != nil {
		_, expression, err := GetValueOrExpressionValue(p.Value)
		if err != nil {
			errs = append(errs, validation.NewValueError(err.Error(), p.GetCore(), p.GetCore().Value))
		}
		if expression != nil {
			if err := expression.Validate(true); err != nil {
				errs = append(errs, validation.NewValueError(err.Error(), p.GetCore(), p.GetCore().Value))
			}
		}
	}

	p.Valid = len(errs) == 0 && p.GetCore().GetValid()

	return errs
}
