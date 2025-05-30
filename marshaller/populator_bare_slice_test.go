package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// To trigger the slice/array case in populateValue, we need:
// 1. A target that does NOT implement ModelFromCore, CoreSetter, or SequencedMap
// 2. A source that is a slice or array
// 3. The target to be a pointer that after target.Elem() can accept the slice/array

// Simple struct with no special interfaces - this should bypass all interface checks
type BareSliceTarget struct {
	SimpleSlice []string
	SimpleArray [3]int
}

// This test tries to create the exact scenario for the slice case
func Test_PopulateValue_BareSliceTarget_Success(t *testing.T) {
	source := BareSliceTarget{
		SimpleSlice: []string{"bare1", "bare2", "bare3"},
		SimpleArray: [3]int{10, 20, 30},
	}

	target := &BareSliceTarget{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	assert.Equal(t, []string{"bare1", "bare2", "bare3"}, target.SimpleSlice)
	assert.Equal(t, [3]int{10, 20, 30}, target.SimpleArray)
}

// Try with an even simpler case: directly pass slice values
// This creates a scenario where populateValue would be called recursively 
// with slice source and pointer target

type ContainerWithSliceField struct {
	SliceField []string `json:"slice_field"`
}

func Test_PopulateValue_ContainerWithSlice_Success(t *testing.T) {
	// This should trigger populateValue for the slice field
	source := ContainerWithSliceField{
		SliceField: []string{"container1", "container2"},
	}

	target := &ContainerWithSliceField{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	assert.Equal(t, []string{"container1", "container2"}, target.SliceField)
}

// Try with a pointer to slice - this might create the exact target scenario
type PointerSliceTarget struct {
	SlicePtr *[]string
	ArrayPtr *[2]int
}

func Test_PopulateValue_PointerSliceTarget_Success(t *testing.T) {
	sourceSlice := []string{"ptr1", "ptr2"}
	sourceArray := [2]int{100, 200}
	
	source := PointerSliceTarget{
		SlicePtr: &sourceSlice,
		ArrayPtr: &sourceArray,
	}

	target := &PointerSliceTarget{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	require.NotNil(t, target.SlicePtr)
	require.NotNil(t, target.ArrayPtr)
	assert.Equal(t, []string{"ptr1", "ptr2"}, *target.SlicePtr)
	assert.Equal(t, [2]int{100, 200}, *target.ArrayPtr)
}

// Create an embedding scenario that might trigger different paths
type EmbeddedSliceStruct struct {
	Inner InnerSliceStruct
}

type InnerSliceStruct struct {
	Data []string
}

func Test_PopulateValue_EmbeddedSliceStruct_Success(t *testing.T) {
	source := EmbeddedSliceStruct{
		Inner: InnerSliceStruct{
			Data: []string{"embedded1", "embedded2"},
		},
	}

	target := &EmbeddedSliceStruct{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	assert.Equal(t, []string{"embedded1", "embedded2"}, target.Inner.Data)
}

// Try a minimal test case with just a slice
type JustSlice struct {
	Data []string
}

func Test_PopulateValue_JustSlice_Success(t *testing.T) {
	source := JustSlice{
		Data: []string{"just1", "just2"},
	}

	target := &JustSlice{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	assert.Equal(t, []string{"just1", "just2"}, target.Data)
}