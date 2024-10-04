// Package pointer provides utilities for working with pointers.
package pointer

// From will create a pointer to the provided value.
func From[T any](t T) *T {
	return &t
}
