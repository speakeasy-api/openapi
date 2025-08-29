package sequencedmap

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElement_GetKey_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		element  *Element[string, int]
		expected string
	}{
		{
			name:     "nil element returns zero value",
			element:  nil,
			expected: "",
		},
		{
			name: "element with key returns key",
			element: &Element[string, int]{
				Key:   "test",
				Value: 42,
			},
			expected: "test",
		},
		{
			name: "element with empty key returns empty string",
			element: &Element[string, int]{
				Key:   "",
				Value: 42,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.element.GetKey()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestElement_GetValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		element  *Element[string, int]
		expected int
	}{
		{
			name:     "nil element returns zero value",
			element:  nil,
			expected: 0,
		},
		{
			name: "element with value returns value",
			element: &Element[string, int]{
				Key:   "test",
				Value: 42,
			},
			expected: 42,
		},
		{
			name: "element with zero value returns zero",
			element: &Element[string, int]{
				Key:   "test",
				Value: 0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.element.GetValue()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMap_NewWithCapacity_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		capacity int
		elements []*Element[string, int]
		expected int
	}{
		{
			name:     "new map with capacity 0",
			capacity: 0,
			elements: nil,
			expected: 0,
		},
		{
			name:     "new map with capacity 5",
			capacity: 5,
			elements: nil,
			expected: 0,
		},
		{
			name:     "new map with capacity and elements",
			capacity: 10,
			elements: []*Element[string, int]{
				NewElem("key1", 1),
				NewElem("key2", 2),
			},
			expected: 2,
		},
		{
			name:     "new map with capacity smaller than elements",
			capacity: 1,
			elements: []*Element[string, int]{
				NewElem("key1", 1),
				NewElem("key2", 2),
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewWithCapacity(tt.capacity, tt.elements...)
			assert.Equal(t, tt.expected, m.Len())
		})
	}
}

func TestMap_SetAny_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		key           any
		value         any
		expectedLen   int
		expectedValue int
		shouldFind    bool
	}{
		{
			name:          "set with correct types",
			key:           "test",
			value:         42,
			expectedLen:   1,
			expectedValue: 42,
			shouldFind:    true,
		},
		{
			name:        "set with wrong key type",
			key:         123,
			value:       42,
			expectedLen: 0,
			shouldFind:  false,
		},
		{
			name:        "set with wrong value type",
			key:         "test",
			value:       "not an int",
			expectedLen: 0,
			shouldFind:  false,
		},
		{
			name:        "set with both wrong types",
			key:         123,
			value:       "not an int",
			expectedLen: 0,
			shouldFind:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()
			m.SetAny(tt.key, tt.value)
			assert.Equal(t, tt.expectedLen, m.Len())

			if tt.shouldFind {
				value, found := m.Get("test")
				assert.True(t, found)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestMap_GetAny_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupKey      string
		setupValue    int
		queryKey      any
		expectedValue any
		expectedFound bool
	}{
		{
			name:          "get with correct key type",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      "test",
			expectedValue: 42,
			expectedFound: true,
		},
		{
			name:          "get with wrong key type",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      123,
			expectedValue: nil,
			expectedFound: false,
		},
		{
			name:          "get non-existent key",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      "nonexistent",
			expectedValue: 0,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()
			m.Set(tt.setupKey, tt.setupValue)

			value, found := m.GetAny(tt.queryKey)
			assert.Equal(t, tt.expectedFound, found)
			if tt.expectedFound {
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestMap_DeleteAny_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupKey    string
		setupValue  int
		deleteKey   any
		expectedLen int
	}{
		{
			name:        "delete with correct key type",
			setupKey:    "test",
			setupValue:  42,
			deleteKey:   "test",
			expectedLen: 0,
		},
		{
			name:        "delete with wrong key type",
			setupKey:    "test",
			setupValue:  42,
			deleteKey:   123,
			expectedLen: 1,
		},
		{
			name:        "delete non-existent key",
			setupKey:    "test",
			setupValue:  42,
			deleteKey:   "nonexistent",
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()
			m.Set(tt.setupKey, tt.setupValue)

			m.DeleteAny(tt.deleteKey)
			assert.Equal(t, tt.expectedLen, m.Len())
		})
	}
}

func TestMap_KeysAny_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
		expected []any
	}{
		{
			name:     "nil map returns empty iterator",
			elements: nil,
			expected: []any{},
		},
		{
			name: "map with elements returns keys",
			elements: []*Element[string, int]{
				NewElem("key1", 1),
				NewElem("key2", 2),
				NewElem("key3", 3),
			},
			expected: []any{"key1", "key2", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.elements != nil {
				m = New(tt.elements...)
			}

			var keys []any
			for key := range m.KeysAny() {
				keys = append(keys, key)
			}

			// Handle nil slice vs empty slice comparison
			if len(tt.expected) == 0 && len(keys) == 0 {
				// Both are empty, consider them equal
				return
			}

			assert.Equal(t, tt.expected, keys)
		})
	}
}

func TestMap_SetUntyped_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         any
		value       any
		expectError bool
		expectedLen int
	}{
		{
			name:        "set with correct types",
			key:         "test",
			value:       42,
			expectError: false,
			expectedLen: 1,
		},
		{
			name:        "set with wrong key type",
			key:         123,
			value:       42,
			expectError: true,
			expectedLen: 0,
		},
		{
			name:        "set with wrong value type",
			key:         "test",
			value:       "not an int",
			expectError: true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()
			err := m.SetUntyped(tt.key, tt.value)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedLen, m.Len())
		})
	}
}

func TestMap_GetUntyped_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupKey      string
		setupValue    int
		queryKey      any
		expectedValue any
		expectedFound bool
	}{
		{
			name:          "get with correct key type",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      "test",
			expectedValue: 42,
			expectedFound: true,
		},
		{
			name:          "get with wrong key type",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      123,
			expectedValue: 0,
			expectedFound: false,
		},
		{
			name:          "get from nil map",
			setupKey:      "",
			setupValue:    0,
			queryKey:      "test",
			expectedValue: 0,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.setupKey != "" {
				m = New[string, int]()
				m.Set(tt.setupKey, tt.setupValue)
			}

			value, found := m.GetUntyped(tt.queryKey)
			assert.Equal(t, tt.expectedFound, found)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestMap_GetOrZero_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupKey      string
		setupValue    int
		queryKey      string
		expectedValue int
	}{
		{
			name:          "get existing key returns value",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      "test",
			expectedValue: 42,
		},
		{
			name:          "get non-existent key returns zero",
			setupKey:      "test",
			setupValue:    42,
			queryKey:      "nonexistent",
			expectedValue: 0,
		},
		{
			name:          "get from nil map returns zero",
			setupKey:      "",
			setupValue:    0,
			queryKey:      "test",
			expectedValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.setupKey != "" {
				m = New[string, int]()
				m.Set(tt.setupKey, tt.setupValue)
			}

			value := m.GetOrZero(tt.queryKey)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestMap_First_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
		expected *Element[string, int]
	}{
		{
			name:     "nil map returns nil",
			elements: nil,
			expected: nil,
		},
		{
			name:     "empty map returns nil",
			elements: []*Element[string, int]{},
			expected: nil,
		},
		{
			name: "map with elements returns first",
			elements: []*Element[string, int]{
				NewElem("first", 1),
				NewElem("second", 2),
			},
			expected: NewElem("first", 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.elements != nil {
				m = New(tt.elements...)
			}

			actual := m.First()
			if tt.expected == nil {
				assert.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				assert.Equal(t, tt.expected.Key, actual.Key)
				assert.Equal(t, tt.expected.Value, actual.Value)
			}
		})
	}
}

func TestMap_Last_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
		expected *Element[string, int]
	}{
		{
			name:     "nil map returns nil",
			elements: nil,
			expected: nil,
		},
		{
			name:     "empty map returns nil",
			elements: []*Element[string, int]{},
			expected: nil,
		},
		{
			name: "map with elements returns last",
			elements: []*Element[string, int]{
				NewElem("first", 1),
				NewElem("last", 2),
			},
			expected: NewElem("last", 2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.elements != nil {
				m = New(tt.elements...)
			}

			actual := m.Last()
			if tt.expected == nil {
				assert.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				assert.Equal(t, tt.expected.Key, actual.Key)
				assert.Equal(t, tt.expected.Value, actual.Value)
			}
		})
	}
}

func TestMap_At_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
		index    int
		expected *Element[string, int]
	}{
		{
			name:     "nil map returns nil",
			elements: nil,
			index:    0,
			expected: nil,
		},
		{
			name:     "empty map returns nil",
			elements: []*Element[string, int]{},
			index:    0,
			expected: nil,
		},
		{
			name: "negative index returns nil",
			elements: []*Element[string, int]{
				NewElem("test", 1),
			},
			index:    -1,
			expected: nil,
		},
		{
			name: "index out of bounds returns nil",
			elements: []*Element[string, int]{
				NewElem("test", 1),
			},
			index:    1,
			expected: nil,
		},
		{
			name: "valid index returns element",
			elements: []*Element[string, int]{
				NewElem("first", 1),
				NewElem("second", 2),
			},
			index:    1,
			expected: NewElem("second", 2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.elements != nil {
				m = New(tt.elements...)
			}

			actual := m.At(tt.index)
			if tt.expected == nil {
				assert.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				assert.Equal(t, tt.expected.Key, actual.Key)
				assert.Equal(t, tt.expected.Value, actual.Value)
			}
		})
	}
}

func TestMap_AllUntyped_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
		expected map[any]any
	}{
		{
			name:     "nil map returns empty iterator",
			elements: nil,
			expected: map[any]any{},
		},
		{
			name: "map with elements returns all",
			elements: []*Element[string, int]{
				NewElem("key1", 1),
				NewElem("key2", 2),
			},
			expected: map[any]any{
				"key1": 1,
				"key2": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.elements != nil {
				m = New(tt.elements...)
			}

			actual := make(map[any]any)
			for key, value := range m.AllUntyped() {
				actual[key] = value
			}

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMap_GetKeyType_Success(t *testing.T) {
	t.Parallel()

	m := New[string, int]()
	keyType := m.GetKeyType()

	assert.Equal(t, reflect.TypeOf(""), keyType)
}

func TestMap_GetValueType_Success(t *testing.T) {
	t.Parallel()

	m := New[string, int]()
	valueType := m.GetValueType()

	assert.Equal(t, reflect.TypeOf(0), valueType)
}

func TestMap_MarshalJSON_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
		expected string
	}{
		{
			name:     "nil map returns null",
			elements: nil,
			expected: "null",
		},
		{
			name:     "empty map returns empty object",
			elements: []*Element[string, int]{},
			expected: "{}",
		},
		{
			name: "map with elements returns JSON object",
			elements: []*Element[string, int]{
				NewElem("key1", 1),
				NewElem("key2", 2),
			},
			expected: `{"key1":1,"key2":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m *Map[string, int]
			if tt.elements != nil {
				m = New(tt.elements...)
			}

			data, err := m.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}
