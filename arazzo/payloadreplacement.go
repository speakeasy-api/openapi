package arazzo

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// PayloadReplacement represents a replacement of a value within a payload such as a request body.
type PayloadReplacement struct {
	marshaller.Model[core.PayloadReplacement]

	// Target is a JSON pointer of XPath expression to the value to be replaced.
	Target jsonpointer.JSONPointer // TODO also support XPath
	// Value represents either the static value of the replacem	ent or an expression that will be evaluated to produce the value.
	Value expression.ValueOrExpression
	// Extensions provides a list of extensions to the PayloadReplacement object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.PayloadReplacement] = (*PayloadReplacement)(nil)

// Validate will validate the payload replacement object against the Arazzo specification.
func (p *PayloadReplacement) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := p.GetCore()
	errs := []error{}

	if core.Target.Present && p.Target == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("payloadReplacement.target is required"), core, core.Target))
	}

	if err := p.Target.Validate(); err != nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("payloadReplacement.target is invalid: %w", err), core, core.Target))
	}

	if core.Value.Present && p.Value == nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("payloadReplacement.value is required"), core, core.Value))
	} else if p.Value != nil {
		_, expression, err := expression.GetValueOrExpressionValue(p.Value)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("payloadReplacement.value is invalid: %w", err), core, core.Value))
		}
		if expression != nil {
			if err := expression.Validate(); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("payloadReplacement.value is invalid: %w", err), core, core.Value))
			}
		}
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
