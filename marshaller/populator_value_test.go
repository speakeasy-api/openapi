package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Populator interface path in populateValue
type MockPopulator struct {
	Value          string
	FromCoreCalled bool
}

func (m *MockPopulator) Populate(c any) error {
	m.FromCoreCalled = true
	if str, ok := c.(string); ok {
		m.Value = str
	}
	return nil
}

func Test_PopulateValue_ModelFromCore_Success(t *testing.T) {
	source := "test-value"
	target := &MockPopulator{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	assert.True(t, target.FromCoreCalled)
	assert.Equal(t, "test-value", target.Value)
}

// Test Populator error path
type ErrorPopulator struct{}

func (e *ErrorPopulator) Populate(c any) error {
	return assert.AnError
}

func Test_PopulateValue_ModelFromCore_Error(t *testing.T) {
	source := "test-value"
	target := &ErrorPopulator{}

	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

// Test slice population in populateValue (lines 153-164)
func Test_PopulateValue_Slice_Success(t *testing.T) {
	type SimpleStruct struct {
		SliceField []string
	}

	source := SimpleStruct{
		SliceField: []string{"item1", "item2", "item3"},
	}

	target := &SimpleStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	assert.Equal(t, []string{"item1", "item2", "item3"}, target.SliceField)
}

// Test array population in populateValue
func Test_PopulateValue_Array_Success(t *testing.T) {
	type ArrayStruct struct {
		ArrayField [3]string
	}

	source := ArrayStruct{
		ArrayField: [3]string{"item1", "item2", "item3"},
	}

	target := &ArrayStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	assert.Equal(t, [3]string{"item1", "item2", "item3"}, target.ArrayField)
}

// Test nil slice handling in populateValue (line 154-155)
func Test_PopulateValue_NilSlice_Success(t *testing.T) {
	type SliceStruct struct {
		SliceField []string
	}

	source := SliceStruct{
		SliceField: nil,
	}

	target := &SliceStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	assert.Nil(t, target.SliceField)
}

// Test recursive slice population with error
type FailingStruct struct {
	Value string
}

func Test_PopulateValue_Slice_Recursive_Error(t *testing.T) {
	type SliceStruct struct {
		SliceField []interface{}
	}

	// Create a slice with a value that can't be converted to target type
	source := SliceStruct{
		SliceField: []interface{}{make(chan int)}, // channel can't be converted to most types
	}

	type TargetStruct struct {
		SliceField []string
	}

	target := &TargetStruct{}

	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot convert")
}

// Test nil pointer source and target both pointers (lines 117-120)
func Test_PopulateValue_NilPointer_BothPointers(t *testing.T) {
	type PointerStruct struct {
		PtrField *string
	}

	source := PointerStruct{
		PtrField: nil,
	}

	target := &PointerStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	assert.Nil(t, target.PtrField)
}

// Skip type conversion test for now - this path requires specific interface scenarios

// Test assignable type path (line 166-167)
func Test_PopulateValue_AssignableType_Success(t *testing.T) {
	type SourceStruct struct {
		StringField string
	}

	type TargetStruct struct {
		StringField string // Same type, directly assignable
	}

	source := SourceStruct{
		StringField: "test-value",
	}

	target := &TargetStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	assert.Equal(t, "test-value", target.StringField)
}

// Test conversion error path (line 171)
func Test_PopulateValue_ConversionError(t *testing.T) {
	type SourceStruct struct {
		ChanField chan int // Cannot be converted to string
	}

	type TargetStruct struct {
		ChanField string
	}

	source := SourceStruct{
		ChanField: make(chan int),
	}

	target := &TargetStruct{}

	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot convert")
}

// Test nested slice with complex structs
func Test_PopulateValue_NestedSliceComplex_Success(t *testing.T) {
	type NestedStruct struct {
		Value string
		Count int
	}

	type ContainerStruct struct {
		Items []NestedStruct
	}

	source := ContainerStruct{
		Items: []NestedStruct{
			{Value: "first", Count: 1},
			{Value: "second", Count: 2},
		},
	}

	target := &ContainerStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	require.Len(t, target.Items, 2)
	assert.Equal(t, "first", target.Items[0].Value)
	assert.Equal(t, 1, target.Items[0].Count)
	assert.Equal(t, "second", target.Items[1].Value)
	assert.Equal(t, 2, target.Items[1].Count)
}
