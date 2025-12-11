package expression_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetValueOrExpressionValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		value          *yaml.Node
		expectValue    bool
		expectExpr     bool
		expectNil      bool
		expectedString string
	}{
		{
			name:       "nil value returns nil",
			value:      nil,
			expectNil:  true,
			expectExpr: false,
		},
		{
			name:        "expression string returns expression",
			value:       &yaml.Node{Kind: yaml.ScalarNode, Value: "$url"},
			expectExpr:  true,
			expectValue: false,
		},
		{
			name:        "non-expression string returns value",
			value:       &yaml.Node{Kind: yaml.ScalarNode, Value: "plain string"},
			expectExpr:  false,
			expectValue: true,
		},
		{
			name:        "integer returns value",
			value:       &yaml.Node{Kind: yaml.ScalarNode, Value: "42", Tag: "!!int"},
			expectExpr:  false,
			expectValue: true,
		},
		{
			name:        "boolean returns value",
			value:       &yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
			expectExpr:  false,
			expectValue: true,
		},
		{
			name: "map node returns value",
			value: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key"},
					{Kind: yaml.ScalarNode, Value: "val"},
				},
			},
			expectExpr:  false,
			expectValue: true,
		},
		{
			name: "sequence node returns value",
			value: &yaml.Node{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "item1"},
				},
			},
			expectExpr:  false,
			expectValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, expr, err := expression.GetValueOrExpressionValue(tt.value)

			require.NoError(t, err)

			switch {
			case tt.expectNil:
				assert.Nil(t, value)
				assert.Nil(t, expr)
			case tt.expectExpr:
				assert.Nil(t, value)
				assert.NotNil(t, expr)
			case tt.expectValue:
				assert.NotNil(t, value)
				assert.Nil(t, expr)
			}
		})
	}
}
