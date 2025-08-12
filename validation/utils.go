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
		if aIsValidationErr && bIsValidationErr {
			if aValidationErr.GetLineNumber() == bValidationErr.GetLineNumber() {
				return aValidationErr.GetColumnNumber() - bValidationErr.GetColumnNumber()
			}
			return aValidationErr.GetLineNumber() - bValidationErr.GetLineNumber()
		} else if aIsValidationErr {
			return -1
		} else if bIsValidationErr {
			return 1
		}

		return 0
	})
}
