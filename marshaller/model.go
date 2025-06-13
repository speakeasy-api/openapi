package marshaller

import (
	"gopkg.in/yaml.v3"
)

// CoreAccessor provides type-safe access to the core field in models
type CoreAccessor[T any] interface {
	GetCore() *T
	SetCore(core *T)
}

// CoreSetter provides runtime access to set the core field
type CoreSetter interface {
	SetCoreValue(core any)
}

// RootNodeAccessor provides access to the RootNode of a model's core for identity matching.
//
// This interface solves a critical problem in array/map synchronization: when high-level
// arrays are reordered (e.g., moving workflows around in an Arazzo document), we need to
// match each high-level element with its corresponding core model to preserve field ordering
// and other state.
//
// Without identity matching, the sync process would match elements by array position:
//
//	Source[0] -> Target[0], Source[1] -> Target[1], etc.
//
// This causes problems when arrays are reordered because the wrong data gets synced to
// the wrong core objects, disrupting field ordering within individual elements.
//
// With RootNode identity matching, we can match elements correctly:
//
//	Source[workflow-A] -> Target[core-for-workflow-A] (regardless of position)
//	Source[workflow-B] -> Target[core-for-workflow-B] (regardless of position)
//
// The RootNode serves as a unique identity because it's the original YAML node that was
// parsed for each element, making it a stable identifier across reorderings.
type RootNodeAccessor interface {
	GetRootNode() *yaml.Node
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

// GetRootNode implements RootNodeAccessor interface by delegating to the core model.
//
// This method provides access to the unique YAML node that was originally parsed for this
// model instance. The RootNode serves as a stable identity for the model that persists
// across array reorderings and other operations.
//
// The method works by checking if the core model implements CoreModeler interface, which
// provides access to the RootNode. If the core doesn't implement CoreModeler, this returns
// nil, which causes the sync process to fall back to index-based matching.
//
// This identity-based matching is crucial for preserving field ordering when high-level
// arrays are reordered, as it ensures each high-level model syncs with its correct
// corresponding core model rather than being matched by array position.
func (m *Model[T]) GetRootNode() *yaml.Node {
	if coreModeler, ok := any(&m.core).(CoreModeler); ok {
		return coreModeler.GetRootNode()
	}
	return nil
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
