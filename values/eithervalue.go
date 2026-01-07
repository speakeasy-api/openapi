package values

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/values/core"
)

// EitherValue represents a union type that can hold either a Left or Right value.
// It provides multiple access patterns for different use cases:
//
// Direct field access (Left, Right) - for setting values
// Pointer access (GetLeft, GetRight) - for nil-safe pointer retrieval
// Value access (LeftValue, RightValue) - for nil-safe value retrieval with zero value fallback
type EitherValue[L any, LCore any, R any, RCore any] struct {
	marshaller.Model[core.EitherValue[LCore, RCore]]

	// Left holds the left-side value. Use directly when setting values in the EitherValue.
	Left *L
	// Right holds the right-side value. Use directly when setting values in the EitherValue.
	Right *R
}

// IsLeft returns true if the EitherValue contains a left value.
// Use this method to check which side of the union is active before accessing values.
func (e *EitherValue[L, LCore, R, RCore]) IsLeft() bool {
	if e == nil {
		return false
	}

	return e.Left != nil || e.Right == nil
}

// GetLeft returns a pointer to the left value in a nil-safe way.
// Returns nil if the EitherValue is nil or if no left value is set.
// Use this when you need a pointer to the left value or want to check for nil.
func (e *EitherValue[L, LCore, R, RCore]) GetLeft() *L {
	if e == nil {
		return nil
	}

	return e.Left
}

// LeftValue returns the left value directly, with zero value fallback for safety.
// Returns the zero value of type L if the EitherValue is nil or no left value is set.
// Use this when you need the actual value and want zero value fallback.
// Should typically be used in conjunction with IsLeft() to verify the value is valid.
func (e *EitherValue[L, LCore, R, RCore]) LeftValue() L {
	if e == nil || e.Left == nil {
		var zero L
		return zero
	}

	return *e.Left
}

// IsRight returns true if the EitherValue contains a right value.
// Use this method to check which side of the union is active before accessing values.
func (e *EitherValue[L, LCore, R, RCore]) IsRight() bool {
	if e == nil {
		return false
	}

	return e.Right != nil || e.Left == nil
}

// GetRight returns a pointer to the right value in a nil-safe way.
// Returns nil if the EitherValue is nil or if no right value is set.
// Use this when you need a pointer to the right value or want to check for nil.
func (e *EitherValue[L, LCore, R, RCore]) GetRight() *R {
	if e == nil {
		return nil
	}

	return e.Right
}

// RightValue returns the right value directly, with zero value fallback for safety.
// Returns the zero value of type R if the EitherValue is nil or no right value is set.
// Use this when you need the actual value and want zero value fallback.
// Should typically be used in conjunction with IsRight() to verify the value is valid.
func (e *EitherValue[L, LCore, R, RCore]) RightValue() R {
	if e == nil || e.Right == nil {
		var zero R
		return zero
	}

	return *e.Right
}

// PopulateWithContext populates the EitherValue with full population context.
func (e *EitherValue[L, LCore, R, RCore]) PopulateWithContext(source any, ctx *marshaller.PopulationContext) error {
	var ec *core.EitherValue[LCore, RCore]
	switch v := source.(type) {
	case *core.EitherValue[LCore, RCore]:
		ec = v
	case core.EitherValue[LCore, RCore]:
		ec = &v
	default:
		return fmt.Errorf("source is not an %T", &core.EitherValue[LCore, RCore]{})
	}

	// Set the core model from the source - this ensures RootNode is copied
	e.SetCoreAny(ec)

	if ec.IsLeft {
		if err := marshaller.PopulateWithContext(ec.Left, &e.Left, ctx); err != nil {
			return fmt.Errorf("failed to populate left: %w", err)
		}

		return nil
	}

	// Right value (typically bool for JSONSchema) doesn't need context
	if err := marshaller.PopulateWithContext(ec.Right, &e.Right, nil); err != nil {
		return fmt.Errorf("failed to populate right: %w", err)
	}

	return nil
}

// GetNavigableNode implements the NavigableNoder interface to return the held value for JSON pointer navigation
func (e *EitherValue[L, LCore, R, RCore]) GetNavigableNode() (any, error) {
	if e.Left != nil {
		return e.Left, nil
	}
	if e.Right != nil {
		return e.Right, nil
	}
	return nil, errors.New("EitherValue has no value set")
}

// IsEqual compares two EitherValue instances for equality.
// It attempts to use IsEqual methods on the contained values if they exist,
// falling back to reflect.DeepEqual otherwise.
func (e *EitherValue[L, LCore, R, RCore]) IsEqual(other *EitherValue[L, LCore, R, RCore]) bool {
	if e == nil && other == nil {
		return true
	}
	if e == nil || other == nil {
		return false
	}

	// Check if both are left or both are right
	if e.IsLeft() != other.IsLeft() {
		return false
	}

	if e.IsLeft() {
		return equalWithIsEqualMethod(e.Left, other.Left)
	}
	return equalWithIsEqualMethod(e.Right, other.Right)
}

var booleanType = reflect.TypeOf(true)

// equalWithIsEqualMethod attempts to use an IsEqual method if available,
// otherwise falls back to reflect.DeepEqual with special handling for empty/nil collections
func equalWithIsEqualMethod(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		// Special case: treat nil and empty slices/maps as equal
		if isEmptyCollection(a) && isEmptyCollection(b) {
			return true
		}
		return false
	}

	// Try to call IsEqual method using reflection
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	// Check if both values have an IsEqual method
	aMethod := aVal.MethodByName("IsEqual")
	if aMethod.IsValid() && aMethod.Type().NumIn() == 1 && aMethod.Type().NumOut() == 1 {
		// Check if the method signature matches: IsEqual(T) bool
		if aMethod.Type().In(0) == bVal.Type() && aMethod.Type().Out(0) == booleanType {
			result := aMethod.Call([]reflect.Value{bVal})
			if len(result) == 0 {
				return false
			}

			return result[0].Bool()
		}
	}

	// Special handling for slices and maps before falling back to reflect.DeepEqual
	if isEmptyCollection(a) && isEmptyCollection(b) {
		return true
	}

	// Fall back to reflect.DeepEqual
	return reflect.DeepEqual(a, b)
}

// isEmptyCollection checks if a value is nil or an empty slice/map
func isEmptyCollection(v any) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice, reflect.Map:
		return val.Len() == 0
	case reflect.Ptr:
		if val.IsNil() {
			return true
		}
		// Check if it points to an empty collection
		elem := val.Elem()
		switch elem.Kind() {
		case reflect.Slice, reflect.Map:
			return elem.Len() == 0
		}
	}

	return false
}
