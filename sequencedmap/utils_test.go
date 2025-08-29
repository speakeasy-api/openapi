package sequencedmap_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
)

func TestLen_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setupMap func() *sequencedmap.Map[string, int]
		expected int
	}{
		{
			name: "nil map",
			setupMap: func() *sequencedmap.Map[string, int] {
				return nil
			},
			expected: 0,
		},
		{
			name: "empty map",
			setupMap: func() *sequencedmap.Map[string, int] {
				return sequencedmap.New[string, int]()
			},
			expected: 0,
		},
		{
			name: "map with one element",
			setupMap: func() *sequencedmap.Map[string, int] {
				m := sequencedmap.New[string, int]()
				m.Set("key1", 1)
				return m
			},
			expected: 1,
		},
		{
			name: "map with multiple elements",
			setupMap: func() *sequencedmap.Map[string, int] {
				m := sequencedmap.New[string, int]()
				m.Set("key1", 1)
				m.Set("key2", 2)
				m.Set("key3", 3)
				return m
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setupMap()
			result := sequencedmap.Len(m)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFrom_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    map[string]int
		expected map[string]int
	}{
		{
			name:     "empty sequence",
			input:    map[string]int{},
			expected: map[string]int{},
		},
		{
			name: "single element",
			input: map[string]int{
				"key1": 1,
			},
			expected: map[string]int{
				"key1": 1,
			},
		},
		{
			name: "multiple elements",
			input: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
			expected: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a sequence from the input map
			seq := func(yield func(string, int) bool) {
				for k, v := range tt.input {
					if !yield(k, v) {
						return
					}
				}
			}

			result := sequencedmap.From(seq)

			// Verify the result contains all expected elements
			assert.Equal(t, len(tt.expected), result.Len())

			for expectedKey, expectedValue := range tt.expected {
				actualValue, found := result.Get(expectedKey)
				assert.True(t, found, "Expected key %s to be found", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Expected value for key %s", expectedKey)
			}
		})
	}
}

func TestFrom_WithOrderedSequence_Success(t *testing.T) {
	t.Parallel()
	// Test that From preserves the order of elements as they are yielded
	seq := func(yield func(string, int) bool) {
		// Yield in a specific order
		if !yield("first", 1) {
			return
		}
		if !yield("second", 2) {
			return
		}
		if !yield("third", 3) {
			return
		}
	}

	result := sequencedmap.From(seq)

	// Verify the order is preserved
	assert.Equal(t, 3, result.Len())

	// Check the order by iterating through the map
	expectedOrder := []struct {
		key   string
		value int
	}{
		{"first", 1},
		{"second", 2},
		{"third", 3},
	}

	i := 0
	for key, value := range result.All() {
		assert.Equal(t, expectedOrder[i].key, key)
		assert.Equal(t, expectedOrder[i].value, value)
		i++
	}
}

func TestFrom_WithDuplicateKeys_Success(t *testing.T) {
	t.Parallel()
	// Test that From handles duplicate keys correctly (last one wins)
	seq := func(yield func(string, int) bool) {
		if !yield("key1", 1) {
			return
		}
		if !yield("key2", 2) {
			return
		}
		if !yield("key1", 10) { // Duplicate key with different value
			return
		}
	}

	result := sequencedmap.From(seq)

	// Should have 2 elements (key1 and key2)
	assert.Equal(t, 2, result.Len())

	// key1 should have the last value (10)
	value1, found1 := result.Get("key1")
	assert.True(t, found1)
	assert.Equal(t, 10, value1)

	// key2 should have its original value
	value2, found2 := result.Get("key2")
	assert.True(t, found2)
	assert.Equal(t, 2, value2)
}

func TestFrom_WithEarlyTermination_Success(t *testing.T) {
	t.Parallel()
	// Test that From handles early termination of the sequence
	// The consumer (From function) can signal early termination by returning false from yield
	seq := func(yield func(string, int) bool) {
		// All elements will be yielded since the From function doesn't terminate early
		yield("key1", 1)
		yield("key2", 2)
		yield("key3", 3)
	}

	result := sequencedmap.From(seq)

	// Should have all 3 elements since From doesn't terminate early
	assert.Equal(t, 3, result.Len())

	// Should have all keys
	_, found1 := result.Get("key1")
	assert.True(t, found1)

	_, found2 := result.Get("key2")
	assert.True(t, found2)

	_, found3 := result.Get("key3")
	assert.True(t, found3)
}

func TestFrom_WithDifferentTypes_Success(t *testing.T) {
	t.Parallel()
	// Test From with different key and value types
	seq := func(yield func(int, string) bool) {
		if !yield(1, "one") {
			return
		}
		if !yield(2, "two") {
			return
		}
		if !yield(3, "three") {
			return
		}
	}

	result := sequencedmap.From(seq)

	assert.Equal(t, 3, result.Len())

	value1, found1 := result.Get(1)
	assert.True(t, found1)
	assert.Equal(t, "one", value1)

	value2, found2 := result.Get(2)
	assert.True(t, found2)
	assert.Equal(t, "two", value2)

	value3, found3 := result.Get(3)
	assert.True(t, found3)
	assert.Equal(t, "three", value3)
}
