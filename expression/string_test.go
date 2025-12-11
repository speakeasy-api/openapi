package expression_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/stretchr/testify/assert"
)

func TestExpression_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression expression.Expression
		expected   string
	}{
		{
			name:       "simple expression",
			expression: expression.Expression("$url"),
			expected:   "$url",
		},
		{
			name:       "complex expression",
			expression: expression.Expression("$response.body#/data/id"),
			expected:   "$response.body#/data/id",
		},
		{
			name:       "empty expression",
			expression: expression.Expression(""),
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.expression.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
