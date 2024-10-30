// Package pointer provides utilities for working with pointers.
package pointer

// From will create a pointer to the provided value.
func From[T any](t T) *T {
	return &t
}

// ValueOrZero will return the value of the pointer or the zero value if the pointer is nil.
func ValueOrZero[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}

	return *v
}
