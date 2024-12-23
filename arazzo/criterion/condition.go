package criterion

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/validation"
)

// Operator represents the operator used to compare the value of a criterion.
// TODO ignoring the Logical grouping, index & property de-reference operators for now as there is consistency issues with the spec on how/when these are used (they should probably be part of the expression type)
type Operator string

const (
	OperatorLT  Operator = "<"
	OperatorLTE Operator = "<="
	OperatorGT  Operator = ">"
	OperatorGTE Operator = ">="
	OperatorEQ  Operator = "=="
	OperatorNE  Operator = "!="
	OperatorNot Operator = "!" // TODO not entirely sure how this is supposed to work
	OperatorAnd Operator = "&&"
	OperatorOr  Operator = "||"
)

// Condition represents a condition that will be evaluated for a given criterion.
type Condition struct {
	rawCondition string

	// Expression is the expression to the value to be evaluated.
	Expression expression.Expression
	// Operator is the operator used to compare the value of the condition.
	Operator Operator
	// Value is the value to compare the value of the condition to.
	Value any
}

// TODO this will need to evolve to have a more AST like structure (while remaining easy to work with)
func newCondition(rawCondition string) (*Condition, error) {
	parts := strings.Split(rawCondition, " ")

	if len(parts) < 3 {
		return nil, fmt.Errorf("condition must at least be in the format [expression] [operator] [value]")
	}

	if strings.ContainsAny(rawCondition, "&|") {
		// TODO this is a complex condition that we don't currently support
		return nil, nil
	}

	// String literal value handling (single quotes) until parsing is tokenized.
	// Reference: https://spec.openapis.org/arazzo/v1.0.0#literals
	if len(parts) > 3 && strings.HasPrefix(parts[2], "'") && strings.HasSuffix(parts[len(parts)-1], "'") {
		parts[2] = strings.Join(parts[2:], " ")
		parts = parts[:3]
	}

	if len(parts) > 3 {
		// TODO this is a complex condition that we don't currently support
		return nil, nil
	}

	c := &Condition{
		rawCondition: rawCondition,
	}

	c.Expression = expression.Expression(parts[0])
	c.Operator = Operator(parts[1])
	c.Value = parts[2]

	return c, nil
}

// Validate will validate the condition is valid as per the Arazzo specification.
func (s *Condition) Validate(line, column int, opts ...validation.Option) []error {
	errs := []error{}

	if s.Expression == "" {
		errs = append(errs, &validation.Error{
			Message: "expression is required",
			Line:    line,
			Column:  column,
		})
	}

	if err := s.Expression.Validate(true); err != nil {
		errs = append(errs, &validation.Error{
			Message: err.Error(),
			Line:    line,
			Column:  column,
		})
	}

	switch s.Operator {
	case OperatorLT, OperatorLTE, OperatorGT, OperatorGTE, OperatorEQ, OperatorNE, OperatorNot, OperatorAnd, OperatorOr:
	default:
		errs = append(errs, &validation.Error{
			Message: fmt.Sprintf("operator must be one of [%s]", strings.Join([]string{string(OperatorLT), string(OperatorLTE), string(OperatorGT), string(OperatorGTE), string(OperatorEQ), string(OperatorNE), string(OperatorNot), string(OperatorAnd), string(OperatorOr)}, ", ")),
			Line:    line,
			Column:  column,
		})
	}

	if s.Value == "" {
		errs = append(errs, &validation.Error{
			Message: "value is required",
			Line:    line,
			Column:  column,
		})
	}

	return errs
}
