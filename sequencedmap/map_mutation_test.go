package sequencedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAll_AddDuringIteration_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		initialKeys    []string
		addAtKey       string
		newKey         string
		newValue       int
		expectedKeys   []string
		expectedValues []int
	}{
		{
			name:           "add element during iteration",
			initialKeys:    []string{"a", "b", "c"},
			addAtKey:       "b",
			newKey:         "d",
			newValue:       4,
			expectedKeys:   []string{"a", "b", "c"}, // New elements are not included in current iteration
			expectedValues: []int{1, 2, 3},
		},
		{
			name:           "add element at beginning during iteration",
			initialKeys:    []string{"a", "b", "c"},
			addAtKey:       "a",
			newKey:         "z",
			newValue:       26,
			expectedKeys:   []string{"a", "b", "c"}, // New elements are not included in current iteration
			expectedValues: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()

			// Initialize map
			for i, key := range tt.initialKeys {
				m.Set(key, i+1)
			}

			var actualKeys []string
			var actualValues []int

			// Iterate and add during iteration
			for key, value := range m.All() {
				actualKeys = append(actualKeys, key)
				actualValues = append(actualValues, value)

				// Add new element when we encounter the trigger key
				if key == tt.addAtKey {
					m.Set(tt.newKey, tt.newValue)
				}
			}

			// The new element should NOT be included in current iteration (snapshot behavior)
			assert.Equal(t, tt.expectedKeys, actualKeys, "keys should match expected order")
			assert.Equal(t, tt.expectedValues, actualValues, "values should match expected order")

			// But the new element should exist in the map for future iterations
			value, exists := m.Get(tt.newKey)
			assert.True(t, exists, "new element should exist in map")
			assert.Equal(t, tt.newValue, value, "new element should have correct value")
		})
	}
}

func TestAll_RemoveDuringIteration_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		initialKeys    []string
		removeAtKey    string
		keyToRemove    string
		expectedKeys   []string
		expectedValues []int
	}{
		{
			name:           "remove later element during iteration",
			initialKeys:    []string{"a", "b", "c", "d"},
			removeAtKey:    "b",
			keyToRemove:    "d",
			expectedKeys:   []string{"a", "b", "c"},
			expectedValues: []int{1, 2, 3},
		},
		{
			name:           "remove earlier element during iteration",
			initialKeys:    []string{"a", "b", "c", "d"},
			removeAtKey:    "c",
			keyToRemove:    "a",
			expectedKeys:   []string{"a", "b", "c", "d"},
			expectedValues: []int{1, 2, 3, 4},
		},
		{
			name:           "remove current element during iteration",
			initialKeys:    []string{"a", "b", "c", "d"},
			removeAtKey:    "b",
			keyToRemove:    "b",
			expectedKeys:   []string{"a", "b", "c", "d"},
			expectedValues: []int{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()

			// Initialize map
			for i, key := range tt.initialKeys {
				m.Set(key, i+1)
			}

			var actualKeys []string
			var actualValues []int

			// Iterate and remove during iteration
			for key, value := range m.All() {
				actualKeys = append(actualKeys, key)
				actualValues = append(actualValues, value)

				// Remove element when we encounter the trigger key
				if key == tt.removeAtKey {
					m.Delete(tt.keyToRemove)
				}
			}

			assert.Equal(t, tt.expectedKeys, actualKeys, "keys should match expected order")
			assert.Equal(t, tt.expectedValues, actualValues, "values should match expected order")
		})
	}
}

func TestAll_MixedMutationsDuringIteration_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		initialKeys []string
		operations  []struct {
			atKey     string
			operation string // "add" or "remove"
			key       string
			value     int
		}
		expectedKeys   []string
		expectedValues []int
	}{
		{
			name:        "add and remove during iteration",
			initialKeys: []string{"a", "b", "c"},
			operations: []struct {
				atKey     string
				operation string
				key       string
				value     int
			}{
				{atKey: "a", operation: "add", key: "d", value: 4},
				{atKey: "b", operation: "remove", key: "c", value: 0},
				{atKey: "c", operation: "add", key: "e", value: 5},
			},
			expectedKeys:   []string{"a", "b"}, // Only elements from snapshot that still exist
			expectedValues: []int{1, 2},        // "c" is removed during iteration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()

			// Initialize map
			for i, key := range tt.initialKeys {
				m.Set(key, i+1)
			}

			var actualKeys []string
			var actualValues []int

			// Iterate and perform operations during iteration
			for key, value := range m.All() {
				actualKeys = append(actualKeys, key)
				actualValues = append(actualValues, value)

				// Perform operations when we encounter trigger keys
				for _, op := range tt.operations {
					if key == op.atKey {
						switch op.operation {
						case "add":
							m.Set(op.key, op.value)
						case "remove":
							m.Delete(op.key)
						}
					}
				}
			}

			assert.Equal(t, tt.expectedKeys, actualKeys, "keys should match expected order")
			assert.Equal(t, tt.expectedValues, actualValues, "values should match expected order")

			// Verify that operations were applied to the map
			assert.True(t, m.Has("d"), "added element 'd' should exist in map")
			assert.False(t, m.Has("c"), "removed element 'c' should not exist in map")
			// Element "e" is not added because we never encounter "c" in iteration (it was removed)
			assert.False(t, m.Has("e"), "element 'e' should not exist because 'c' was removed before we could encounter it")
		})
	}
}

func TestAllOrdered_MutationDuringIteration_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		order       OrderType
		initialKeys []string
		addKey      string
		addValue    int
		addAtKey    string
	}{
		{
			name:        "add during OrderAdded iteration",
			order:       OrderAdded,
			initialKeys: []string{"a", "b", "c"},
			addKey:      "d",
			addValue:    4,
			addAtKey:    "b",
		},
		{
			name:        "add during OrderAddedReverse iteration",
			order:       OrderAddedReverse,
			initialKeys: []string{"a", "b", "c"},
			addKey:      "d",
			addValue:    4,
			addAtKey:    "b",
		},
		{
			name:        "add during OrderKeyAsc iteration",
			order:       OrderKeyAsc,
			initialKeys: []string{"c", "a", "b"},
			addKey:      "d",
			addValue:    4,
			addAtKey:    "b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()

			// Initialize map
			for i, key := range tt.initialKeys {
				m.Set(key, i+1)
			}

			var actualKeys []string
			var actualValues []int

			// Iterate and add during iteration
			for key, value := range m.AllOrdered(tt.order) {
				actualKeys = append(actualKeys, key)
				actualValues = append(actualValues, value)

				// Add new element when we encounter the trigger key
				if key == tt.addAtKey {
					m.Set(tt.addKey, tt.addValue)
				}
			}

			// Should complete iteration without panic or corruption
			assert.NotEmpty(t, actualKeys, "should have collected keys")
			assert.Len(t, actualValues, len(actualKeys), "keys and values should have same length")
		})
	}
}

func TestKeys_MutationDuringIteration_Success(t *testing.T) {
	t.Parallel()

	t.Run("add key during Keys iteration", func(t *testing.T) {
		t.Parallel()
		m := New[string, int]()
		m.Set("a", 1)
		m.Set("b", 2)
		m.Set("c", 3)

		var actualKeys []string

		for key := range m.Keys() {
			actualKeys = append(actualKeys, key)

			// Add new element during iteration
			if key == "b" {
				m.Set("d", 4)
			}
		}

		// Should NOT include the new key in current iteration (snapshot behavior)
		expectedKeys := []string{"a", "b", "c"}
		assert.Equal(t, expectedKeys, actualKeys, "keys should not include newly added key in current iteration")

		// But the new key should exist in the map for future iterations
		assert.True(t, m.Has("d"), "new key should exist in map")
	})

	t.Run("remove key during Keys iteration", func(t *testing.T) {
		t.Parallel()
		m := New[string, int]()
		m.Set("a", 1)
		m.Set("b", 2)
		m.Set("c", 3)
		m.Set("d", 4)

		var actualKeys []string

		for key := range m.Keys() {
			actualKeys = append(actualKeys, key)

			// Remove later element during iteration
			if key == "b" {
				m.Delete("d")
			}
		}

		// Should not include the removed key
		expectedKeys := []string{"a", "b", "c"}
		assert.Equal(t, expectedKeys, actualKeys, "keys should not include removed key")
	})
}

func TestValues_MutationDuringIteration_Success(t *testing.T) {
	t.Parallel()

	t.Run("add value during Values iteration", func(t *testing.T) {
		t.Parallel()
		m := New[string, int]()
		m.Set("a", 1)
		m.Set("b", 2)
		m.Set("c", 3)

		var actualValues []int
		keyCount := 0

		for value := range m.Values() {
			actualValues = append(actualValues, value)
			keyCount++

			// Add new element during iteration
			if keyCount == 2 { // After second element
				m.Set("d", 4)
			}
		}

		// Should NOT include the new value in current iteration (snapshot behavior)
		expectedValues := []int{1, 2, 3}
		assert.Equal(t, expectedValues, actualValues, "values should not include newly added value in current iteration")

		// But the new value should exist in the map for future iterations
		value, exists := m.Get("d")
		assert.True(t, exists, "new element should exist in map")
		assert.Equal(t, 4, value, "new element should have correct value")
	})
}
