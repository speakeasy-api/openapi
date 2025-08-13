package marshaller

import (
	"reflect"
	"sync"

	"go.yaml.in/yaml/v4"
)

// CoreAccessor provides type-safe access to the core field in models
type CoreAccessor[T any] interface {
	GetCore() *T
	SetCore(core *T)
}

// CoreSetter provides runtime access to set the core field
type CoreSetter interface {
	SetCoreAny(core any)
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

	objectCache   *sync.Map
	documentCache *sync.Map
}

// GetCore will return the low level representation of the model.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (m *Model[T]) GetCore() *T {
	if m == nil {
		return nil
	}

	return &m.core
}

// GetCoreAny will return the low level representation of the model untyped.
// Useful for using with interfaces and reflection.
func (m *Model[T]) GetCoreAny() any {
	if m == nil {
		return nil
	}

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
	if m == nil {
		return nil
	}

	if coreModeler, ok := any(&m.core).(CoreModeler); ok {
		return coreModeler.GetRootNode()
	}
	return nil
}

func (m *Model[T]) GetRootNodeLine() int {
	if rootNode := m.GetRootNode(); rootNode != nil {
		return rootNode.Line
	}
	return -1
}

func (m *Model[T]) GetRootNodeColumn() int {
	if rootNode := m.GetRootNode(); rootNode != nil {
		return rootNode.Column
	}
	return -1
}

func (m *Model[T]) GetPropertyLine(prop string) int {
	// Use reflection to find the property in the core and then see if it is a marshaller.Node and if it is get the line of the key node if set
	if m == nil {
		return -1
	}

	// Get reflection value of the core
	coreValue := reflect.ValueOf(&m.core).Elem()
	if !coreValue.IsValid() {
		return -1
	}

	// Find the field by name
	fieldValue := coreValue.FieldByName(prop)
	if !fieldValue.IsValid() {
		return -1
	}

	// Check if the field implements the interface we need to get the key node
	// We need to check if it has a GetKeyNode method or if it's a Node type
	fieldInterface := fieldValue.Interface()

	// Try to cast to a Node-like interface that has GetKeyNode method
	if nodeWithKeyNode, ok := fieldInterface.(interface{ GetKeyNode() *yaml.Node }); ok {
		keyNode := nodeWithKeyNode.GetKeyNode()
		if keyNode != nil {
			return keyNode.Line
		}
	}

	return -1
}

// SetCore implements CoreAccessor interface
func (m *Model[T]) SetCore(core *T) {
	if core != nil {
		m.core = *core
	}
}

// SetCoreAny implements CoreSetter interface
func (m *Model[T]) SetCoreAny(core any) {
	if coreVal, ok := core.(*T); ok {
		m.core = *coreVal
	} else if coreVal, ok := core.(T); ok {
		m.core = coreVal
	}
}

func (m *Model[T]) GetCachedReferencedObject(key string) (any, bool) {
	if m.objectCache == nil {
		return nil, false
	}
	return m.objectCache.Load(key)
}

func (m *Model[T]) StoreReferencedObjectInCache(key string, obj any) {
	m.objectCache.Store(key, obj)
}

func (m *Model[T]) GetCachedReferenceDocument(key string) ([]byte, bool) {
	if m.documentCache == nil {
		return nil, false
	}
	value, ok := m.documentCache.Load(key)
	if !ok {
		return nil, false
	}
	doc, ok := value.([]byte)
	return doc, ok
}

func (m *Model[T]) StoreReferenceDocumentInCache(key string, doc []byte) {
	m.documentCache.Store(key, doc)
}

func (m *Model[T]) InitCache() {
	if m.objectCache == nil {
		m.objectCache = &sync.Map{}
	}
	if m.documentCache == nil {
		m.documentCache = &sync.Map{}
	}
}
