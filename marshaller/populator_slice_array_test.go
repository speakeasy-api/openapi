package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test slice handling in populateValue by creating a scenario where
// the target doesn't implement special interfaces but the source has slice fields
func Test_PopulateValue_Slice_DirectPopulation_Success(t *testing.T) {
	// Create a scenario where populateValue handles slices directly
	// This happens when the target is a simple type (not implementing special interfaces)
	// and the source contains slices

	type SimpleTarget struct {
		StringSlice []string
		IntSlice    []int
	}

	// Source with slice data
	source := SimpleTarget{
		StringSlice: []string{"item1", "item2", "item3"},
		IntSlice:    []int{10, 20, 30},
	}

	// Target that will be populated
	target := &SimpleTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify slice population
	assert.Equal(t, []string{"item1", "item2", "item3"}, target.StringSlice)
	assert.Equal(t, []int{10, 20, 30}, target.IntSlice)
}

// Test array handling in populateValue
func Test_PopulateValue_Array_DirectPopulation_Success(t *testing.T) {
	type ArrayTarget struct {
		StringArray [3]string
		IntArray    [2]int
	}

	// Source with array data
	source := ArrayTarget{
		StringArray: [3]string{"a", "b", "c"},
		IntArray:    [2]int{100, 200},
	}

	// Target that will be populated
	target := &ArrayTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify array population
	assert.Equal(t, [3]string{"a", "b", "c"}, target.StringArray)
	assert.Equal(t, [2]int{100, 200}, target.IntArray)
}

// Test nil slice handling (line 154-155)
func Test_PopulateValue_NilSlice_DirectPopulation_Success(t *testing.T) {
	type NilSliceTarget struct {
		StringSlice []string
		IntSlice    []int
	}

	// Source with nil slices
	source := NilSliceTarget{
		StringSlice: nil,
		IntSlice:    nil,
	}

	// Target that will be populated
	target := &NilSliceTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify nil slices remain nil
	assert.Nil(t, target.StringSlice)
	assert.Nil(t, target.IntSlice)
}

// Test empty slice handling
func Test_PopulateValue_EmptySlice_DirectPopulation_Success(t *testing.T) {
	type EmptySliceTarget struct {
		StringSlice []string
		IntSlice    []int
	}

	// Source with empty slices (not nil, but length 0)
	source := EmptySliceTarget{
		StringSlice: []string{},
		IntSlice:    []int{},
	}

	// Target that will be populated
	target := &EmptySliceTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify empty slices are created
	assert.NotNil(t, target.StringSlice)
	assert.NotNil(t, target.IntSlice)
	assert.Len(t, target.StringSlice, 0)
	assert.Len(t, target.IntSlice, 0)
}

// Test slice with complex nested elements
func Test_PopulateValue_NestedSlice_DirectPopulation_Success(t *testing.T) {
	type NestedItem struct {
		Name  string
		Value int
	}

	type NestedSliceTarget struct {
		Items []NestedItem
	}

	// Source with nested struct slice
	source := NestedSliceTarget{
		Items: []NestedItem{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
			{Name: "third", Value: 3},
		},
	}

	// Target that will be populated
	target := &NestedSliceTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify nested slice population
	require.Len(t, target.Items, 3)
	assert.Equal(t, "first", target.Items[0].Name)
	assert.Equal(t, 1, target.Items[0].Value)
	assert.Equal(t, "second", target.Items[1].Name)
	assert.Equal(t, 2, target.Items[1].Value)
	assert.Equal(t, "third", target.Items[2].Name)
	assert.Equal(t, 3, target.Items[2].Value)
}

// Test error in recursive populateValue call (line 161-163)
func Test_PopulateValue_SliceRecursion_Error(t *testing.T) {
	// Create a slice with elements that can't be converted
	type IncompatibleSource struct {
		Items []chan int // channels can't be converted to most types
	}

	type IncompatibleTarget struct {
		Items []string // can't convert from chan int to string
	}

	// Source with unconvertible slice elements
	source := IncompatibleSource{
		Items: []chan int{make(chan int), make(chan int)},
	}

	// Target that will cause conversion error
	target := &IncompatibleTarget{}

	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot convert")
}

// Test slice of pointers
func Test_PopulateValue_SliceOfPointers_DirectPopulation_Success(t *testing.T) {
	type PointerSliceTarget struct {
		StringPtrs []*string
	}

	// Create string values
	val1 := "pointer1"
	val2 := "pointer2"

	// Source with slice of pointers
	source := PointerSliceTarget{
		StringPtrs: []*string{&val1, &val2},
	}

	// Target that will be populated
	target := &PointerSliceTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify pointer slice population
	require.Len(t, target.StringPtrs, 2)
	require.NotNil(t, target.StringPtrs[0])
	require.NotNil(t, target.StringPtrs[1])
	assert.Equal(t, "pointer1", *target.StringPtrs[0])
	assert.Equal(t, "pointer2", *target.StringPtrs[1])
}

// Test multi-dimensional slice
func Test_PopulateValue_MultiDimensionalSlice_DirectPopulation_Success(t *testing.T) {
	type MultiDimTarget struct {
		Matrix [][]int
	}

	// Source with 2D slice
	source := MultiDimTarget{
		Matrix: [][]int{
			{1, 2, 3},
			{4, 5, 6},
		},
	}

	// Target that will be populated
	target := &MultiDimTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify 2D slice population
	require.Len(t, target.Matrix, 2)
	assert.Equal(t, []int{1, 2, 3}, target.Matrix[0])
	assert.Equal(t, []int{4, 5, 6}, target.Matrix[1])
}

// Test large slice to ensure the loop works correctly
func Test_PopulateValue_LargeSlice_DirectPopulation_Success(t *testing.T) {
	type LargeSliceTarget struct {
		Numbers []int
	}

	// Create a large slice
	sourceNumbers := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		sourceNumbers[i] = i
	}

	source := LargeSliceTarget{
		Numbers: sourceNumbers,
	}

	// Target that will be populated
	target := &LargeSliceTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify large slice population
	require.Len(t, target.Numbers, 1000)
	assert.Equal(t, 0, target.Numbers[0])
	assert.Equal(t, 500, target.Numbers[500])
	assert.Equal(t, 999, target.Numbers[999])
}
