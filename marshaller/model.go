package marshaller

// CoreAccessor provides type-safe access to the core field in models
type CoreAccessor[T any] interface {
	GetCore() *T
	SetCore(core *T)
}

// CoreSetter provides runtime access to set the core field
type CoreSetter interface {
	SetCoreValue(core any)
}

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

// SetCore implements CoreAccessor interface
func (m *Model[T]) SetCore(core *T) {
	if core != nil {
		m.core = *core
	}
}

// SetCoreValue implements CoreSetter interface
func (m *Model[T]) SetCoreValue(core any) {
	if coreVal, ok := core.(*T); ok {
		m.core = *coreVal
	} else if coreVal, ok := core.(T); ok {
		m.core = coreVal
	}
}
