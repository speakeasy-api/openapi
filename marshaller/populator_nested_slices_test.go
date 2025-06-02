package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test slice of slices to potentially trigger the slice case in populateValue
func Test_PopulateValue_SliceOfSlices_Success(t *testing.T) {
	type SliceOfSlicesStruct struct {
		NestedSlices [][]string
		NestedArrays [][3]int
	}

	source := SliceOfSlicesStruct{
		NestedSlices: [][]string{
			{"a", "b", "c"},
			{"d", "e", "f"},
			{"g", "h", "i"},
		},
		NestedArrays: [][3]int{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
		},
	}

	target := &SliceOfSlicesStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify nested slices
	require.Len(t, target.NestedSlices, 3)
	assert.Equal(t, []string{"a", "b", "c"}, target.NestedSlices[0])
	assert.Equal(t, []string{"d", "e", "f"}, target.NestedSlices[1])
	assert.Equal(t, []string{"g", "h", "i"}, target.NestedSlices[2])

	// Verify nested arrays
	require.Len(t, target.NestedArrays, 3)
	assert.Equal(t, [3]int{1, 2, 3}, target.NestedArrays[0])
	assert.Equal(t, [3]int{4, 5, 6}, target.NestedArrays[1])
	assert.Equal(t, [3]int{7, 8, 9}, target.NestedArrays[2])
}

// Test slice of slices of slices (3D)
func Test_PopulateValue_SliceOfSlicesOfSlices_Success(t *testing.T) {
	type TripleNestedStruct struct {
		ThreeDSlice [][][]string
	}

	source := TripleNestedStruct{
		ThreeDSlice: [][][]string{
			{
				{"1a", "1b"},
				{"1c", "1d"},
			},
			{
				{"2a", "2b"},
				{"2c", "2d"},
			},
		},
	}

	target := &TripleNestedStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify 3D slice structure
	require.Len(t, target.ThreeDSlice, 2)
	require.Len(t, target.ThreeDSlice[0], 2)
	require.Len(t, target.ThreeDSlice[1], 2)

	assert.Equal(t, []string{"1a", "1b"}, target.ThreeDSlice[0][0])
	assert.Equal(t, []string{"1c", "1d"}, target.ThreeDSlice[0][1])
	assert.Equal(t, []string{"2a", "2b"}, target.ThreeDSlice[1][0])
	assert.Equal(t, []string{"2c", "2d"}, target.ThreeDSlice[1][1])
}

// Test slice of slices with nil inner slices
func Test_PopulateValue_SliceOfSlices_WithNilInner_Success(t *testing.T) {
	type SliceWithNilsStruct struct {
		MixedSlices [][]string
	}

	source := SliceWithNilsStruct{
		MixedSlices: [][]string{
			{"a", "b"},
			nil, // This should trigger the nil slice path
			{"c", "d"},
		},
	}

	target := &SliceWithNilsStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify mixed slices with nil
	require.Len(t, target.MixedSlices, 3)
	assert.Equal(t, []string{"a", "b"}, target.MixedSlices[0])
	assert.Nil(t, target.MixedSlices[1])
	assert.Equal(t, []string{"c", "d"}, target.MixedSlices[2])
}

// Test array of slices
func Test_PopulateValue_ArrayOfSlices_Success(t *testing.T) {
	type ArrayOfSlicesStruct struct {
		ArrayOfSlices [3][]string
	}

	source := ArrayOfSlicesStruct{
		ArrayOfSlices: [3][]string{
			{"first", "slice"},
			{"second", "slice"},
			{"third", "slice"},
		},
	}

	target := &ArrayOfSlicesStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify array of slices
	assert.Equal(t, []string{"first", "slice"}, target.ArrayOfSlices[0])
	assert.Equal(t, []string{"second", "slice"}, target.ArrayOfSlices[1])
	assert.Equal(t, []string{"third", "slice"}, target.ArrayOfSlices[2])
}

// Test slice of arrays
func Test_PopulateValue_SliceOfArrays_Success(t *testing.T) {
	type SliceOfArraysStruct struct {
		SliceOfArrays [][2]string
	}

	source := SliceOfArraysStruct{
		SliceOfArrays: [][2]string{
			{"a1", "a2"},
			{"b1", "b2"},
			{"c1", "c2"},
		},
	}

	target := &SliceOfArraysStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify slice of arrays
	require.Len(t, target.SliceOfArrays, 3)
	assert.Equal(t, [2]string{"a1", "a2"}, target.SliceOfArrays[0])
	assert.Equal(t, [2]string{"b1", "b2"}, target.SliceOfArrays[1])
	assert.Equal(t, [2]string{"c1", "c2"}, target.SliceOfArrays[2])
}

// Test complex nested structure with mixed types
func Test_PopulateValue_ComplexNestedSlices_Success(t *testing.T) {
	type ComplexItem struct {
		Values []string
		Counts []int
	}

	type ComplexNestedStruct struct {
		ItemMatrix [][]ComplexItem
	}

	source := ComplexNestedStruct{
		ItemMatrix: [][]ComplexItem{
			{
				{Values: []string{"row1col1val1", "row1col1val2"}, Counts: []int{1, 2}},
				{Values: []string{"row1col2val1"}, Counts: []int{3}},
			},
			{
				{Values: []string{"row2col1val1"}, Counts: []int{4, 5, 6}},
				{Values: []string{"row2col2val1", "row2col2val2"}, Counts: []int{7}},
			},
		},
	}

	target := &ComplexNestedStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify complex nested structure
	require.Len(t, target.ItemMatrix, 2)
	require.Len(t, target.ItemMatrix[0], 2)
	require.Len(t, target.ItemMatrix[1], 2)

	// Check first item
	assert.Equal(t, []string{"row1col1val1", "row1col1val2"}, target.ItemMatrix[0][0].Values)
	assert.Equal(t, []int{1, 2}, target.ItemMatrix[0][0].Counts)

	// Check second item
	assert.Equal(t, []string{"row1col2val1"}, target.ItemMatrix[0][1].Values)
	assert.Equal(t, []int{3}, target.ItemMatrix[0][1].Counts)

	// Check third item
	assert.Equal(t, []string{"row2col1val1"}, target.ItemMatrix[1][0].Values)
	assert.Equal(t, []int{4, 5, 6}, target.ItemMatrix[1][0].Counts)

	// Check fourth item
	assert.Equal(t, []string{"row2col2val1", "row2col2val2"}, target.ItemMatrix[1][1].Values)
	assert.Equal(t, []int{7}, target.ItemMatrix[1][1].Counts)
}

// Test empty outer slice
func Test_PopulateValue_EmptyOuterSlice_Success(t *testing.T) {
	type EmptyOuterStruct struct {
		EmptyOuter [][]string
	}

	source := EmptyOuterStruct{
		EmptyOuter: [][]string{}, // Empty outer slice
	}

	target := &EmptyOuterStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify empty outer slice
	assert.NotNil(t, target.EmptyOuter)
	assert.Len(t, target.EmptyOuter, 0)
}

// Test nil outer slice
func Test_PopulateValue_NilOuterSlice_Success(t *testing.T) {
	type NilOuterStruct struct {
		NilOuter [][]string
	}

	source := NilOuterStruct{
		NilOuter: nil, // Nil outer slice
	}

	target := &NilOuterStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)

	// Verify nil outer slice
	assert.Nil(t, target.NilOuter)
}
