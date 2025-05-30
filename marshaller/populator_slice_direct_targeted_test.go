package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create a scenario where ModelFromCore.FromCore calls something that triggers slice path
type SliceFromCoreProcessor struct {
	ProcessedSlices [][]string
	ProcessedArrays [][3]int
}

func (s *SliceFromCoreProcessor) FromCore(c any) error {
	// This implements ModelFromCore interface
	// Inside here, we could theoretically call populateValue with slices directly
	// But since we don't have access to populateValue directly, we'll simulate the scenario
	
	if data, ok := c.(map[string]any); ok {
		if sliceData, exists := data["slices"]; exists {
			if slices, ok := sliceData.([][]string); ok {
				s.ProcessedSlices = slices
			}
		}
		if arrayData, exists := data["arrays"]; exists {
			if arrays, ok := arrayData.([][3]int); ok {
				s.ProcessedArrays = arrays
			}
		}
	}
	return nil
}

func Test_PopulateValue_SliceFromCoreProcessor_Success(t *testing.T) {
	source := map[string]any{
		"slices": [][]string{{"a", "b"}, {"c", "d"}},
		"arrays": [][3]int{{1, 2, 3}, {4, 5, 6}},
	}

	target := &SliceFromCoreProcessor{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	assert.Len(t, target.ProcessedSlices, 2)
	assert.Equal(t, []string{"a", "b"}, target.ProcessedSlices[0])
	assert.Equal(t, []string{"c", "d"}, target.ProcessedSlices[1])
	
	assert.Len(t, target.ProcessedArrays, 2)
	assert.Equal(t, [3]int{1, 2, 3}, target.ProcessedArrays[0])
	assert.Equal(t, [3]int{4, 5, 6}, target.ProcessedArrays[1])
}

// Try a different approach: Create a situation where the slice path is more likely to be hit
// by using simple non-struct types that could trigger the slice case

// Let's try to create an unusual scenario where populateValue might be called directly
type UnusualSliceTarget struct {
	// No special interfaces, just basic slice fields
	Data []string
}

// This test is designed to see if we can force the slice path
func Test_PopulateValue_UnusualSliceScenario_Success(t *testing.T) {
	source := UnusualSliceTarget{
		Data: []string{"unusual1", "unusual2", "unusual3"},
	}

	target := &UnusualSliceTarget{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	assert.Equal(t, []string{"unusual1", "unusual2", "unusual3"}, target.Data)
}

// Create a test that might trigger slice paths through reflection
type ReflectiveSliceTarget struct {
	SliceField any // Using interface{} to potentially trigger different reflection paths
}

func Test_PopulateValue_ReflectiveSlice_Success(t *testing.T) {
	// Create source where the any field contains a slice
	source := ReflectiveSliceTarget{
		SliceField: []string{"reflective1", "reflective2"},
	}

	target := &ReflectiveSliceTarget{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	// Verify the slice was copied
	if sliceValue, ok := target.SliceField.([]string); ok {
		assert.Equal(t, []string{"reflective1", "reflective2"}, sliceValue)
	} else {
		t.Errorf("Expected []string, got %T", target.SliceField)
	}
}

// Test with slice of interface{} to see if we can trigger the slice path
type InterfaceSliceTarget struct {
	Items []any
}

func Test_PopulateValue_InterfaceSlice_Success(t *testing.T) {
	source := InterfaceSliceTarget{
		Items: []any{"item1", 42, true},
	}

	target := &InterfaceSliceTarget{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	require.Len(t, target.Items, 3)
	assert.Equal(t, "item1", target.Items[0])
	assert.Equal(t, 42, target.Items[1])
	assert.Equal(t, true, target.Items[2])
}