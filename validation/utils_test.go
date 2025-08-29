package validation

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test SortValidationErrors function
func TestSortValidationErrors_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errors   []error
		expected []error
	}{
		{
			name:     "empty slice",
			errors:   []error{},
			expected: []error{},
		},
		{
			name: "single validation error",
			errors: []error{
				&Error{
					UnderlyingError: stderrors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
		},
		{
			name: "multiple validation errors sorted by line",
			errors: []error{
				&Error{
					UnderlyingError: stderrors.New("error3"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
				&Error{
					UnderlyingError: stderrors.New("error1"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				&Error{
					UnderlyingError: stderrors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 8},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("error1"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				&Error{
					UnderlyingError: stderrors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 8},
				},
				&Error{
					UnderlyingError: stderrors.New("error3"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
			},
		},
		{
			name: "validation errors with same line sorted by column",
			errors: []error{
				&Error{
					UnderlyingError: stderrors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 15},
				},
				&Error{
					UnderlyingError: stderrors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 3},
				},
				&Error{
					UnderlyingError: stderrors.New("error3"),
					Node:            &yaml.Node{Line: 5, Column: 20},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 3},
				},
				&Error{
					UnderlyingError: stderrors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 15},
				},
				&Error{
					UnderlyingError: stderrors.New("error3"),
					Node:            &yaml.Node{Line: 5, Column: 20},
				},
			},
		},
		{
			name: "mix of validation errors and regular errors",
			errors: []error{
				stderrors.New("regular error 2"),
				&Error{
					UnderlyingError: stderrors.New("validation error"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
				stderrors.New("regular error 1"),
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("validation error"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
				stderrors.New("regular error 2"),
				stderrors.New("regular error 1"),
			},
		},
		{
			name: "only regular errors",
			errors: []error{
				stderrors.New("error C"),
				stderrors.New("error A"),
				stderrors.New("error B"),
			},
			expected: []error{
				stderrors.New("error C"),
				stderrors.New("error A"),
				stderrors.New("error B"),
			},
		},
		{
			name: "validation errors with nil nodes",
			errors: []error{
				&Error{
					UnderlyingError: stderrors.New("error with nil node"),
					Node:            nil,
				},
				&Error{
					UnderlyingError: stderrors.New("error with node"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("error with nil node"),
					Node:            nil,
				},
				&Error{
					UnderlyingError: stderrors.New("error with node"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
		},
		{
			name: "complex mixed scenario",
			errors: []error{
				stderrors.New("regular error"),
				&Error{
					UnderlyingError: stderrors.New("validation error line 10"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
				&Error{
					UnderlyingError: stderrors.New("validation error line 2 col 15"),
					Node:            &yaml.Node{Line: 2, Column: 15},
				},
				&Error{
					UnderlyingError: stderrors.New("validation error line 2 col 3"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				stderrors.New("another regular error"),
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("validation error line 2 col 3"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				&Error{
					UnderlyingError: stderrors.New("validation error line 2 col 15"),
					Node:            &yaml.Node{Line: 2, Column: 15},
				},
				&Error{
					UnderlyingError: stderrors.New("validation error line 10"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
				stderrors.New("regular error"),
				stderrors.New("another regular error"),
			},
		},
		{
			name: "validation errors with zero line/column",
			errors: []error{
				&Error{
					UnderlyingError: stderrors.New("error at 0,0"),
					Node:            &yaml.Node{Line: 0, Column: 0},
				},
				&Error{
					UnderlyingError: stderrors.New("error at 1,1"),
					Node:            &yaml.Node{Line: 1, Column: 1},
				},
				&Error{
					UnderlyingError: stderrors.New("error at 0,5"),
					Node:            &yaml.Node{Line: 0, Column: 5},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: stderrors.New("error at 0,0"),
					Node:            &yaml.Node{Line: 0, Column: 0},
				},
				&Error{
					UnderlyingError: stderrors.New("error at 0,5"),
					Node:            &yaml.Node{Line: 0, Column: 5},
				},
				&Error{
					UnderlyingError: stderrors.New("error at 1,1"),
					Node:            &yaml.Node{Line: 1, Column: 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy to avoid modifying the original slice
			errorsCopy := make([]error, len(tt.errors))
			copy(errorsCopy, tt.errors)

			SortValidationErrors(errorsCopy)

			assert.Len(t, errorsCopy, len(tt.expected))

			for i, expectedErr := range tt.expected {
				if i >= len(errorsCopy) {
					t.Errorf("Expected error at index %d, but slice is too short", i)
					continue
				}

				actualErr := errorsCopy[i]

				// Check if both are validation errors
				var expectedValidationErr, actualValidationErr *Error
				expectedIsValidation := stderrors.As(expectedErr, &expectedValidationErr)
				actualIsValidation := stderrors.As(actualErr, &actualValidationErr)

				switch {
				case expectedIsValidation && actualIsValidation:
					// Compare validation errors
					assert.Equal(t, expectedValidationErr.UnderlyingError.Error(), actualValidationErr.UnderlyingError.Error())
					if expectedValidationErr.Node == nil {
						assert.Nil(t, actualValidationErr.Node)
					} else {
						require.NotNil(t, actualValidationErr.Node)
						assert.Equal(t, expectedValidationErr.Node.Line, actualValidationErr.Node.Line)
						assert.Equal(t, expectedValidationErr.Node.Column, actualValidationErr.Node.Column)
					}
				case !expectedIsValidation && !actualIsValidation:
					// Compare regular errors
					assert.Equal(t, expectedErr.Error(), actualErr.Error())
				default:
					t.Errorf("Type mismatch at index %d: expected validation=%v, actual validation=%v",
						i, expectedIsValidation, actualIsValidation)
				}
			}
		})
	}
}

// Test edge cases for SortValidationErrors
func TestSortValidationErrors_EdgeCases_Success(t *testing.T) {
	t.Parallel()

	t.Run("nil slice", func(t *testing.T) {
		t.Parallel()

		var nilSlice []error
		assert.NotPanics(t, func() {
			SortValidationErrors(nilSlice)
		})
	})

	t.Run("slice with nil errors", func(t *testing.T) {
		t.Parallel()

		errors := []error{
			nil,
			&Error{
				UnderlyingError: stderrors.New("valid error"),
				Node:            &yaml.Node{Line: 1, Column: 1},
			},
			nil,
		}

		assert.NotPanics(t, func() {
			SortValidationErrors(errors)
		})

		// Validation errors should come first, then nil errors
		var validationErr *Error
		require.ErrorAs(t, errors[0], &validationErr)
		assert.Equal(t, "valid error", validationErr.UnderlyingError.Error())
		assert.NoError(t, errors[1])
		assert.NoError(t, errors[2])
	})

	t.Run("negative line/column numbers", func(t *testing.T) {
		t.Parallel()

		errors := []error{
			&Error{
				UnderlyingError: stderrors.New("error with negative line"),
				Node:            &yaml.Node{Line: -1, Column: 5},
			},
			&Error{
				UnderlyingError: stderrors.New("error with positive line"),
				Node:            &yaml.Node{Line: 1, Column: 5},
			},
			&Error{
				UnderlyingError: stderrors.New("error with negative column"),
				Node:            &yaml.Node{Line: 1, Column: -1},
			},
		}

		SortValidationErrors(errors)

		// Should be sorted by line first, then column
		var err0, err1, err2 *Error
		require.ErrorAs(t, errors[0], &err0)
		require.ErrorAs(t, errors[1], &err1)
		require.ErrorAs(t, errors[2], &err2)
		assert.Equal(t, "error with negative line", err0.UnderlyingError.Error())
		assert.Equal(t, "error with negative column", err1.UnderlyingError.Error())
		assert.Equal(t, "error with positive line", err2.UnderlyingError.Error())
	})
}
