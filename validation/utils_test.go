package validation

import (
	"errors"
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
					UnderlyingError: errors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
		},
		{
			name: "multiple validation errors sorted by line",
			errors: []error{
				&Error{
					UnderlyingError: errors.New("error3"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
				&Error{
					UnderlyingError: errors.New("error1"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				&Error{
					UnderlyingError: errors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 8},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("error1"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				&Error{
					UnderlyingError: errors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 8},
				},
				&Error{
					UnderlyingError: errors.New("error3"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
			},
		},
		{
			name: "validation errors with same line sorted by column",
			errors: []error{
				&Error{
					UnderlyingError: errors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 15},
				},
				&Error{
					UnderlyingError: errors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 3},
				},
				&Error{
					UnderlyingError: errors.New("error3"),
					Node:            &yaml.Node{Line: 5, Column: 20},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("error1"),
					Node:            &yaml.Node{Line: 5, Column: 3},
				},
				&Error{
					UnderlyingError: errors.New("error2"),
					Node:            &yaml.Node{Line: 5, Column: 15},
				},
				&Error{
					UnderlyingError: errors.New("error3"),
					Node:            &yaml.Node{Line: 5, Column: 20},
				},
			},
		},
		{
			name: "mix of validation errors and regular errors",
			errors: []error{
				errors.New("regular error 2"),
				&Error{
					UnderlyingError: errors.New("validation error"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
				errors.New("regular error 1"),
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("validation error"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
				errors.New("regular error 2"),
				errors.New("regular error 1"),
			},
		},
		{
			name: "only regular errors",
			errors: []error{
				errors.New("error C"),
				errors.New("error A"),
				errors.New("error B"),
			},
			expected: []error{
				errors.New("error C"),
				errors.New("error A"),
				errors.New("error B"),
			},
		},
		{
			name: "validation errors with nil nodes",
			errors: []error{
				&Error{
					UnderlyingError: errors.New("error with nil node"),
					Node:            nil,
				},
				&Error{
					UnderlyingError: errors.New("error with node"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("error with nil node"),
					Node:            nil,
				},
				&Error{
					UnderlyingError: errors.New("error with node"),
					Node:            &yaml.Node{Line: 5, Column: 10},
				},
			},
		},
		{
			name: "complex mixed scenario",
			errors: []error{
				errors.New("regular error"),
				&Error{
					UnderlyingError: errors.New("validation error line 10"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
				&Error{
					UnderlyingError: errors.New("validation error line 2 col 15"),
					Node:            &yaml.Node{Line: 2, Column: 15},
				},
				&Error{
					UnderlyingError: errors.New("validation error line 2 col 3"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				errors.New("another regular error"),
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("validation error line 2 col 3"),
					Node:            &yaml.Node{Line: 2, Column: 3},
				},
				&Error{
					UnderlyingError: errors.New("validation error line 2 col 15"),
					Node:            &yaml.Node{Line: 2, Column: 15},
				},
				&Error{
					UnderlyingError: errors.New("validation error line 10"),
					Node:            &yaml.Node{Line: 10, Column: 5},
				},
				errors.New("regular error"),
				errors.New("another regular error"),
			},
		},
		{
			name: "validation errors with zero line/column",
			errors: []error{
				&Error{
					UnderlyingError: errors.New("error at 0,0"),
					Node:            &yaml.Node{Line: 0, Column: 0},
				},
				&Error{
					UnderlyingError: errors.New("error at 1,1"),
					Node:            &yaml.Node{Line: 1, Column: 1},
				},
				&Error{
					UnderlyingError: errors.New("error at 0,5"),
					Node:            &yaml.Node{Line: 0, Column: 5},
				},
			},
			expected: []error{
				&Error{
					UnderlyingError: errors.New("error at 0,0"),
					Node:            &yaml.Node{Line: 0, Column: 0},
				},
				&Error{
					UnderlyingError: errors.New("error at 0,5"),
					Node:            &yaml.Node{Line: 0, Column: 5},
				},
				&Error{
					UnderlyingError: errors.New("error at 1,1"),
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
				expectedIsValidation := errors.As(expectedErr, &expectedValidationErr)
				actualIsValidation := errors.As(actualErr, &actualValidationErr)

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
				UnderlyingError: errors.New("valid error"),
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
				UnderlyingError: errors.New("error with negative line"),
				Node:            &yaml.Node{Line: -1, Column: 5},
			},
			&Error{
				UnderlyingError: errors.New("error with positive line"),
				Node:            &yaml.Node{Line: 1, Column: 5},
			},
			&Error{
				UnderlyingError: errors.New("error with negative column"),
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

	t.Run("same line and column sorted by error message", func(t *testing.T) {
		t.Parallel()

		errors := []error{
			&Error{
				UnderlyingError: errors.New("zzz error"),
				Node:            &yaml.Node{Line: 5, Column: 10},
			},
			&Error{
				UnderlyingError: errors.New("aaa error"),
				Node:            &yaml.Node{Line: 5, Column: 10},
			},
			&Error{
				UnderlyingError: errors.New("mmm error"),
				Node:            &yaml.Node{Line: 5, Column: 10},
			},
		}

		SortValidationErrors(errors)

		var err0, err1, err2 *Error
		require.ErrorAs(t, errors[0], &err0)
		require.ErrorAs(t, errors[1], &err1)
		require.ErrorAs(t, errors[2], &err2)
		assert.Equal(t, "aaa error", err0.UnderlyingError.Error())
		assert.Equal(t, "mmm error", err1.UnderlyingError.Error())
		assert.Equal(t, "zzz error", err2.UnderlyingError.Error())
	})

	t.Run("same line column and identical error message", func(t *testing.T) {
		t.Parallel()

		errors := []error{
			&Error{
				UnderlyingError: errors.New("same error"),
				Node:            &yaml.Node{Line: 5, Column: 10},
				Severity:        SeverityError,
			},
			&Error{
				UnderlyingError: errors.New("same error"),
				Node:            &yaml.Node{Line: 5, Column: 10},
				Severity:        SeverityWarning,
			},
		}

		SortValidationErrors(errors)

		// Both have same message so order should remain stable
		var err0, err1 *Error
		require.ErrorAs(t, errors[0], &err0)
		require.ErrorAs(t, errors[1], &err1)
		// Both should have the same message
		assert.Equal(t, "same error", err0.UnderlyingError.Error())
		assert.Equal(t, "same error", err1.UnderlyingError.Error())
		// Stable sort means first stays first
		assert.Equal(t, SeverityError, err0.Severity)
		assert.Equal(t, SeverityWarning, err1.Severity)
	})

	t.Run("interleaved regular and validation errors forces all comparison branches", func(t *testing.T) {
		t.Parallel()

		// Interleave regular and validation errors to force the sorting algorithm
		// to compare them in both directions (a=regular/b=validation AND a=validation/b=regular)
		errors := []error{
			errors.New("regular error 1"),
			&Error{
				UnderlyingError: errors.New("validation error 1"),
				Node:            &yaml.Node{Line: 10, Column: 5},
			},
			errors.New("regular error 2"),
			&Error{
				UnderlyingError: errors.New("validation error 2"),
				Node:            &yaml.Node{Line: 5, Column: 3},
			},
			errors.New("regular error 3"),
			&Error{
				UnderlyingError: errors.New("validation error 3"),
				Node:            &yaml.Node{Line: 15, Column: 7},
			},
			errors.New("regular error 4"),
		}

		SortValidationErrors(errors)

		// Validation errors should come first, sorted by line number
		var validationErr0, validationErr1, validationErr2 *Error
		require.ErrorAs(t, errors[0], &validationErr0)
		require.ErrorAs(t, errors[1], &validationErr1)
		require.ErrorAs(t, errors[2], &validationErr2)
		assert.Equal(t, 5, validationErr0.Node.Line, "first validation error should be line 5")
		assert.Equal(t, 10, validationErr1.Node.Line, "second validation error should be line 10")
		assert.Equal(t, 15, validationErr2.Node.Line, "third validation error should be line 15")

		// Regular errors should follow, preserving stable order
		var notValidation *Error
		assert.NotErrorAs(t, errors[3], &notValidation, "index 3 should be regular error")
		assert.NotErrorAs(t, errors[4], &notValidation, "index 4 should be regular error")
		assert.NotErrorAs(t, errors[5], &notValidation, "index 5 should be regular error")
		assert.NotErrorAs(t, errors[6], &notValidation, "index 6 should be regular error")
	})

	t.Run("validation errors with nil underlying errors", func(t *testing.T) {
		t.Parallel()

		errs := []error{
			&Error{
				UnderlyingError: nil,
				Node:            &yaml.Node{Line: 5, Column: 10},
			},
			&Error{
				UnderlyingError: errors.New("has message"),
				Node:            &yaml.Node{Line: 5, Column: 10},
			},
			&Error{
				UnderlyingError: nil,
				Node:            &yaml.Node{Line: 5, Column: 10},
			},
		}

		assert.NotPanics(t, func() {
			SortValidationErrors(errs)
		}, "sorting with nil UnderlyingError should not panic")

		// Error with message should sort after nil-message errors (empty string < "has message")
		var err0, err1, err2 *Error
		require.ErrorAs(t, errs[0], &err0)
		require.ErrorAs(t, errs[1], &err1)
		require.ErrorAs(t, errs[2], &err2)
		assert.NoError(t, err0.UnderlyingError, "nil error should sort first")
		assert.NoError(t, err1.UnderlyingError, "nil error should sort first")
		assert.Error(t, err2.UnderlyingError, "non-nil error should sort last")
	})

	t.Run("one nil and one non-nil underlying error", func(t *testing.T) {
		t.Parallel()

		errs := []error{
			&Error{
				UnderlyingError: errors.New("error B"),
				Node:            &yaml.Node{Line: 1, Column: 1},
			},
			&Error{
				UnderlyingError: nil,
				Node:            &yaml.Node{Line: 1, Column: 1},
			},
		}

		assert.NotPanics(t, func() {
			SortValidationErrors(errs)
		}, "sorting with one nil UnderlyingError should not panic")

		var err0, err1 *Error
		require.ErrorAs(t, errs[0], &err0)
		require.ErrorAs(t, errs[1], &err1)
		require.NoError(t, err0.UnderlyingError, "nil underlying error sorts before non-nil")
		require.Error(t, err1.UnderlyingError)
	})

	t.Run("validation errors first then regular errors forces bIsValidationErr", func(t *testing.T) {
		t.Parallel()

		// Start with validation errors, then regular errors
		// The merge sort should compare elements in the opposite direction during some phase
		errors := []error{
			&Error{
				UnderlyingError: errors.New("validation error 1"),
				Node:            &yaml.Node{Line: 20, Column: 10},
			},
			&Error{
				UnderlyingError: errors.New("validation error 2"),
				Node:            &yaml.Node{Line: 10, Column: 5},
			},
			errors.New("regular error 1"),
			errors.New("regular error 2"),
		}

		SortValidationErrors(errors)

		// Validation errors should come first, sorted by line
		var validationErr0, validationErr1 *Error
		require.ErrorAs(t, errors[0], &validationErr0)
		require.ErrorAs(t, errors[1], &validationErr1)
		assert.Equal(t, 10, validationErr0.Node.Line)
		assert.Equal(t, 20, validationErr1.Node.Line)
		// Regular errors follow
		assert.Equal(t, "regular error 1", errors[2].Error())
		assert.Equal(t, "regular error 2", errors[3].Error())
	})
}
