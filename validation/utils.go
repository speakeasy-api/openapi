package validation

import (
	"errors"
	"slices"
)

// SortValidationErrors sorts the provided validation errors by line and column number lowest to highest.
func SortValidationErrors(allErrors []error) {
	slices.SortStableFunc(allErrors, func(a, b error) int {
		var aValidationErr *Error
		var bValidationErr *Error
		aIsValidationErr := errors.As(a, &aValidationErr)
		bIsValidationErr := errors.As(b, &bValidationErr)
		switch {
		case aIsValidationErr && bIsValidationErr:
			if aValidationErr.GetLineNumber() == bValidationErr.GetLineNumber() {
				if aValidationErr.GetColumnNumber() == bValidationErr.GetColumnNumber() {
					// When line and column are the same, sort by error message
					if a.Error() < b.Error() {
						return -1
					} else if a.Error() > b.Error() {
						return 1
					}
					return 0
				}
				return aValidationErr.GetColumnNumber() - bValidationErr.GetColumnNumber()
			}
			return aValidationErr.GetLineNumber() - bValidationErr.GetLineNumber()
		case aIsValidationErr:
			return -1
		case bIsValidationErr:
			return 1
		default:
			return 0
		}
	})
}
