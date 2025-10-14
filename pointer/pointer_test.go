package pointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrom_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		wantValue any
	}{
		{
			name:      "creates pointer from string",
			input:     "test",
			wantValue: From("test"),
		},
		{
			name:      "creates pointer from empty string",
			input:     "",
			wantValue: From(""),
		},
		{
			name:      "creates pointer from int",
			input:     42,
			wantValue: From(42),
		},
		{
			name:      "creates pointer from zero int",
			input:     0,
			wantValue: From(0),
		},
		{
			name:      "creates pointer from negative int",
			input:     -1,
			wantValue: From(-1),
		},
		{
			name:      "creates pointer from bool true",
			input:     true,
			wantValue: From(true),
		},
		{
			name:      "creates pointer from bool false",
			input:     false,
			wantValue: From(false),
		},
		{
			name:      "creates pointer from float64",
			input:     3.14,
			wantValue: From(3.14),
		},
		{
			name:      "creates pointer from zero float64",
			input:     0.0,
			wantValue: From(0.0),
		},
		{
			name:      "creates pointer from negative float64",
			input:     -2.5,
			wantValue: From(-2.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result any
			switch v := tt.input.(type) {
			case string:
				result = From(v)
			case int:
				result = From(v)
			case bool:
				result = From(v)
			case float64:
				result = From(v)
			}

			assert.Equal(t, tt.wantValue, result, "should return correct pointer value")
		})
	}
}

func TestFrom_Struct(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Name  string
		Value int
	}

	input := testStruct{Name: "test", Value: 42}
	wantValue := From(testStruct{Name: "test", Value: 42})
	result := From(input)

	assert.Equal(t, wantValue, result, "should return correct pointer value")
}

func TestValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		wantValue any
	}{
		{
			name:      "returns value from string pointer",
			input:     From("test"),
			wantValue: "test",
		},
		{
			name:      "returns value from int pointer",
			input:     From(42),
			wantValue: 42,
		},
		{
			name:      "returns value from bool pointer",
			input:     From(true),
			wantValue: true,
		},
		{
			name:      "returns value from float64 pointer",
			input:     From(3.14),
			wantValue: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result any
			switch v := tt.input.(type) {
			case *string:
				result = Value(v)
			case *int:
				result = Value(v)
			case *bool:
				result = Value(v)
			case *float64:
				result = Value(v)
			}

			assert.Equal(t, tt.wantValue, result, "should return correct value")
		})
	}
}

func TestValue_ReturnsZeroForNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		wantValue any
	}{
		{
			name:      "returns empty string for nil string pointer",
			input:     (*string)(nil),
			wantValue: "",
		},
		{
			name:      "returns zero for nil int pointer",
			input:     (*int)(nil),
			wantValue: 0,
		},
		{
			name:      "returns false for nil bool pointer",
			input:     (*bool)(nil),
			wantValue: false,
		},
		{
			name:      "returns zero for nil float64 pointer",
			input:     (*float64)(nil),
			wantValue: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result any
			switch tt.input.(type) {
			case *string:
				result = Value((*string)(nil))
			case *int:
				result = Value((*int)(nil))
			case *bool:
				result = Value((*bool)(nil))
			case *float64:
				result = Value((*float64)(nil))
			}

			assert.Equal(t, tt.wantValue, result, "should return zero value")
		})
	}
}

func TestValue_Struct(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Name  string
		Value int
	}

	t.Run("returns value from struct pointer", func(t *testing.T) {
		t.Parallel()

		input := From(testStruct{Name: "test", Value: 42})
		wantValue := testStruct{Name: "test", Value: 42}
		result := Value(input)

		assert.Equal(t, wantValue, result, "should return correct struct value")
	})

	t.Run("returns zero struct for nil pointer", func(t *testing.T) {
		t.Parallel()

		wantValue := testStruct{}
		result := Value((*testStruct)(nil))

		assert.Equal(t, wantValue, result, "should return zero struct value")
	})
}

func TestPointer_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     any
		wantValue any
	}{
		{
			name:      "string round trip",
			value:     "test",
			wantValue: "test",
		},
		{
			name:      "int round trip",
			value:     42,
			wantValue: 42,
		},
		{
			name:      "bool round trip",
			value:     true,
			wantValue: true,
		},
		{
			name:      "float64 round trip",
			value:     3.14,
			wantValue: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result any
			switch v := tt.value.(type) {
			case string:
				result = Value(From(v))
			case int:
				result = Value(From(v))
			case bool:
				result = Value(From(v))
			case float64:
				result = Value(From(v))
			}

			assert.Equal(t, tt.wantValue, result, "should match original value after round trip")
		})
	}
}
