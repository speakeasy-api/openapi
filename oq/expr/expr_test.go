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

func TestParse_UnterminatedFunction(t *testing.T) {
	t.Parallel()

	// Should not panic when tokens are exhausted inside a function call
	assert.NotPanics(t, func() {
		_, err := expr.Parse(`has(field`)
		require.Error(t, err)
	})
	assert.NotPanics(t, func() {
		_, err := expr.Parse(`matches(field,`)
		require.Error(t, err)
	})
}

func TestEval_Operators_Coverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected bool
	}{
		{
			name:     "greater or equal true",
			exprStr:  `depth >= 5`,
			row:      testRow{"depth": expr.IntVal(5)},
			expected: true,
		},
		{
			name:     "less or equal true",
			exprStr:  `depth <= 5`,
			row:      testRow{"depth": expr.IntVal(3)},
			expected: true,
		},
		{
			name:     "less than true",
			exprStr:  `depth < 10`,
			row:      testRow{"depth": expr.IntVal(3)},
			expected: true,
		},
		{
			name:     "and short-circuit false",
			exprStr:  `depth > 100 and is_component`,
			row:      testRow{"depth": expr.IntVal(1), "is_component": expr.BoolVal(true)},
			expected: false,
		},
		{
			name:     "or short-circuit true",
			exprStr:  `is_component or depth > 100`,
			row:      testRow{"depth": expr.IntVal(1), "is_component": expr.BoolVal(true)},
			expected: true,
		},
		{
			name:     "not true value",
			exprStr:  `not is_component`,
			row:      testRow{"is_component": expr.BoolVal(true)},
			expected: false,
		},
		{
			name:     "has null field",
			exprStr:  `has(missing)`,
			row:      testRow{},
			expected: false,
		},
		{
			name:     "has empty string",
			exprStr:  `has(name)`,
			row:      testRow{"name": expr.StringVal("")},
			expected: false,
		},
		{
			name:     "has non-empty string",
			exprStr:  `has(name)`,
			row:      testRow{"name": expr.StringVal("Pet")},
			expected: true,
		},
		{
			name:     "has false bool",
			exprStr:  `has(flag)`,
			row:      testRow{"flag": expr.BoolVal(false)},
			expected: false,
		},
		{
			name:     "matches non-string field",
			exprStr:  `name matches ".*"`,
			row:      testRow{"name": expr.IntVal(42)},
			expected: false,
		},
		{
			name:     "integer equality both sides",
			exprStr:  `depth == 0`,
			row:      testRow{"depth": expr.IntVal(0)},
			expected: true,
		},
		{
			name:     "boolean equality",
			exprStr:  `is_component == is_inline`,
			row:      testRow{"is_component": expr.BoolVal(true), "is_inline": expr.BoolVal(true)},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, tt.expected, result.Bool)
		})
	}
}

func TestEval_TypeConversion_Coverage(t *testing.T) {
	t.Parallel()

	// Test toBool with int
	e, err := expr.Parse(`depth`)
	require.NoError(t, err)
	row := testRow{"depth": expr.IntVal(5)}
	result := e.Eval(row)
	assert.Equal(t, expr.KindInt, result.Kind)

	// Test toBool with string (non-empty = truthy in boolean context)
	e, err = expr.Parse(`name and depth > 0`)
	require.NoError(t, err)
	row = testRow{"name": expr.StringVal("Pet"), "depth": expr.IntVal(1)}
	result = e.Eval(row)
	assert.True(t, result.Bool)

	// Test toBool with empty string (falsy)
	e, err = expr.Parse(`name and depth > 0`)
	require.NoError(t, err)
	row = testRow{"name": expr.StringVal(""), "depth": expr.IntVal(1)}
	result = e.Eval(row)
	assert.False(t, result.Bool)

	// Test comparison with string-to-int coercion
	e, err = expr.Parse(`depth > 0`)
	require.NoError(t, err)
	row = testRow{"depth": expr.BoolVal(true)} // bool true -> 1 in comparison
	result = e.Eval(row)
	assert.True(t, result.Bool)

	// Test string equality with int (cross-type via toString)
	e, err = expr.Parse(`name == "5"`)
	require.NoError(t, err)
	row = testRow{"name": expr.IntVal(5)}
	result = e.Eval(row)
	assert.True(t, result.Bool)
}

func TestParse_NullVal(t *testing.T) {
	t.Parallel()

	v := expr.NullVal()
	assert.Equal(t, expr.KindNull, v.Kind)
}

func TestParse_LiteralValues(t *testing.T) {
	t.Parallel()

	// true literal
	e, err := expr.Parse(`true`)
	require.NoError(t, err)
	result := e.Eval(testRow{})
	assert.Equal(t, expr.KindBool, result.Kind)
	assert.True(t, result.Bool)

	// false literal
	e, err = expr.Parse(`false`)
	require.NoError(t, err)
	result = e.Eval(testRow{})
	assert.Equal(t, expr.KindBool, result.Kind)
	assert.False(t, result.Bool)

	// numeric literal
	e, err = expr.Parse(`depth > 0`)
	require.NoError(t, err)
	result = e.Eval(testRow{"depth": expr.IntVal(5)})
	assert.True(t, result.Bool)
}

func TestParse_AlternativeOperator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected expr.Value
	}{
		{
			name:     "left is truthy",
			exprStr:  `name // "default"`,
			row:      testRow{"name": expr.StringVal("Pet")},
			expected: expr.StringVal("Pet"),
		},
		{
			name:     "left is null",
			exprStr:  `missing // "default"`,
			row:      testRow{},
			expected: expr.StringVal("default"),
		},
		{
			name:     "left is empty string (falsy)",
			exprStr:  `name // "default"`,
			row:      testRow{"name": expr.StringVal("")},
			expected: expr.StringVal("default"),
		},
		{
			name:     "left is false",
			exprStr:  `flag // true`,
			row:      testRow{"flag": expr.BoolVal(false)},
			expected: expr.BoolVal(true),
		},
		{
			name:     "left is zero (falsy int)",
			exprStr:  `count // 42`,
			row:      testRow{"count": expr.IntVal(0)},
			expected: expr.IntVal(42),
		},
		{
			name:     "left is nonzero int (truthy)",
			exprStr:  `count // 42`,
			row:      testRow{"count": expr.IntVal(5)},
			expected: expr.IntVal(5),
		},
		{
			name:     "chained alternative",
			exprStr:  `a // b // "fallback"`,
			row:      testRow{"a": expr.NullVal(), "b": expr.StringVal("")},
			expected: expr.StringVal("fallback"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParse_IfThenElse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected expr.Value
	}{
		{
			name:     "if true then value",
			exprStr:  `if is_component then depth else 0 end`,
			row:      testRow{"is_component": expr.BoolVal(true), "depth": expr.IntVal(5)},
			expected: expr.IntVal(5),
		},
		{
			name:     "if false else value",
			exprStr:  `if is_component then depth else 0 end`,
			row:      testRow{"is_component": expr.BoolVal(false), "depth": expr.IntVal(5)},
			expected: expr.IntVal(0),
		},
		{
			name:     "if without else returns null",
			exprStr:  `if is_component then depth end`,
			row:      testRow{"is_component": expr.BoolVal(false), "depth": expr.IntVal(5)},
			expected: expr.NullVal(),
		},
		{
			name:     "nested if-then-else",
			exprStr:  `if depth > 10 then "deep" elif depth > 5 then "medium" else "shallow" end`,
			row:      testRow{"depth": expr.IntVal(7)},
			expected: expr.StringVal("medium"),
		},
		{
			name:     "if in boolean context",
			exprStr:  `if is_component then depth > 3 else depth > 5 end`,
			row:      testRow{"is_component": expr.BoolVal(true), "depth": expr.IntVal(4)},
			expected: expr.BoolVal(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParse_StringInterpolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected string
	}{
		{
			name:     "simple interpolation",
			exprStr:  `"hello \(name)"`,
			row:      testRow{"name": expr.StringVal("world")},
			expected: "hello world",
		},
		{
			name:     "interpolation with expr",
			exprStr:  `"\(name) has depth \(depth)"`,
			row:      testRow{"name": expr.StringVal("Pet"), "depth": expr.IntVal(3)},
			expected: "Pet has depth 3",
		},
		{
			name:     "no interpolation",
			exprStr:  `"plain string"`,
			row:      testRow{},
			expected: "plain string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, expr.KindString, result.Kind)
			assert.Equal(t, tt.expected, result.Str)
		})
	}
}

func TestParse_IfThenElse_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		exprStr string
	}{
		{"missing then", `if true depth end`},
		{"missing end", `if true then depth`},
		{"missing end after else", `if true then depth else 0`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := expr.Parse(tt.exprStr)
			assert.Error(t, err)
		})
	}
}

func TestParse_InterpolationError(t *testing.T) {
	t.Parallel()

	// Unterminated interpolation
	_, err := expr.Parse(`"hello \(name"`)
	require.Error(t, err)
}

func TestParse_ComplexPrecedence(t *testing.T) {
	t.Parallel()

	// a and b or c and d — "and" binds tighter, so this is (a and b) or (c and d)
	e, err := expr.Parse(`depth > 0 and is_component or depth < 0 and is_inline`)
	require.NoError(t, err)

	// Both "and" groups are false -> false
	result := e.Eval(testRow{
		"depth":        expr.IntVal(0),
		"is_component": expr.BoolVal(true),
		"is_inline":    expr.BoolVal(true),
	})
	assert.False(t, result.Bool)

	// First "and" group is true -> true
	result = e.Eval(testRow{
		"depth":        expr.IntVal(5),
		"is_component": expr.BoolVal(true),
		"is_inline":    expr.BoolVal(false),
	})
	assert.True(t, result.Bool)
}

func TestEval_StringFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected expr.Value
	}{
		{"lower", `lower("Hello")`, testRow{}, expr.StringVal("hello")},
		{"upper", `upper("Hello")`, testRow{}, expr.StringVal("HELLO")},
		{"lower field", `lower(name)`, testRow{"name": expr.StringVal("Pet")}, expr.StringVal("pet")},
		{"trim", `trim("  hello  ")`, testRow{}, expr.StringVal("hello")},
		{"len string", `len("hello")`, testRow{}, expr.IntVal(5)},
		{"len field", `len(name)`, testRow{"name": expr.StringVal("Pet")}, expr.IntVal(3)},
		{"len array", `len(tags)`, testRow{"tags": expr.ArrayVal([]string{"a", "b", "c"})}, expr.IntVal(3)},
		{"startswith true", `startswith(name, "Pe")`, testRow{"name": expr.StringVal("Pet")}, expr.BoolVal(true)},
		{"startswith false", `startswith(name, "Ow")`, testRow{"name": expr.StringVal("Pet")}, expr.BoolVal(false)},
		{"endswith true", `endswith(name, "et")`, testRow{"name": expr.StringVal("Pet")}, expr.BoolVal(true)},
		{"contains string true", `contains(name, "et")`, testRow{"name": expr.StringVal("Pet")}, expr.BoolVal(true)},
		{"contains string false", `contains(name, "xx")`, testRow{"name": expr.StringVal("Pet")}, expr.BoolVal(false)},
		{"replace", `replace(name, "Pet", "Cat")`, testRow{"name": expr.StringVal("Pet")}, expr.StringVal("Cat")},
		{"split with index", `split(path, "/", 1)`, testRow{"path": expr.StringVal("/users/123")}, expr.StringVal("users")},
		{"split out of range", `split(path, "/", 99)`, testRow{"path": expr.StringVal("/users")}, expr.NullVal()},
		{"composition lower+startswith", `startswith(lower(name), "pe")`, testRow{"name": expr.StringVal("Pet")}, expr.BoolVal(true)},
		{"count array", `count(tags)`, testRow{"tags": expr.ArrayVal([]string{"a", "b"})}, expr.IntVal(2)},
		{"count string", `count(name)`, testRow{"name": expr.StringVal("Pet")}, expr.IntVal(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEval_Arithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected expr.Value
	}{
		{"add", `depth + 1`, testRow{"depth": expr.IntVal(5)}, expr.IntVal(6)},
		{"subtract", `depth - 3`, testRow{"depth": expr.IntVal(5)}, expr.IntVal(2)},
		{"multiply", `depth * 2`, testRow{"depth": expr.IntVal(5)}, expr.IntVal(10)},
		{"divide", `depth / 2`, testRow{"depth": expr.IntVal(10)}, expr.IntVal(5)},
		{"divide by zero", `depth / 0`, testRow{"depth": expr.IntVal(10)}, expr.NullVal()},
		{"precedence mul before add", `2 + 3 * 4`, testRow{}, expr.IntVal(14)},
		{"field arithmetic", `in_degree + out_degree`, testRow{"in_degree": expr.IntVal(3), "out_degree": expr.IntVal(5)}, expr.IntVal(8)},
		{"arithmetic in comparison", `in_degree + out_degree > 5`, testRow{"in_degree": expr.IntVal(3), "out_degree": expr.IntVal(5)}, expr.BoolVal(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEval_Contains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exprStr  string
		row      testRow
		expected bool
	}{
		{"string contains", `name contains "et"`, testRow{"name": expr.StringVal("Pet")}, true},
		{"string not contains", `name contains "xx"`, testRow{"name": expr.StringVal("Pet")}, false},
		{"array contains", `tags contains "billing"`, testRow{"tags": expr.ArrayVal([]string{"api", "billing"})}, true},
		{"array not contains", `tags contains "admin"`, testRow{"tags": expr.ArrayVal([]string{"api", "billing"})}, false},
		{"null contains", `missing contains "x"`, testRow{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := expr.Parse(tt.exprStr)
			require.NoError(t, err)
			result := parsed.Eval(tt.row)
			assert.Equal(t, expr.KindBool, result.Kind)
			assert.Equal(t, tt.expected, result.Bool)
		})
	}
}

func TestEval_ArrayVal(t *testing.T) {
	t.Parallel()

	// Array toBool
	e, err := expr.Parse(`tags`)
	require.NoError(t, err)

	result := e.Eval(testRow{"tags": expr.ArrayVal([]string{"a"})})
	assert.Equal(t, expr.KindArray, result.Kind)

	result = e.Eval(testRow{"tags": expr.ArrayVal(nil)})
	assert.Equal(t, expr.KindArray, result.Kind)

	// has() with array
	e, err = expr.Parse(`has(tags)`)
	require.NoError(t, err)
	result = e.Eval(testRow{"tags": expr.ArrayVal([]string{"a"})})
	assert.True(t, result.Bool)
	result = e.Eval(testRow{"tags": expr.ArrayVal(nil)})
	assert.False(t, result.Bool)
}
