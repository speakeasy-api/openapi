package sequencedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap_IsEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		map1     *Map[string, int]
		map2     *Map[string, int]
		expected bool
	}{
		{
			name:     "both nil maps should be equal",
			map1:     nil,
			map2:     nil,
			expected: true,
		},
		{
			name:     "nil map and empty map should be equal",
			map1:     nil,
			map2:     New[string, int](),
			expected: true,
		},
		{
			name:     "empty map and nil map should be equal",
			map1:     New[string, int](),
			map2:     nil,
			expected: true,
		},
		{
			name:     "both empty maps should be equal",
			map1:     New[string, int](),
			map2:     New[string, int](),
			expected: true,
		},
		{
			name: "maps with same key-value pairs should be equal",
			map1: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			map2: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			expected: true,
		},
		{
			name: "maps with same key-value pairs in different order should be equal",
			map1: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			map2: New(
				NewElem("key2", 2),
				NewElem("key1", 1),
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.map1.IsEqual(tt.map2)
			assert.Equal(t, tt.expected, actual, "maps should match expected equality")
		})
	}
}

func TestMap_IsEqual_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		map1     *Map[string, int]
		map2     *Map[string, int]
		expected bool
	}{
		{
			name:     "nil map vs non-empty map should not be equal",
			map1:     nil,
			map2:     New(NewElem("key1", 1)),
			expected: false,
		},
		{
			name:     "non-empty map vs nil map should not be equal",
			map1:     New(NewElem("key1", 1)),
			map2:     nil,
			expected: false,
		},
		{
			name:     "empty map vs non-empty map should not be equal",
			map1:     New[string, int](),
			map2:     New(NewElem("key1", 1)),
			expected: false,
		},
		{
			name: "maps with different values should not be equal",
			map1: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			map2: New(
				NewElem("key1", 1),
				NewElem("key2", 3),
			),
			expected: false,
		},
		{
			name: "maps with different keys should not be equal",
			map1: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			map2: New(
				NewElem("key1", 1),
				NewElem("key3", 2),
			),
			expected: false,
		},
		{
			name: "maps with different lengths should not be equal",
			map1: New(
				NewElem("key1", 1),
			),
			map2: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.map1.IsEqual(tt.map2)
			assert.Equal(t, tt.expected, actual, "maps should match expected equality")
		})
	}
}

func TestMap_IsEqualFunc_Success(t *testing.T) {
	t.Parallel()

	customEqualFunc := func(a, b int) bool {
		return a == b
	}

	tests := []struct {
		name     string
		map1     *Map[string, int]
		map2     *Map[string, int]
		expected bool
	}{
		{
			name:     "both nil maps should be equal with custom func",
			map1:     nil,
			map2:     nil,
			expected: true,
		},
		{
			name:     "nil map and empty map should be equal with custom func",
			map1:     nil,
			map2:     New[string, int](),
			expected: true,
		},
		{
			name:     "empty map and nil map should be equal with custom func",
			map1:     New[string, int](),
			map2:     nil,
			expected: true,
		},
		{
			name:     "both empty maps should be equal with custom func",
			map1:     New[string, int](),
			map2:     New[string, int](),
			expected: true,
		},
		{
			name: "maps with same values should be equal with custom func",
			map1: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			map2: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.map1.IsEqualFunc(tt.map2, customEqualFunc)
			assert.Equal(t, tt.expected, actual, "maps should match expected equality with custom func")
		})
	}
}

func TestMap_IsEqualFunc_Error(t *testing.T) {
	t.Parallel()

	customEqualFunc := func(a, b int) bool {
		return a == b
	}

	tests := []struct {
		name     string
		map1     *Map[string, int]
		map2     *Map[string, int]
		expected bool
	}{
		{
			name:     "nil map vs non-empty map should not be equal with custom func",
			map1:     nil,
			map2:     New(NewElem("key1", 1)),
			expected: false,
		},
		{
			name: "maps with different values should not be equal with custom func",
			map1: New(
				NewElem("key1", 1),
				NewElem("key2", 2),
			),
			map2: New(
				NewElem("key1", 1),
				NewElem("key2", 3),
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.map1.IsEqualFunc(tt.map2, customEqualFunc)
			assert.Equal(t, tt.expected, actual, "maps should match expected equality with custom func")
		})
	}
}

func TestMap_IsEqualFunc_WithCustomLogic(t *testing.T) {
	t.Parallel()

	// Custom function that considers all positive numbers equal
	customEqualFunc := func(a, b int) bool {
		return (a > 0 && b > 0) || a == b
	}

	t.Run("custom logic treats positive numbers as equal", func(t *testing.T) {
		t.Parallel()
		map1 := New(
			NewElem("key1", 1),
			NewElem("key2", 5),
		)
		map2 := New(
			NewElem("key1", 3),
			NewElem("key2", 7),
		)

		actual := map1.IsEqualFunc(map2, customEqualFunc)
		assert.True(t, actual, "maps with positive values should be equal with custom func")
	})

	t.Run("custom logic treats zero and negative numbers strictly", func(t *testing.T) {
		t.Parallel()
		map1 := New(
			NewElem("key1", 0),
			NewElem("key2", -1),
		)
		map2 := New(
			NewElem("key1", 0),
			NewElem("key2", -2),
		)

		actual := map1.IsEqualFunc(map2, customEqualFunc)
		assert.False(t, actual, "maps with different negative values should not be equal with custom func")
	})
}
