package expression

import (
	"go.yaml.in/yaml/v3"
)

// ValueOrExpression represents a raw value or expression in the Arazzo document.
type ValueOrExpression = *yaml.Node

// GetValueOrExpressionValue will return the value or expression from the provided ValueOrExpression.
// Pass WithArazzo11Grammar() to also recognize Arazzo 1.1 expressions such as `$message` and `$self`.
func GetValueOrExpressionValue(value ValueOrExpression, opts ...ValidateOption) (*yaml.Node, *Expression, error) {
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
			if asExpression.IsExpression(opts...) {
				return nil, &asExpression, nil
			}
		default:
			break
		}
	}

	return value, nil, nil
}
