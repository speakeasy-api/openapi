// Package sequencedmap provides a map implementation that maintains the order of keys as they are added.
package sequencedmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"reflect"
	"slices"
)

// Element is a key-value pair that is stored in a sequenced map.
type Element[K comparable, V any] struct {
	Key   K
	Value V
}

// NewElem creates a new element with the specified key and value.
func NewElem[K comparable, V any](key K, value V) *Element[K, V] {
	return &Element[K, V]{
		Key:   key,
		Value: value,
	}
}

// Map is a map implementation that maintains the order of keys as they are added.
type Map[K comparable, V any] struct {
	m map[K]*Element[K, V]
	l []*Element[K, V]
}

// New creates a new map with the specified elements.
func New[K comparable, V any](elements ...*Element[K, V]) *Map[K, V] {
	return new(-1, elements...)
}

// NewWithCapacity creates a new map with the specified capacity and elements.
func NewWithCapacity[K comparable, V any](capacity int, elements ...*Element[K, V]) *Map[K, V] {
	return new(capacity, elements...)
}

func new[K comparable, V any](capacity int, elements ...*Element[K, V]) *Map[K, V] {
	if len(elements) > capacity && capacity > 0 {
		capacity = len(elements)
	}

	var internalMap map[K]*Element[K, V]
	if capacity > 0 {
		internalMap = make(map[K]*Element[K, V], capacity)
	} else {
		internalMap = make(map[K]*Element[K, V])
	}

	var internalList []*Element[K, V]
	if capacity > 0 {
		internalList = make([]*Element[K, V], 0, capacity)
	} else {
		internalList = make([]*Element[K, V], 0)
	}

	m := &Map[K, V]{
		m: internalMap,
		l: internalList,
	}

	for _, element := range elements {
		m.m[element.Key] = element
		m.l = append(m.l, element)
	}

	return m
}

// Init initializes the underlying resources of the map.
func (m *Map[K, V]) Init() {
	if m.m == nil && m.l == nil {
		m.m = make(map[K]*Element[K, V])
		m.l = make([]*Element[K, V], 0)
	}
}

// Len returns the number of elements in the map. nil safe.
func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.l)
}

// Set sets the value for the specified key.
func (m *Map[K, V]) Set(key K, value V) {
	element := &Element[K, V]{
		Key:   key,
		Value: value,
	}

	// Check if key already exists
	if existingElement, exists := m.m[key]; exists {
		// Update existing element in place
		existingElement.Value = value
	} else {
		// Add new element
		m.m[key] = element
		m.l = append(m.l, element)
	}
}

// Set with any type
func (m *Map[K, V]) SetAny(key, value any) {
	k, ok := key.(K)
	if !ok {
		return // silently ignore type mismatches
	}
	v, ok := value.(V)
	if !ok {
		return // silently ignore type mismatches
	}
	m.Set(k, v)
}

// Get with any type
func (m *Map[K, V]) GetAny(key any) (any, bool) {
	k, ok := key.(K)
	if !ok {
		return nil, false
	}
	v, found := m.Get(k)
	return v, found
}

// Delete with any type
func (m *Map[K, V]) DeleteAny(key any) {
	k, ok := key.(K)
	if !ok {
		return // silently ignore type mismatches
	}
	m.Delete(k)
}

// Keys with any type
func (m *Map[K, V]) KeysAny() iter.Seq[any] {
	return func(yield func(any) bool) {
		if m == nil {
			return
		}

		for _, element := range m.l {
			if !yield(element.Key) {
				return
			}
		}
	}
}

// SetUntyped sets the value for the specified key with untyped key and value.
// This allows for using the map in generic code.
// An error is returned if the key or value is not of the correct type.
func (m *Map[K, V]) SetUntyped(key, value any) error {
	k, ok := key.(K)
	if !ok {
		var zeroK K
		return fmt.Errorf("expected key to be of type %T, got %T (value: %v)", zeroK, key, key)
	}

	v, ok := value.(V)
	if !ok {
		var zeroV V
		return fmt.Errorf("expected value to be of type %T, got %T (value: %v)", zeroV, value, value)
	}

	m.Set(k, v)

	return nil
}

// Get returns the value for the specified key and a boolean indicating whether the key was found.
func (m *Map[K, V]) Get(key K) (V, bool) {
	var zero V
	if m == nil {
		return zero, false
	}

	element, ok := m.m[key]
	if !ok {
		return zero, false
	}

	return element.Value, true
}

// GetUntyped returns the untyped value for the specified key with untyped key and a boolean indicating whether the key was found.
// This allows for using the map in generic code.
// If they key is not of the correct type, the zero value is returned.
func (m *Map[K, V]) GetUntyped(key any) (any, bool) {
	var zero V
	if m == nil {
		return zero, false
	}

	k, ok := key.(K)
	if !ok {
		return zero, false
	}

	element, ok := m.m[k]
	if !ok {
		return zero, false
	}

	return element.Value, true
}

// GetOrZero returns the value for the specified key or the zero value if the key is not found.
func (m *Map[K, V]) GetOrZero(key K) V {
	var zero V
	if m == nil {
		return zero
	}

	element, ok := m.m[key]
	if !ok {
		return zero
	}

	return element.Value
}

// Has returns a boolean indicating whether the map contains the specified key.
func (m *Map[K, V]) Has(key K) bool {
	if m == nil {
		return false
	}

	_, ok := m.m[key]
	return ok
}

// Delete removes the element with the specified key from the map.
func (m *Map[K, V]) Delete(key K) {
	if m == nil {
		return
	}

	delete(m.m, key)

	i := slices.IndexFunc(m.l, func(e *Element[K, V]) bool {
		return e.Key == key
	})

	if i >= 0 {
		m.l = slices.Delete(m.l, i, i+1)
	}
}

// All returns an iterator that iterates over all elements in the map, in the order they were added.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil {
			return
		}

		for _, element := range m.l {
			if !yield(element.Key, element.Value) {
				return
			}
		}
	}
}

// AllUntyped returns an iterator that iterates over all elements in the map with untyped key and value.
// This allows for using the map in generic code.
func (m *Map[K, V]) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {
		if m == nil {
			return
		}

		for _, element := range m.l {
			if !yield(element.Key, element.Value) {
				return
			}
		}
	}
}

// Keys returns an iterator that iterates over all keys in the map, in the order they were added.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if m == nil {
			return
		}

		for _, element := range m.l {
			if !yield(element.Key) {
				return
			}
		}
	}
}

// Values returns an iterator that iterates over all values in the map, in the order they were added.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		if m == nil {
			return
		}

		for _, element := range m.l {
			if !yield(element.Value) {
				return
			}
		}
	}
}

// GetKeyType returns the type of the keys in the map.
func (m *Map[K, V]) GetKeyType() reflect.Type {
	var zero K
	return reflect.TypeOf(zero)
}

// GetValueType returns the type of the values in the map.
func (m *Map[K, V]) GetValueType() reflect.Type {
	var zero V
	return reflect.TypeOf(zero)
}

// NavigateWithKey returns the value for the specified key with the key as a string.
// This is an implementation of the jsonpointer.KeyNavigable interface.
func (m *Map[K, V]) NavigateWithKey(key string) (any, error) {
	if m == nil {
		return nil, fmt.Errorf("sequencedmap.Map is nil")
	}

	keyType := reflect.TypeOf((*K)(nil)).Elem()

	if reflect.TypeOf((*K)(nil)).Elem().Kind() != reflect.String {
		return nil, fmt.Errorf("sequencedmap.Map key type must be string")
	}

	var ka any = key
	k, ok := ka.(K)
	if !ok {
		return nil, fmt.Errorf("key not convertible to sequencedmap.Map key type %v", keyType)
	}

	v, ok := m.Get(k)
	if !ok {
		return nil, fmt.Errorf("key %v not found in sequencedmap.Map", k)
	}

	return v, nil
}

// MarshalJSON returns the JSON representation of the map.
func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}

	// TODO there might be a more efficient way to serialize this but this is fine for now
	var buf bytes.Buffer

	buf.WriteString("{")

	for i, element := range m.l {
		ks := fmt.Sprintf("%v", element.Key)
		kb, err := json.Marshal(ks)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteString(":")
		vb, err := json.Marshal(element.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(vb)

		if i < len(m.l)-1 {
			buf.WriteString(",")
		}
	}

	buf.WriteString("}")

	return buf.Bytes(), nil
}
