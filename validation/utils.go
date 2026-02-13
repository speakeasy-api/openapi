package validation

import (
	"errors"
	"slices"
)

// SortValidationErrors sorts the provided validation errors by line and column number lowest to highest.
// It pre-partitions errors by type to avoid O(n log n) errors.As() calls in the sort comparator.
func SortValidationErrors(allErrors []error) {
	if len(allErrors) == 0 {
		return
	}

	// Single-pass partition: separate *Error (direct or wrapped) from non-validation errors.
	var validErrs []*Error
	var otherIdxs []int // indices of non-validation errors in the original slice
	for i, err := range allErrors {
		var vErr *Error
		if errors.As(err, &vErr) {
			validErrs = append(validErrs, vErr)
		} else {
			otherIdxs = append(otherIdxs, i)
		}
	}

	// Sort validation errors with direct field access — no errors.As or Error() needed.
	slices.SortStableFunc(validErrs, compareValidationErrors)

	// Save non-validation errors before reconstruction (origIdx slots may be overwritten).
	otherErrs := make([]error, len(otherIdxs))
	for i, origIdx := range otherIdxs {
		otherErrs[i] = allErrors[origIdx]
	}

	// Reconstruct: validation errors first (sorted), then non-validation errors (original order).
	idx := 0
	for _, vErr := range validErrs {
		allErrors[idx] = vErr
		idx++
	}
	for _, err := range otherErrs {
		allErrors[idx] = err
		idx++
	}
}

// compareValidationErrors compares two validation errors by line, column, severity, rule,
// underlying message, and document location.
func compareValidationErrors(a, b *Error) int {
	if a.GetLineNumber() != b.GetLineNumber() {
		return a.GetLineNumber() - b.GetLineNumber()
	}
	if a.GetColumnNumber() != b.GetColumnNumber() {
		return a.GetColumnNumber() - b.GetColumnNumber()
	}
	if a.Severity != b.Severity {
		if a.Severity < b.Severity {
			return -1
		}
		return 1
	}
	if a.Rule != b.Rule {
		if a.Rule < b.Rule {
			return -1
		}
		return 1
	}
	aMsg := a.UnderlyingError.Error()
	bMsg := b.UnderlyingError.Error()
	if aMsg != bMsg {
		if aMsg < bMsg {
			return -1
		}
		return 1
	}
	if a.DocumentLocation != b.DocumentLocation {
		if a.DocumentLocation < b.DocumentLocation {
			return -1
		}
		return 1
	}
	return 0
}
