// Package sequencedmap provides a map implementation that maintains the order of keys as they are added.
package sequencedmap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"sort"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

// OrderType represents the different ways to order iteration through the map
type OrderType int

const (
	// OrderAdded iterates in the order items were added (default behavior)
	OrderAdded OrderType = iota
	// OrderAddedReverse iterates in reverse order of when items were added
	OrderAddedReverse
	// OrderKeyAsc iterates with keys in alphabetical ascending order
	OrderKeyAsc
	// OrderKeyDesc iterates with keys in alphabetical descending order
	OrderKeyDesc
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

// GetKey returns the key of the element. If the element is nil, the zero value of the key type is returned.
func (e *Element[K, V]) GetKey() K {
	if e == nil {
		var zero K
		return zero
	}
	return e.Key
}

// GetValue returns the value of the element. If the element is nil, the zero value of the value type is returned.
func (e *Element[K, V]) GetValue() V {
	if e == nil {
		var zero V
		return zero
	}
	return e.Value
}

// Map is a map implementation that maintains the order of keys as they are added.
type Map[K comparable, V any] struct {
	m map[K]*Element[K, V]
	l []*Element[K, V]
}

var _ interfaces.SequencedMapInterface = (*Map[any, any])(nil)

// New creates a new map with the specified elements.
func New[K comparable, V any](elements ...*Element[K, V]) *Map[K, V] {
	return newMap(-1, elements...)
}

// NewWithCapacity creates a new map with the specified capacity and elements.
func NewWithCapacity[K comparable, V any](capacity int, elements ...*Element[K, V]) *Map[K, V] {
	return newMap(capacity, elements...)
}

func newMap[K comparable, V any](capacity int, elements ...*Element[K, V]) *Map[K, V] {
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

// IsInitialized returns true if the map has been initialized.
func (m *Map[K, V]) IsInitialized() bool {
	if m == nil {
		return false
	}
	return m.m != nil && m.l != nil
}

// Len returns the number of elements in the map. nil safe.
func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.l)
}

// Set sets the value for the specified key. If the key does not exist, it is added to the end of the list.
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

// Add adds the specified key-value pair to the map. If the key already exists, it is moved to the end of the list.
func (m *Map[K, V]) Add(key K, value V) {
	element := &Element[K, V]{
		Key:   key,
		Value: value,
	}

	// Check if key already exists
	if existingElement, exists := m.m[key]; exists {
		// Move existing element to the end of the list
		index := slices.Index(m.l, existingElement)
		if index >= 0 {
			m.l = slices.Delete(m.l, index, index+1)
		}
		m.m[key] = element
		m.l = append(m.l, element)
	} else {
		// Add new element
		m.m[key] = element
		m.l = append(m.l, element)
	}
}

// SetAny Set with any type
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

// AddAny Add with any type
func (m *Map[K, V]) AddAny(key, value any) {
	k, ok := key.(K)
	if !ok {
		return // silently ignore type mismatches
	}
	v, ok := value.(V)
	if !ok {
		return // silently ignore type mismatches
	}
	m.Add(k, v)
}

// GetAny Get with any type
func (m *Map[K, V]) GetAny(key any) (any, bool) {
	k, ok := key.(K)
	if !ok {
		return nil, false
	}
	v, found := m.Get(k)
	return v, found
}

// DeleteAny Delete with any type
func (m *Map[K, V]) DeleteAny(key any) {
	k, ok := key.(K)
	if !ok {
		return // silently ignore type mismatches
	}
	m.Delete(k)
}

// KeysAny Keys with any type
// The iterator is safe for concurrent mutations - it creates a snapshot at the start of iteration.
func (m *Map[K, V]) KeysAny() iter.Seq[any] {
	return func(yield func(any) bool) {
		if m == nil {
			return
		}

		// Create a snapshot of the current elements to avoid issues with concurrent modifications
		snapshot := make([]*Element[K, V], len(m.l))
		copy(snapshot, m.l)

		for _, element := range snapshot {
			// Check if element still exists in the map (it might have been deleted during iteration)
			if _, exists := m.m[element.Key]; exists {
				if !yield(element.Key) {
					return
				}
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

// First returns the first element in the map.
func (m *Map[K, V]) First() *Element[K, V] {
	if m == nil || len(m.l) == 0 {
		return nil
	}

	return m.l[0]
}

// Last returns the last element in the map.
func (m *Map[K, V]) Last() *Element[K, V] {
	if m == nil || len(m.l) == 0 {
		return nil
	}

	return m.l[len(m.l)-1]
}

// At returns the element at the specified index.
func (m *Map[K, V]) At(index int) *Element[K, V] {
	if m == nil || len(m.l) == 0 {
		return nil
	}
	if index < 0 || index >= len(m.l) {
		return nil
	}

	return m.l[index]
}

// All returns an iterator that iterates over all elements in the map, in the order they were added.
// The iterator is safe for concurrent mutations - it creates a snapshot at the start of iteration,
// so elements added or removed during iteration will not affect the current iteration.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil {
			return
		}

		// Create a snapshot of the current elements to avoid issues with concurrent modifications
		// This is memory efficient as we're only copying pointers, not the actual elements
		snapshot := make([]*Element[K, V], len(m.l))
		copy(snapshot, m.l)

		for _, element := range snapshot {
			// Check if element still exists in the map (it might have been deleted during iteration)
			if _, exists := m.m[element.Key]; exists {
				if !yield(element.Key, element.Value) {
					return
				}
			}
		}
	}
}

// AllOrdered returns an iterator that iterates over all elements in the map in the specified order.
// The iterator is safe for concurrent mutations - it creates a snapshot at the start of iteration,
// so elements added or removed during iteration will not affect the current iteration.
func (m *Map[K, V]) AllOrdered(order OrderType) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil {
			return
		}

		// Create a snapshot of the current elements to avoid issues with concurrent modifications
		snapshot := make([]*Element[K, V], len(m.l))
		copy(snapshot, m.l)

		switch order {
		case OrderAdded:
			// Same as All() - iterate in insertion order
			for _, element := range snapshot {
				// Check if element still exists in the map (it might have been deleted during iteration)
				if _, exists := m.m[element.Key]; exists {
					if !yield(element.Key, element.Value) {
						return
					}
				}
			}

		case OrderAddedReverse:
			// Iterate in reverse insertion order
			for i := len(snapshot) - 1; i >= 0; i-- {
				element := snapshot[i]
				// Check if element still exists in the map (it might have been deleted during iteration)
				if _, exists := m.m[element.Key]; exists {
					if !yield(element.Key, element.Value) {
						return
					}
				}
			}

		case OrderKeyAsc:
			// Sort by key in ascending order
			sortedElements := make([]*Element[K, V], len(snapshot))
			copy(sortedElements, snapshot)
			sort.Slice(sortedElements, func(i, j int) bool {
				return compareKeys(sortedElements[i].Key, sortedElements[j].Key) < 0
			})
			for _, element := range sortedElements {
				// Check if element still exists in the map (it might have been deleted during iteration)
				if _, exists := m.m[element.Key]; exists {
					if !yield(element.Key, element.Value) {
						return
					}
				}
			}

		case OrderKeyDesc:
			// Sort by key in descending order
			sortedElements := make([]*Element[K, V], len(snapshot))
			copy(sortedElements, snapshot)
			sort.Slice(sortedElements, func(i, j int) bool {
				return compareKeys(sortedElements[i].Key, sortedElements[j].Key) > 0
			})
			for _, element := range sortedElements {
				// Check if element still exists in the map (it might have been deleted during iteration)
				if _, exists := m.m[element.Key]; exists {
					if !yield(element.Key, element.Value) {
						return
					}
				}
			}
		}
	}
}

// AllUntyped returns an iterator that iterates over all elements in the map with untyped key and value.
// This allows for using the map in generic code.
// The iterator is safe for concurrent mutations - it creates a snapshot at the start of iteration.
func (m *Map[K, V]) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {
		if m == nil {
			return
		}

		// Create a snapshot of the current elements to avoid issues with concurrent modifications
		snapshot := make([]*Element[K, V], len(m.l))
		copy(snapshot, m.l)

		for _, element := range snapshot {
			// Check if element still exists in the map (it might have been deleted during iteration)
			if _, exists := m.m[element.Key]; exists {
				if !yield(element.Key, element.Value) {
					return
				}
			}
		}
	}
}

// Keys returns an iterator that iterates over all keys in the map, in the order they were added.
// The iterator is safe for concurrent mutations - it creates a snapshot at the start of iteration.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if m == nil {
			return
		}

		// Create a snapshot of the current elements to avoid issues with concurrent modifications
		snapshot := make([]*Element[K, V], len(m.l))
		copy(snapshot, m.l)

		for _, element := range snapshot {
			// Check if element still exists in the map (it might have been deleted during iteration)
			if _, exists := m.m[element.Key]; exists {
				if !yield(element.Key) {
					return
				}
			}
		}
	}
}

// Values returns an iterator that iterates over all values in the map, in the order they were added.
// The iterator is safe for concurrent mutations - it creates a snapshot at the start of iteration.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		if m == nil {
			return
		}

		// Create a snapshot of the current elements to avoid issues with concurrent modifications
		snapshot := make([]*Element[K, V], len(m.l))
		copy(snapshot, m.l)

		for _, element := range snapshot {
			// Check if element still exists in the map (it might have been deleted during iteration)
			if _, exists := m.m[element.Key]; exists {
				if !yield(element.Value) {
					return
				}
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
		return nil, errors.New("sequencedmap.Map is nil")
	}

	keyType := reflect.TypeOf((*K)(nil)).Elem()

	if reflect.TypeOf((*K)(nil)).Elem().Kind() != reflect.String {
		return nil, errors.New("sequencedmap.Map key type must be string")
	}

	var ka any = key
	k, ok := ka.(K)
	if !ok {
		// Try to convert if the underlying types are the same
		var zero K
		if reflect.TypeOf(ka).ConvertibleTo(reflect.TypeOf(zero)) {
			k = reflect.ValueOf(ka).Convert(reflect.TypeOf(zero)).Interface().(K)
		} else {
			return nil, fmt.Errorf("key not convertible to sequencedmap.Map key type %v", keyType)
		}
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

// UnmarshalYAML implements the yaml.Unmarshaler interface for the Map type.
// It unmarshals YAML data into the sequenced map, preserving the order of keys.
func (m *Map[K, V]) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("cannot unmarshal %s into sequencedmap.Map", value.ShortTag())
	}

	// Initialize the map if it's not already initialized
	if !m.IsInitialized() {
		m.Init()
	}

	// Clear existing data
	m.m = make(map[K]*Element[K, V])
	m.l = make([]*Element[K, V], 0)

	// Process key-value pairs from the YAML mapping node
	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		// Unmarshal the key
		var key K
		if err := keyNode.Decode(&key); err != nil {
			return fmt.Errorf("failed to decode key: %w", err)
		}

		// Unmarshal the value
		var val V
		if err := valueNode.Decode(&val); err != nil {
			return fmt.Errorf("failed to decode value for key %v: %w", key, err)
		}

		// Add the key-value pair to the map
		element := &Element[K, V]{
			Key:   key,
			Value: val,
		}
		m.m[key] = element
		m.l = append(m.l, element)
	}

	return nil
}

// MarshalYAML implements the yaml.Marshaler interface for the Map type.
// It marshals the sequenced map to YAML, preserving the order of keys.
func (m *Map[K, V]) MarshalYAML() (interface{}, error) {
	if m == nil {
		return nil, nil
	}

	content := make([]*yaml.Node, 0, len(m.l)*2)
	for _, element := range m.l {
		var keyNode yaml.Node
		if err := keyNode.Encode(element.Key); err != nil {
			return nil, err
		}
		var valueNode yaml.Node
		if err := valueNode.Encode(element.Value); err != nil {
			return nil, err
		}
		content = append(content, &keyNode, &valueNode)
	}

	return yml.CreateMapNode(context.Background(), content), nil
}

// compareKeys provides a generic comparison function for keys
func compareKeys[K comparable](a, b K) int {
	// Convert to strings for comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// IsEqual compares two Map instances for equality.
// It compares both the keys and values, and requires them to be in the same order.
// Treats both empty and nil maps as equal.
func (m *Map[K, V]) IsEqual(other *Map[K, V]) bool {
	if m == nil && other == nil {
		return true
	}

	// Treat nil and empty maps as equal
	mLen := 0
	if m != nil {
		mLen = m.Len()
	}
	otherLen := 0
	if other != nil {
		otherLen = other.Len()
	}

	if mLen == 0 && otherLen == 0 {
		return true
	}

	if mLen != otherLen {
		return false
	}

	// Compare all key-value pairs in order
	for key, valueA := range m.All() {
		valueB, exists := other.Get(key)
		if !exists {
			return false
		}

		// Use reflect.DeepEqual for value comparison
		if !reflect.DeepEqual(valueA, valueB) {
			return false
		}
	}
	return true
}

// IsEqualFunc compares two Map instances for equality using a custom comparison function.
// This is useful when you need custom comparison logic for the values.
// Treats both empty and nil maps as equal.
func (m *Map[K, V]) IsEqualFunc(other *Map[K, V], equalFunc func(V, V) bool) bool {
	if m == nil && other == nil {
		return true
	}

	// Treat nil and empty maps as equal
	mLen := 0
	if m != nil {
		mLen = m.Len()
	}
	otherLen := 0
	if other != nil {
		otherLen = other.Len()
	}

	if mLen == 0 && otherLen == 0 {
		return true
	}

	if mLen != otherLen {
		return false
	}

	// Compare all key-value pairs using the custom function
	for key, valueA := range m.All() {
		valueB, exists := other.Get(key)
		if !exists {
			return false
		}

		if !equalFunc(valueA, valueB) {
			return false
		}
	}
	return true
}
