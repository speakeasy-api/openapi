package sliceutil_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/internal/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		slice    []int
		fn       func(int) int
		expected []int
	}{
		{
			name:     "empty slice",
			slice:    []int{},
			fn:       func(i int) int { return i },
			expected: []int{},
		},
		{
			name:     "single element",
			slice:    []int{1},
			fn:       func(i int) int { return i },
			expected: []int{1},
		},
		{
			name:     "multiple elements",
			slice:    []int{1, 2, 3},
			fn:       func(i int) int { return i },
			expected: []int{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sliceutil.Map(tt.slice, tt.fn)
			assert.Equal(t, tt.expected, result)
		})
	}
}
