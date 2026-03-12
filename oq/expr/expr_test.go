package expr_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/oq/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRow map[string]expr.Value

func (r testRow) Field(name string) expr.Value {
	if v, ok := r[name]; ok {
		return v
	}
	return expr.NullVal()
}

func TestParse_Comparison_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expr     string
		row      testRow
		expected bool
	}{
		{
			name:     "integer equality",
			expr:     `depth == 5`,
			row:      testRow{"depth": expr.IntVal(5)},
			expected: true,
		},
		{
			name:     "integer inequality",
			expr:     `depth != 5`,
			row:      testRow{"depth": expr.IntVal(3)},
			expected: true,
		},
		{
			name:     "greater than",
			expr:     `depth > 3`,
			row:      testRow{"depth": expr.IntVal(5)},
			expected: true,
		},
		{
			name:     "less than false",
			expr:     `depth < 3`,
			row:      testRow{"depth": expr.IntVal(5)},
			expected: false,
		},
		{
			name:     "string equality",
			expr:     `type == "object"`,
			row:      testRow{"type": expr.StringVal("object")},
			expected: true,
		},
		{
			name:     "boolean field",
			expr:     `is_component`,
			row:      testRow{"is_component": expr.BoolVal(true)},
			expected: true,
		},
		{
			name:     "and operator",
			expr:     `depth > 3 and is_component`,
			row:      testRow{"depth": expr.IntVal(5), "is_component": expr.BoolVal(true)},
			expected: true,
		},
		{
			name:     "or operator",
			expr:     `depth > 10 or is_component`,
			row:      testRow{"depth": expr.IntVal(2), "is_component": expr.BoolVal(true)},
			expected: true,
		},
		{
			name:     "not operator",
			expr:     `not is_inline`,
			row:      testRow{"is_inline": expr.BoolVal(false)},
			expected: true,
		},
		{
			name:     "has function",
			expr:     `has(oneOf)`,
			row:      testRow{"oneOf": expr.IntVal(2)},
			expected: true,
		},
		{
			name:     "has function false",
			expr:     `has(oneOf)`,
			row:      testRow{"oneOf": expr.IntVal(0)},
			expected: false,
		},
		{
			name:     "matches operator",
			expr:     `name matches "Error.*"`,
			row:      testRow{"name": expr.StringVal("ErrorResponse")},
			expected: true,
		},
		{
			name:     "matches operator no match",
			expr:     `name matches "Error.*"`,
			row:      testRow{"name": expr.StringVal("Pet")},
			expected: false,
		},
		{
			name:     "complex expression",
			expr:     `property_count > 0 and in_degree == 0`,
			row:      testRow{"property_count": expr.IntVal(3), "in_degree": expr.IntVal(0)},
			expected: true,
		},
		{
			name:     "parenthesized expression",
			expr:     `(depth > 3 or depth < 1) and is_component`,
			row:      testRow{"depth": expr.IntVal(5), "is_component": expr.BoolVal(true)},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := expr.Parse(tt.expr)
			require.NoError(t, err)

			result := parsed.Eval(tt.row)
			assert.Equal(t, expr.KindBool, result.Kind)
			assert.Equal(t, tt.expected, result.Bool)
		})
	}
}

func TestParse_Error(t *testing.T) {
	t.Parallel()

	_, err := expr.Parse("")
	require.Error(t, err)

	_, err = expr.Parse("name matches \"[invalid\"")
	require.Error(t, err)
}

func TestParse_UnterminatedBackslashString(t *testing.T) {
	t.Parallel()

	// Should not panic on unterminated string ending with backslash
	assert.NotPanics(t, func() {
		expr.Parse(`name == "x\`) //nolint:errcheck
	})
}
