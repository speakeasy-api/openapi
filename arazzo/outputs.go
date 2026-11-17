package arazzo

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// Outputs is a map of friendly name to output values that extract data from the workflows/steps.
type Outputs = *sequencedmap.Map[string, *OutputValue]

// OutputValue is the value of a workflow or step output. It is either a runtime expression or,
// in Arazzo 1.1+, a Selector Object.
type OutputValue struct {
	values.EitherValue[Selector, core.Selector, expression.Expression, string]
}

// NewOutputValueFromExpression creates an output value from a runtime expression.
func NewOutputValueFromExpression(e expression.Expression) *OutputValue {
	return &OutputValue{
		EitherValue: values.EitherValue[Selector, core.Selector, expression.Expression, string]{
			Right: pointer.From(e),
		},
	}
}

// NewOutputValueFromSelector creates an output value from a Selector Object. Arazzo 1.1+ only.
func NewOutputValueFromSelector(s *Selector) *OutputValue {
	return &OutputValue{
		EitherValue: values.EitherValue[Selector, core.Selector, expression.Expression, string]{
			Left: s,
		},
	}
}

// IsExpression returns true if the output value is a runtime expression.
func (o *OutputValue) IsExpression() bool {
	return o != nil && o.IsRight()
}

// GetExpression returns the runtime expression, or nil if the output value is a Selector Object.
func (o *OutputValue) GetExpression() *expression.Expression {
	if o == nil {
		return nil
	}
	return o.GetRight()
}

// IsSelector returns true if the output value is a Selector Object.
func (o *OutputValue) IsSelector() bool {
	return o != nil && o.IsLeft()
}

// GetSelector returns the Selector Object, or nil if the output value is a runtime expression.
func (o *OutputValue) GetSelector() *Selector {
	if o == nil {
		return nil
	}
	return o.GetLeft()
}

type outputsNodeGetter interface {
	validation.MapKeyNodeGetter
	validation.MapValueNodeGetter
}

// validateOutputs validates a set of workflow or step outputs, reporting errors against the parent's outputs field.
func validateOutputs(ctx context.Context, parentName string, outputs Outputs, parent validation.CoreModeler, outputsField outputsNodeGetter, opts ...validation.Option) []error {
	errs := []error{}

	exprOpts := specversion.ExpressionOptions(validation.NewOptions(opts...))

	for name, output := range outputs.All() {
		if !outputNameRegex.MatchString(name) {
			errs = append(errs, validation.NewMapKeyError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("%s.outputs name must be a valid name [`%s`]: `%s`", parentName, outputNameRegex.String(), name), parent, outputsField, name))
		}

		s, e := output.GetSelector(), output.GetExpression()
		switch {
		case s != nil:
			errs = append(errs, s.Validate(ctx, opts...)...)
		case e != nil:
			if err := e.Validate(exprOpts...); err != nil {
				errs = append(errs, validation.NewMapValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("%s.outputs expression is invalid: %w", parentName, err), parent, outputsField, name))
			}
		default:
			errs = append(errs, validation.NewMapValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("%s.outputs value must be a runtime expression or a selector object", parentName), parent, outputsField, name))
		}
	}

	return errs
}
