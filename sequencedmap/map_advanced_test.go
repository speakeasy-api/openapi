package sequencedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMap_NavigateWithKey_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupMap      *Map[string, int]
		key           string
		expectedValue any
		expectError   bool
	}{
		{
			name: "navigate with existing key",
			setupMap: New(
				NewElem("test", 42),
			),
			key:           "test",
			expectedValue: 42,
			expectError:   false,
		},
		{
			name: "navigate with non-existent key",
			setupMap: New(
				NewElem("test", 42),
			),
			key:         "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			value, err := tt.setupMap.NavigateWithKey(tt.key)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestMap_NavigateWithKey_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMap    any
		key         string
		expectError bool
	}{
		{
			name:        "nil map returns error",
			setupMap:    (*Map[string, int])(nil),
			key:         "test",
			expectError: true,
		},
		{
			name:        "non-string key type returns error",
			setupMap:    New[int, string](),
			key:         "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			switch m := tt.setupMap.(type) {
			case *Map[string, int]:
				_, err := m.NavigateWithKey(tt.key)
				if tt.expectError {
					require.Error(t, err)
				}
			case *Map[int, string]:
				_, err := m.NavigateWithKey(tt.key)
				if tt.expectError {
					require.Error(t, err)
				}
			}
		})
	}
}

func TestMap_UnmarshalYAML_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yamlData string
		expected map[string]int
	}{
		{
			name:     "simple mapping",
			yamlData: "key1: 1\nkey2: 2",
			expected: map[string]int{
				"key1": 1,
				"key2": 2,
			},
		},
		{
			name:     "empty mapping",
			yamlData: "{}",
			expected: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlData), &node)
			require.NoError(t, err)

			// Get the actual mapping node (first child of document node)
			var mappingNode *yaml.Node
			if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
				mappingNode = node.Content[0]
			} else {
				mappingNode = &node
			}

			m := New[string, int]()
			err = m.UnmarshalYAML(mappingNode)
			require.NoError(t, err)

			// Verify all expected key-value pairs
			for key, expectedValue := range tt.expected {
				actualValue, found := m.Get(key)
				assert.True(t, found, "key %s should be found", key)
				assert.Equal(t, expectedValue, actualValue, "value for key %s should match", key)
			}

			assert.Equal(t, len(tt.expected), m.Len(), "map length should match expected")
		})
	}
}

func TestMap_UnmarshalYAML_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nodeKind yaml.Kind
	}{
		{
			name:     "scalar node returns error",
			nodeKind: yaml.ScalarNode,
		},
		{
			name:     "sequence node returns error",
			nodeKind: yaml.SequenceNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &yaml.Node{
				Kind: tt.nodeKind,
			}

			m := New[string, int]()
			err := m.UnmarshalYAML(node)
			assert.Error(t, err)
		})
	}
}

func TestMap_MarshalYAML_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elements []*Element[string, int]
	}{
		{
			name:     "nil map returns nil",
			elements: nil,
		},
		{
			name:     "empty map returns empty content",
			elements: []*Element[string, int]{},
		},
		{
			name: "map with elements returns YAML content",
			elements: []*Element[string, int]{
				NewElem("key1", 1),
				NewElem("key2", 2),
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

			result, err := m.MarshalYAML()
			require.NoError(t, err)

			if tt.elements == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestCompareKeys_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "a less than b",
			a:        "apple",
			b:        "banana",
			expected: -1,
		},
		{
			name:     "a greater than b",
			a:        "banana",
			b:        "apple",
			expected: 1,
		},
		{
			name:     "a equals b",
			a:        "apple",
			b:        "apple",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := compareKeys(tt.a, tt.b)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMap_AddAny_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         any
		value       any
		expectedLen int
	}{
		{
			name:        "add with correct types",
			key:         "test",
			value:       42,
			expectedLen: 1,
		},
		{
			name:        "add with wrong key type",
			key:         123,
			value:       42,
			expectedLen: 0,
		},
		{
			name:        "add with wrong value type",
			key:         "test",
			value:       "not an int",
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[string, int]()
			m.AddAny(tt.key, tt.value)
			assert.Equal(t, tt.expectedLen, m.Len())
		})
	}
}

func TestMap_Init_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupMap     func() *Map[string, int]
		expectedInit bool
	}{
		{
			name: "init uninitialized map",
			setupMap: func() *Map[string, int] {
				return &Map[string, int]{}
			},
			expectedInit: true,
		},
		{
			name: "init already initialized map",
			setupMap: func() *Map[string, int] {
				return New[string, int]()
			},
			expectedInit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setupMap()
			m.Init()
			assert.Equal(t, tt.expectedInit, m.IsInitialized())
		})
	}
}

func TestMap_IsInitialized_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupMap func() *Map[string, int]
		expected bool
	}{
		{
			name: "nil map is not initialized",
			setupMap: func() *Map[string, int] {
				return nil
			},
			expected: false,
		},
		{
			name: "empty struct is not initialized",
			setupMap: func() *Map[string, int] {
				return &Map[string, int]{}
			},
			expected: false,
		},
		{
			name: "new map is initialized",
			setupMap: func() *Map[string, int] {
				return New[string, int]()
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setupMap()
			actual := m.IsInitialized()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNewElem_Success(t *testing.T) {
	t.Parallel()

	elem := NewElem("test", 42)

	assert.Equal(t, "test", elem.Key)
	assert.Equal(t, 42, elem.Value)
}
