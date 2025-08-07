// Package pointer provides utilities for working with pointers.
package pointer

// From will create a pointer from the provided value.
func From[T any](t T) *T {
	return &t
}

// Value will return the value of the pointer or the zero value if the pointer is nil.
func Value[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}
