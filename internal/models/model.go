package models

// Model is a generic model that can be used to validate and marshal/unmarshal a model.
type Model[T any] struct {
	// Valid indicates whether this model passed validation.
	Valid bool
	core  T
}

// GetCore will return the low level representation of the model.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (m *Model[T]) GetCore() *T {
	return &m.core
}
