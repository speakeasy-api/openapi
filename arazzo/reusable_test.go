package arazzo

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeToComponentType_Success(t *testing.T) {
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
			actual := typeToComponentType(tt.input)
			assert.Equal(t, tt.expected, actual, "type conversion should match expected component type")
		})
	}
}
