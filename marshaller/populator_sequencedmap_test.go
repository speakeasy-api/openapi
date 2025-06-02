package marshaller_test

import (
	"iter"
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock SequencedMap implementation for testing populateSequencedMap
type MockSequencedMap struct {
	data     map[any]any
	keyOrder []any
}

func (m *MockSequencedMap) Init() {
	if m.data == nil {
		m.data = make(map[any]any)
	}
	if m.keyOrder == nil {
		m.keyOrder = make([]any, 0)
	}
}

func (m *MockSequencedMap) SetUntyped(key, value any) error {
	if m.data == nil {
		m.Init()
	}
	if _, exists := m.data[key]; !exists {
		m.keyOrder = append(m.keyOrder, key)
	}
	m.data[key] = value
	return nil
}

func (m *MockSequencedMap) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {
		if m.data == nil {
			return
		}
		for _, key := range m.keyOrder {
			if value, exists := m.data[key]; exists {
				if !yield(key, value) {
					return
				}
			}
		}
	}
}

func (m *MockSequencedMap) GetValueType() reflect.Type {
	return reflect.TypeOf("")
}

// Test populateSequencedMap function coverage
func Test_PopulateModel_SequencedMap_Success(t *testing.T) {
	// Create source SequencedMap
	source := &MockSequencedMap{}
	source.Init()
	require.NoError(t, source.SetUntyped("key1", "value1"))
	require.NoError(t, source.SetUntyped("key2", "value2"))
	require.NoError(t, source.SetUntyped("key3", "value3"))

	// Create target SequencedMap
	target := &MockSequencedMap{}

	// Test populateSequencedMap by calling PopulateModel
	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify the data was copied
	assert.NotNil(t, target.data)
	assert.Equal(t, "value1", target.data["key1"])
	assert.Equal(t, "value2", target.data["key2"])
	assert.Equal(t, "value3", target.data["key3"])

	// Verify order is maintained
	keys := make([]any, 0)
	for key := range target.AllUntyped() {
		keys = append(keys, key)
	}
	assert.Equal(t, []any{"key1", "key2", "key3"}, keys)
}

// Test populateSequencedMap with nil source
func Test_PopulateModel_SequencedMap_NilSource_Success(t *testing.T) {
	// Create source SequencedMap with no data (uninitialized)
	source := &MockSequencedMap{}

	// Create target SequencedMap
	target := &MockSequencedMap{}

	// Test populateSequencedMap by calling PopulateModel
	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify target was initialized but has no data
	assert.NotNil(t, target.data)
	assert.Empty(t, target.data)
}

// Test populateSequencedMap error case: source not SequencedMap
func Test_PopulateModel_SequencedMap_InvalidSource_Error(t *testing.T) {
	// Create a struct that's not a SequencedMap but will reach populateSequencedMap path
	type NotSequencedMapSource struct {
		Field string
	}

	source := &NotSequencedMapSource{Field: "not-a-sequenced-map"}

	// Create target SequencedMap
	target := &MockSequencedMap{}

	// Test should fail with type error when it tries to cast source to SequencedMap
	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected source to be SequencedMap")
}

// Mock struct that implements SequencedMap interface but will cause target error
type InvalidTargetSequencedMap struct {
	Field string
}

func (i *InvalidTargetSequencedMap) Init() {}

func (i *InvalidTargetSequencedMap) SetUntyped(key, value any) error {
	return nil
}

func (i *InvalidTargetSequencedMap) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {}
}

func (i *InvalidTargetSequencedMap) GetValueType() reflect.Type {
	return reflect.TypeOf("")
}

// Test populateSequencedMap error case: target not SequencedMap interface
func Test_PopulateModel_SequencedMap_InvalidTarget_Error(t *testing.T) {
	// Create source SequencedMap
	source := &MockSequencedMap{}
	source.Init()
	require.NoError(t, source.SetUntyped("key", "value"))

	// Create a mock target that looks like it implements SequencedMap but doesn't cast correctly
	// This creates a scenario where the type checking would pass but interface assertion fails
	type FakeSequencedMap struct {
		field string // nolint:unused
	}

	// Don't implement the interface - this will cause the populateSequencedMap to fail
	// when it tries to cast target.Interface().(SequencedMap)
	target := &FakeSequencedMap{}

	// This should fail because target doesn't actually implement SequencedMap interface
	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot convert")
}

// Mock SequencedMap that returns error on SetUntyped
type ErrorSequencedMap struct {
	MockSequencedMap
}

func (e *ErrorSequencedMap) SetUntyped(key, value any) error {
	return assert.AnError
}

// Test populateSequencedMap with SetUntyped error
func Test_PopulateModel_SequencedMap_SetError(t *testing.T) {
	// Create source SequencedMap
	source := &MockSequencedMap{}
	source.Init()
	require.NoError(t, source.SetUntyped("key", "value"))

	// Create target SequencedMap that errors on Set
	target := &ErrorSequencedMap{}

	// Test should fail with the set error
	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

// Mock SequencedMap with complex value type for testing populateValue recursion
type ComplexValue struct {
	Field string
}

type ComplexSequencedMap struct {
	data     map[any]any
	keyOrder []any
}

func (c *ComplexSequencedMap) Init() {
	if c.data == nil {
		c.data = make(map[any]any)
	}
	if c.keyOrder == nil {
		c.keyOrder = make([]any, 0)
	}
}

func (c *ComplexSequencedMap) SetUntyped(key, value any) error {
	if c.data == nil {
		c.Init()
	}
	if _, exists := c.data[key]; !exists {
		c.keyOrder = append(c.keyOrder, key)
	}
	c.data[key] = value
	return nil
}

func (c *ComplexSequencedMap) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {
		if c.data == nil {
			return
		}
		for _, key := range c.keyOrder {
			if value, exists := c.data[key]; exists {
				if !yield(key, value) {
					return
				}
			}
		}
	}
}

func (c *ComplexSequencedMap) GetValueType() reflect.Type {
	return reflect.TypeOf(ComplexValue{})
}

// Test populateSequencedMap with complex values that require populateValue recursion
func Test_PopulateModel_SequencedMap_ComplexValues_Success(t *testing.T) {
	// Create source SequencedMap with complex values
	source := &ComplexSequencedMap{}
	source.Init()
	require.NoError(t, source.SetUntyped("item1", ComplexValue{Field: "value1"}))
	require.NoError(t, source.SetUntyped("item2", ComplexValue{Field: "value2"}))

	// Create target SequencedMap
	target := &ComplexSequencedMap{}

	// Test populateSequencedMap
	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify complex values were copied correctly
	assert.NotNil(t, target.data)

	item1, ok := target.data["item1"].(ComplexValue)
	require.True(t, ok)
	assert.Equal(t, "value1", item1.Field)

	item2, ok := target.data["item2"].(ComplexValue)
	require.True(t, ok)
	assert.Equal(t, "value2", item2.Field)
}
