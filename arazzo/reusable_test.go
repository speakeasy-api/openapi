package arazzo

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeToComponentType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		{
			name:     "Parameter type converts to parameters",
			input:    reflect.TypeOf(Parameter{}),
			expected: "parameters",
		},
		{
			name:     "SuccessAction type converts to successActions",
			input:    reflect.TypeOf(SuccessAction{}),
			expected: "successActions",
		},
		{
			name:     "FailureAction type converts to failureActions",
			input:    reflect.TypeOf(FailureAction{}),
			expected: "failureActions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := typeToComponentType(tt.input)
			assert.Equal(t, tt.expected, actual, "type conversion should match expected component type")
		})
	}
}

func TestComponentTypeToReusableType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "parameters converts to reusableParameter",
			input:    "parameters",
			expected: "reusableParameter",
		},
		{
			name:     "successActions converts to reusableSuccessAction",
			input:    "successActions",
			expected: "reusableSuccessAction",
		},
		{
			name:     "failureActions converts to reusableFailureAction",
			input:    "failureActions",
			expected: "reusableFailureAction",
		},
		{
			name:     "unknown type returns empty",
			input:    "unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := componentTypeToReusableType(tt.input)
			assert.Equal(t, tt.expected, actual, "component type conversion should match expected reusable type")
		})
	}
}

func TestReusable_IsReference_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reusable ReusableParameter
		expected bool
	}{
		{
			name:     "nil reference returns false",
			reusable: ReusableParameter{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.reusable.IsReference())
		})
	}
}

func TestReusable_Get_Success(t *testing.T) {
	t.Parallel()

	param := &Parameter{Name: "testParam"}

	tests := []struct {
		name       string
		reusable   ReusableParameter
		components *Components
		expected   *Parameter
	}{
		{
			name:       "inline object returns object",
			reusable:   ReusableParameter{Object: param},
			components: nil,
			expected:   param,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.reusable.Get(tt.components)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReusable_GetReferencedObject_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reusable   ReusableParameter
		components *Components
		expectNil  bool
	}{
		{
			name:       "not a reference returns nil",
			reusable:   ReusableParameter{},
			components: nil,
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.reusable.GetReferencedObject(tt.components)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}
