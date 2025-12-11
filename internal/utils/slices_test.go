package utils_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestMapSlice_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []int
		fn       func(int) string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []int{},
			fn:       func(i int) string { return "" },
			expected: []string{},
		},
		{
			name:  "single element",
			input: []int{1},
			fn: func(i int) string {
				if i == 1 {
					return "one"
				}
				return ""
			},
			expected: []string{"one"},
		},
		{
			name:  "multiple elements",
			input: []int{1, 2, 3},
			fn: func(i int) string {
				return string(rune('a' + i - 1))
			},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := utils.MapSlice(tt.input, tt.fn)
			assert.Equal(t, tt.expected, result)
		})
	}
}
