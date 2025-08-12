package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
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

	// Try Left type without strict mode
	leftValidationErrs, leftUnmarshalErr = marshaller.UnmarshalCore(ctx, parentName, node, &v.Left)
	if leftUnmarshalErr == nil && !hasTypeMismatchErrors(leftValidationErrs) {
		// No unmarshalling error and no type mismatch validation errors - this is successful
		v.IsLeft = true
		v.SetRootNode(node)
		return leftValidationErrs, nil
	}

	// Try Right type without strict mode
	rightValidationErrs, rightUnmarshalErr = marshaller.UnmarshalCore(ctx, parentName, node, &v.Right)
	if rightUnmarshalErr == nil && !hasTypeMismatchErrors(rightValidationErrs) {
		// No unmarshalling error and no type mismatch validation errors - this is successful
		v.IsRight = true
		v.SetRootNode(node)
		return rightValidationErrs, nil
	}

	// Both types failed - determine if we should return validation errors or unmarshalling errors
	if leftUnmarshalErr == nil && rightUnmarshalErr == nil {
		// Both failed with validation errors only (no real unmarshalling errors)
		// Combine the validation errors and return them instead of an error
		allValidationErrs := append(leftValidationErrs, rightValidationErrs...)
		return allValidationErrs, nil
	}

	// At least one had a real unmarshalling error - return as unmarshalling failure
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

	return nil, fmt.Errorf("unable to marshal into either %s or %s: %w", reflect.TypeOf((*L)(nil)).Elem().Name(), reflect.TypeOf((*R)(nil)).Elem().Name(), errors.Join(errs...))
}

// hasTypeMismatchErrors checks if the validation errors contain type mismatch errors
// indicating that the type couldn't be unmarshalled successfully
func hasTypeMismatchErrors(validationErrs []error) bool {
	if len(validationErrs) == 0 {
		return false
	}

	for _, err := range validationErrs {
		// Check if this is a type mismatch error by looking for common patterns
		errStr := err.Error()
		if strings.Contains(errStr, "expected") && (strings.Contains(errStr, "got") || strings.Contains(errStr, "but received")) {
			return true
		}
		if strings.Contains(errStr, "type mismatch") || strings.Contains(errStr, "cannot unmarshal") {
			return true
		}
	}

	return false
}

func (v *EitherValue[L, R]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", mv.Kind())
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
	return nil, fmt.Errorf("EitherValue has neither Left nor Right set")
}

func (v *EitherValue[L, R]) GetNavigableNode() (any, error) {
	if v.IsLeft {
		return v.Left, nil
	}
	return v.Right, nil
}
