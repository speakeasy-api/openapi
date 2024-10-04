package arazzo

import (
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"gopkg.in/yaml.v3"
)

// Value represents a raw value in the Arazzo document.
type Value = *yaml.Node

// ValueOrExpression represents a raw value or expression in the Arazzo document.
type ValueOrExpression = *yaml.Node

// GetValueOrExpressionValue will return the value or expression from the provided ValueOrExpression.
func GetValueOrExpressionValue(value ValueOrExpression) (*yaml.Node, *expression.Expression, error) {
	if value == nil {
		return nil, nil, nil
	}

	if value.Kind == yaml.ScalarNode {
		var val any
		if err := value.Decode(&val); err != nil {
			return nil, nil, err
		}

		switch v := val.(type) {
		case string:
			asExpression := expression.Expression(v)
			if asExpression.IsExpression() {
				return nil, &asExpression, nil
			}
		}
	}

	return value, nil, nil
}
