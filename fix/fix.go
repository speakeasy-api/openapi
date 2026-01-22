package fix

import (
	"fmt"
)

// Fix represents a suggested fix for a lint finding
type Fix[T any] struct {
	// Description describes what the fix does
	Description string

	// ApplyFunc is the function that applies the fix
	ApplyFunc func(doc T) error
}

func (f Fix[T]) Apply(doc any) error {
	tDoc, ok := doc.(T)
	if !ok {
		return fmt.Errorf("invalid document type: expected %T, got %T", *new(T), doc)
	}
	if f.ApplyFunc != nil {
		return f.ApplyFunc(tDoc)
	}
	return nil
}

func (f Fix[T]) FixDescription() string {
	return f.Description
}
