package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type EitherValue[L any, R any] struct {
	marshaller.CoreModel `model:"eitherValue"`

	Left   marshaller.Node[L]
	IsLeft bool

	Right   marshaller.Node[R]
	IsRight bool
}

var _ interfaces.CoreModel = (*EitherValue[any, any])(nil)

func (v *EitherValue[L, R]) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	var leftUnmarshalErr error
	var leftValidationErrs []error
	var rightUnmarshalErr error
	var rightValidationErrs []error

	v.SetRootNode(node)

	// Try Left type without strict mode
	leftValidationErrs, leftUnmarshalErr = marshaller.UnmarshalCore(ctx, parentName, node, &v.Left)
	if leftUnmarshalErr == nil && !hasTypeMismatchErrors(parentName, leftValidationErrs) {
		// No unmarshaling error and no type mismatch validation errors - this is successful
		v.IsLeft = true
		return leftValidationErrs, nil
	}

	// Try Right type without strict mode
	rightValidationErrs, rightUnmarshalErr = marshaller.UnmarshalCore(ctx, parentName, node, &v.Right)
	if rightUnmarshalErr == nil && !hasTypeMismatchErrors(parentName, rightValidationErrs) {
		// No unmarshaling error and no type mismatch validation errors - this is successful
		v.IsRight = true
		return rightValidationErrs, nil
	}

	leftType := typeToName[L]()
	rightType := typeToName[R]()

	// Both types failed - determine if we should return validation errors or unmarshaling errors
	// Both failed with validation errors only (no real unmarshaling errors)
	if leftUnmarshalErr == nil && rightUnmarshalErr == nil {
		// Filter out child errors from both left and right validation errors
		leftParentErrs, leftChildErrs := filterChildErrors(parentName, leftValidationErrs)
		rightParentErrs, rightChildErrs := filterChildErrors(parentName, rightValidationErrs)

		// Combine parent-level validation errors for the error message
		allParentErrs := make([]error, 0, len(leftParentErrs)+len(rightParentErrs))
		allParentErrs = append(allParentErrs, leftParentErrs...)
		allParentErrs = append(allParentErrs, rightParentErrs...)

		msg := fmt.Sprintf("failed to validate either %s [%s] or %s [%s]", leftType, getUnwrappedErrors(leftParentErrs), rightType, getUnwrappedErrors(rightParentErrs))

		var validationError error
		if hasTypeMismatchErrors(parentName, allParentErrs) {
			validationError = validation.NewTypeMismatchError(parentName, msg)
		} else {
			name := parentName
			if name != "" {
				name += " "
			}

			validationError = fmt.Errorf("%s%s", name, msg)
		}

		// Get severity and rule from the worst error
		severity, rule := getWorstSeverityAndRule(allParentErrs)

		// Return the validation error along with all child errors separately
		result := []error{validation.NewValidationError(severity, rule, validationError, node)}
		result = append(result, leftChildErrs...)
		result = append(result, rightChildErrs...)

		return result, nil
	}

	// At least one had a real unmarshaling error - return as unmarshaling failure
	errs := []error{}
	if leftUnmarshalErr != nil {
		errs = append(errs, leftUnmarshalErr)
	} else {
		errs = append(errs, fmt.Errorf("left type validation failed: %v", leftValidationErrs))
	}

	if rightUnmarshalErr != nil {
		errs = append(errs, rightUnmarshalErr)
	} else {
		errs = append(errs, fmt.Errorf("right type validation failed: %v", rightValidationErrs))
	}

	return nil, fmt.Errorf("unable to marshal into either %s or %s: %w", leftType, rightType, errors.Join(errs...))
}

// isParentError checks if an error belongs to the current parentName level
func isParentError(parentName string, err error) bool {
	var typeMismatchErr *validation.TypeMismatchError
	if !errors.As(err, &typeMismatchErr) {
		return true // Non-type-mismatch errors are considered parent errors
	}

	return typeMismatchErr.ParentName == parentName
}

// hasTypeMismatchErrors checks if the validation errors contain type mismatch errors
// indicating that the type couldn't be unmarshaled successfully.
// It ignores type mismatch errors from child properties to avoid cascading failures.
func hasTypeMismatchErrors(parentName string, validationErrs []error) bool {
	for _, err := range validationErrs {
		// Check if it's a TypeMismatchError (isParentError returns true for non-TypeMismatchErrors)
		var typeMismatchErr *validation.TypeMismatchError
		if !errors.As(err, &typeMismatchErr) {
			continue
		}

		// Check if it's at the parent level
		if !isParentError(parentName, err) {
			continue
		}

		return true
	}

	return false
}

// filterChildErrors separates child errors from parent errors based on parentName
func filterChildErrors(parentName string, validationErrs []error) (parentErrs []error, childErrs []error) {
	for _, err := range validationErrs {
		if isParentError(parentName, err) {
			parentErrs = append(parentErrs, err)
		} else {
			childErrs = append(childErrs, err)
		}
	}
	return parentErrs, childErrs
}

func (v *EitherValue[L, R]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected `struct`, got `%s`", mv.Kind())
	}

	lf := mv.FieldByName("Left")
	rf := mv.FieldByName("Right")

	// Check which side is active in the high-level model
	leftIsNil := lf.IsNil()
	rightIsNil := rf.IsNil()

	// Track the original state to detect side switches
	originalIsLeft := v.IsLeft
	originalIsRight := v.IsRight

	// Detect if we're switching sides
	switchingSides := false
	if !leftIsNil && originalIsRight {
		// Switching from Right to Left
		switchingSides = true
	} else if !rightIsNil && originalIsLeft {
		// Switching from Left to Right
		switchingSides = true
	}

	// Determine which valueNode to use
	var nodeToUse *yaml.Node
	if switchingSides {
		// Force creation of new node when switching sides
		// This prevents reusing the old node structure which may be incompatible
		nodeToUse = nil
	} else {
		nodeToUse = valueNode
	}

	// Reset flags
	v.IsLeft = false
	v.IsRight = false

	if !leftIsNil {
		// Left is active - sync left value and set flag
		lv, err := marshaller.SyncValue(ctx, lf.Interface(), &v.Left.Value, nodeToUse, false)
		if err != nil {
			return nil, err
		}
		v.IsLeft = true
		v.SetRootNode(lv)
		return lv, nil
	} else if !rightIsNil {
		// Right is active - sync right value and set flag
		rv, err := marshaller.SyncValue(ctx, rf.Interface(), &v.Right.Value, nodeToUse, false)
		if err != nil {
			return nil, err
		}

		v.IsRight = true
		v.SetRootNode(rv)
		return rv, nil
	}

	// Both are nil - this shouldn't happen in a valid EitherValue, but handle gracefully
	return nil, errors.New("EitherValue has neither Left nor Right set")
}

func (v *EitherValue[L, R]) GetNavigableNode() (any, error) {
	if v.IsLeft {
		return v.Left, nil
	}
	return v.Right, nil
}

func getUnwrappedErrors(errs []error) string {
	var unwrappedErrs []string
	for _, err := range errs {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			unwrapped = err
		}

		unwrappedErrs = append(unwrappedErrs, unwrapped.Error())
	}
	return strings.Join(unwrappedErrs, ", ")
}

func typeToName[T any]() string {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	name := typ.Name()
	if name == "" {
		switch typ.Kind() {
		case reflect.Slice, reflect.Array:
			name = "sequence"
		case reflect.Map, reflect.Struct:
			name = "object"
		}
	}

	return name
}

// getWorstSeverityAndRule finds the worst severity and its first rule from a list of errors.
// Severity order (worst to best): error > warning > hint
// Returns the severity and rule of the first error with the worst severity.
// If no validation errors are found, returns SeverityError and RuleValidationTypeMismatch as defaults.
func getWorstSeverityAndRule(errs []error) (validation.Severity, string) {
	var worstSeverity validation.Severity
	var worstRule string
	worstSeverityRank := -1 // -1 means no validation error found yet

	for _, err := range errs {
		var validationErr *validation.Error
		if !errors.As(err, &validationErr) {
			continue
		}

		rank := validationErr.Severity.Rank()
		if rank > worstSeverityRank {
			worstSeverityRank = rank
			worstSeverity = validationErr.Severity
			worstRule = validationErr.Rule
		}
	}

	// Default to error severity and type mismatch rule if no validation errors found
	if worstSeverityRank == -1 {
		return validation.SeverityError, validation.RuleValidationTypeMismatch
	}

	return worstSeverity, worstRule
}
