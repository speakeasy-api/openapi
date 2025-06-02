package marshaller_test

import (
	"iter"
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test populateSequencedMap function coverage
func Test_PopulateModel_SequencedMap_Success(t *testing.T) {
	// Create source SequencedMap
	source := sequencedmap.New[string, string]()
	require.NoError(t, source.SetUntyped("key1", "value1"))
	require.NoError(t, source.SetUntyped("key2", "value2"))
	require.NoError(t, source.SetUntyped("key3", "value3"))

	// Create target SequencedMap
	target := sequencedmap.New[string, string]()

	// Test populateSequencedMap by calling PopulateModel
	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify the data was copied
	value1, ok := target.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", value1)

	value2, ok := target.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, "value2", value2)

	value3, ok := target.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, "value3", value3)

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
	source := sequencedmap.New[string, string]()

	// Create target SequencedMap
	target := sequencedmap.New[string, string]()

	// Test populateSequencedMap by calling PopulateModel
	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify target has no data
	assert.Equal(t, 0, target.Len())
}

// Test populateSequencedMap error case: source not SequencedMap
func Test_PopulateModel_SequencedMap_InvalidSource_Error(t *testing.T) {
	// Create a struct that's not a SequencedMap but will reach populateSequencedMap path
	type NotSequencedMapSource struct {
		Field string
	}

	source := &NotSequencedMapSource{Field: "not-a-sequenced-map"}

	// Create target SequencedMap
	target := sequencedmap.New[string, string]()

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
	source := sequencedmap.New[string, string]()
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

// Complex value type for testing populateValue recursion
type ComplexValue struct {
	Field string
}

// Test populateSequencedMap with complex values that require populateValue recursion
func Test_PopulateModel_SequencedMap_ComplexValues_Success(t *testing.T) {
	// Create source SequencedMap with complex values
	source := sequencedmap.New[string, ComplexValue]()
	require.NoError(t, source.SetUntyped("item1", ComplexValue{Field: "value1"}))
	require.NoError(t, source.SetUntyped("item2", ComplexValue{Field: "value2"}))

	// Create target SequencedMap
	target := sequencedmap.New[string, ComplexValue]()

	// Test populateSequencedMap
	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify complex values were copied correctly
	item1, ok := target.Get("item1")
	require.True(t, ok)
	assert.Equal(t, "value1", item1.Field)

	item2, ok := target.Get("item2")
	require.True(t, ok)
	assert.Equal(t, "value2", item2.Field)
}
