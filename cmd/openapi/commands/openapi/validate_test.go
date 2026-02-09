package openapi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValidationErrors_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		validationErrors []error
		expected         string
	}{
		{
			name: "single item without left padding",
			validationErrors: []error{
				errors.New("missing openapi field"),
			},
			expected: "1. missing openapi field\n",
		},
		{
			name: "double-digit list aligns index column",
			validationErrors: []error{
				errors.New("err 1"),
				errors.New("err 2"),
				errors.New("err 3"),
				errors.New("err 4"),
				errors.New("err 5"),
				errors.New("err 6"),
				errors.New("err 7"),
				errors.New("err 8"),
				errors.New("err 9"),
				errors.New("err 10"),
				errors.New("err 11"),
				errors.New("err 12"),
			},
			expected: " 1. err 1\n 2. err 2\n 3. err 3\n 4. err 4\n 5. err 5\n 6. err 6\n 7. err 7\n 8. err 8\n 9. err 9\n10. err 10\n11. err 11\n12. err 12\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := formatValidationErrors(tt.validationErrors)
			assert.Equal(t, tt.expected, actual, "should format validation errors with aligned numbering")
		})
	}
}
