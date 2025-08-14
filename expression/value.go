package expression

import (
	"gopkg.in/yaml.v3"
)

// ValueOrExpression represents a raw value or expression in the Arazzo document.
type ValueOrExpression = *yaml.Node

// GetValueOrExpressionValue will return the value or expression from the provided ValueOrExpression.
func GetValueOrExpressionValue(value ValueOrExpression) (*yaml.Node, *Expression, error) {
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
			asExpression := Expression(v)
			if asExpression.IsExpression() {
				return nil, &asExpression, nil
			}
		default:
			break
		}
	}

	return value, nil, nil
}
