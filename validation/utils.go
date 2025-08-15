package validation

import (
	"errors"
	"slices"
)

// SortValidationErrors sorts the provided validation errors by line and column number lowest to highest.
func SortValidationErrors(allErrors []error) {
	slices.SortFunc(allErrors, func(a, b error) int {
		var aValidationErr *Error
		var bValidationErr *Error
		aIsValidationErr := errors.As(a, &aValidationErr)
		bIsValidationErr := errors.As(b, &bValidationErr)
		switch {
		case aIsValidationErr && bIsValidationErr:
			if aValidationErr.GetLineNumber() == bValidationErr.GetLineNumber() {
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
