package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Based on analysis of populateValue, the slice case is triggered when:
// 1. value.Kind() is reflect.Slice or reflect.Array
// 2. target is a pointer (target.Kind() == reflect.Ptr)
// 3. target.Elem() gives us the actual slice/array to populate
//
// This happens during recursive calls within populateValue itself

// Create a scenario with slice of interface{} where each interface{} contains a slice
// This should force recursive populateValue calls with slice values

type RecursiveSliceStruct struct {
	SliceOfAny []interface{}
}

func Test_PopulateValue_RecursiveSliceCall_Success(t *testing.T) {
	// Each interface{} element contains a slice, which should trigger
	// recursive populateValue calls with slice values and pointer targets
	source := RecursiveSliceStruct{
		SliceOfAny: []interface{}{
			[]string{"sub1", "sub2"}, // This slice should trigger the slice case in populateValue
			[]int{10, 20, 30},       // This slice should also trigger it
			[2]string{"arr1", "arr2"}, // This array should trigger the array case
		},
	}

	target := &RecursiveSliceStruct{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	require.Len(t, target.SliceOfAny, 3)
	
	// Check first slice
	if firstSlice, ok := target.SliceOfAny[0].([]string); ok {
		assert.Equal(t, []string{"sub1", "sub2"}, firstSlice)
	} else {
		t.Errorf("Expected []string, got %T", target.SliceOfAny[0])
	}
	
	// Check second slice  
	if secondSlice, ok := target.SliceOfAny[1].([]int); ok {
		assert.Equal(t, []int{10, 20, 30}, secondSlice)
	} else {
		t.Errorf("Expected []int, got %T", target.SliceOfAny[1])
	}
	
	// Check array
	if thirdArray, ok := target.SliceOfAny[2].([2]string); ok {
		assert.Equal(t, [2]string{"arr1", "arr2"}, thirdArray)
	} else {
		t.Errorf("Expected [2]string, got %T", target.SliceOfAny[2])
	}
}

// Try with deeply nested structure that might force the slice path
type DeeplyNestedWithSlices struct {
	Level1 *Level1Struct
}

type Level1Struct struct {
	Level2 *Level2Struct
}

type Level2Struct struct {
	SliceData []string
	ArrayData [2]int
}

func Test_PopulateValue_DeeplyNestedSlices_Success(t *testing.T) {
	source := DeeplyNestedWithSlices{
		Level1: &Level1Struct{
			Level2: &Level2Struct{
				SliceData: []string{"deep1", "deep2"},
				ArrayData: [2]int{100, 200},
			},
		},
	}

	target := &DeeplyNestedWithSlices{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	require.NotNil(t, target.Level1)
	require.NotNil(t, target.Level1.Level2)
	assert.Equal(t, []string{"deep1", "deep2"}, target.Level1.Level2.SliceData)
	assert.Equal(t, [2]int{100, 200}, target.Level1.Level2.ArrayData)
}

// Test with mixed interface{} and slice nesting
type MixedNestingStruct struct {
	Data interface{}
}

func Test_PopulateValue_MixedNesting_SliceInInterface_Success(t *testing.T) {
	// Create a structure where a slice is nested inside interface{} values
	// which might trigger different paths in populateValue
	source := MixedNestingStruct{
		Data: map[string]interface{}{
			"nested_slice": []string{"mixed1", "mixed2"},
			"nested_array": [3]int{1, 2, 3},
		},
	}

	target := &MixedNestingStruct{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	if dataMap, ok := target.Data.(map[string]interface{}); ok {
		if nestedSlice, exists := dataMap["nested_slice"]; exists {
			if slice, ok := nestedSlice.([]string); ok {
				assert.Equal(t, []string{"mixed1", "mixed2"}, slice)
			} else {
				t.Errorf("Expected []string for nested_slice, got %T", nestedSlice)
			}
		}
		
		if nestedArray, exists := dataMap["nested_array"]; exists {
			if array, ok := nestedArray.([3]int); ok {
				assert.Equal(t, [3]int{1, 2, 3}, array)
			} else {
				t.Errorf("Expected [3]int for nested_array, got %T", nestedArray)
			}
		}
	} else {
		t.Errorf("Expected map[string]interface{}, got %T", target.Data)
	}
}

// Try to force the exact scenario: slice value with pointer target
// by using a map where values are slices
type SliceValueMapStruct struct {
	SliceMap map[string][]string
	ArrayMap map[string][2]int
}

func Test_PopulateValue_SliceValueInMap_Success(t *testing.T) {
	source := SliceValueMapStruct{
		SliceMap: map[string][]string{
			"key1": {"map1", "map2"},
			"key2": {"map3", "map4"},
		},
		ArrayMap: map[string][2]int{
			"arr1": {10, 20},
			"arr2": {30, 40},
		},
	}

	target := &SliceValueMapStruct{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	require.Len(t, target.SliceMap, 2)
	assert.Equal(t, []string{"map1", "map2"}, target.SliceMap["key1"])
	assert.Equal(t, []string{"map3", "map4"}, target.SliceMap["key2"])
	
	require.Len(t, target.ArrayMap, 2)
	assert.Equal(t, [2]int{10, 20}, target.ArrayMap["arr1"])
	assert.Equal(t, [2]int{30, 40}, target.ArrayMap["arr2"])
}