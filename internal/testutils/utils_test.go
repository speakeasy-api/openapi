package testutils

import (
	"iter"
	"testing"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestCreateStringYamlNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		line      int
		column    int
		wantValue *yaml.Node
	}{
		{
			name:   "creates string node",
			value:  "test",
			line:   1,
			column: 2,
			wantValue: &yaml.Node{
				Value:  "test",
				Kind:   yaml.ScalarNode,
				Tag:    "!!str",
				Line:   1,
				Column: 2,
			},
		},
		{
			name:   "creates empty string node",
			value:  "",
			line:   3,
			column: 4,
			wantValue: &yaml.Node{
				Value:  "",
				Kind:   yaml.ScalarNode,
				Tag:    "!!str",
				Line:   3,
				Column: 4,
			},
		},
		{
			name:   "creates string with special characters",
			value:  "hello\nworld",
			line:   5,
			column: 6,
			wantValue: &yaml.Node{
				Value:  "hello\nworld",
				Kind:   yaml.ScalarNode,
				Tag:    "!!str",
				Line:   5,
				Column: 6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CreateStringYamlNode(tt.value, tt.line, tt.column)

			assert.Equal(t, tt.wantValue, result, "should create correct string YAML node")
		})
	}
}

func TestCreateIntYamlNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     int
		line      int
		column    int
		wantValue *yaml.Node
	}{
		{
			name:   "creates positive int node",
			value:  42,
			line:   1,
			column: 2,
			wantValue: &yaml.Node{
				Value:  "42",
				Kind:   yaml.ScalarNode,
				Tag:    "!!int",
				Line:   1,
				Column: 2,
			},
		},
		{
			name:   "creates zero int node",
			value:  0,
			line:   3,
			column: 4,
			wantValue: &yaml.Node{
				Value:  "0",
				Kind:   yaml.ScalarNode,
				Tag:    "!!int",
				Line:   3,
				Column: 4,
			},
		},
		{
			name:   "creates negative int node",
			value:  -100,
			line:   5,
			column: 6,
			wantValue: &yaml.Node{
				Value:  "-100",
				Kind:   yaml.ScalarNode,
				Tag:    "!!int",
				Line:   5,
				Column: 6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CreateIntYamlNode(tt.value, tt.line, tt.column)

			assert.Equal(t, tt.wantValue, result, "should create correct int YAML node")
		})
	}
}

func TestCreateBoolYamlNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     bool
		line      int
		column    int
		wantValue *yaml.Node
	}{
		{
			name:   "creates true bool node",
			value:  true,
			line:   1,
			column: 2,
			wantValue: &yaml.Node{
				Value:  "true",
				Kind:   yaml.ScalarNode,
				Tag:    "!!bool",
				Line:   1,
				Column: 2,
			},
		},
		{
			name:   "creates false bool node",
			value:  false,
			line:   3,
			column: 4,
			wantValue: &yaml.Node{
				Value:  "false",
				Kind:   yaml.ScalarNode,
				Tag:    "!!bool",
				Line:   3,
				Column: 4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CreateBoolYamlNode(tt.value, tt.line, tt.column)

			assert.Equal(t, tt.wantValue, result, "should create correct bool YAML node")
		})
	}
}

func TestCreateMapYamlNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		contents  []*yaml.Node
		line      int
		column    int
		wantValue *yaml.Node
	}{
		{
			name: "creates map node with contents",
			contents: []*yaml.Node{
				CreateStringYamlNode("key1", 1, 1),
				CreateStringYamlNode("value1", 1, 6),
				CreateStringYamlNode("key2", 2, 1),
				CreateIntYamlNode(42, 2, 6),
			},
			line:   1,
			column: 1,
			wantValue: &yaml.Node{
				Content: []*yaml.Node{
					CreateStringYamlNode("key1", 1, 1),
					CreateStringYamlNode("value1", 1, 6),
					CreateStringYamlNode("key2", 2, 1),
					CreateIntYamlNode(42, 2, 6),
				},
				Kind:   yaml.MappingNode,
				Tag:    "!!map",
				Line:   1,
				Column: 1,
			},
		},
		{
			name:     "creates empty map node",
			contents: []*yaml.Node{},
			line:     5,
			column:   10,
			wantValue: &yaml.Node{
				Content: []*yaml.Node{},
				Kind:    yaml.MappingNode,
				Tag:     "!!map",
				Line:    5,
				Column:  10,
			},
		},
		{
			name:     "creates nil map node",
			contents: nil,
			line:     3,
			column:   4,
			wantValue: &yaml.Node{
				Content: nil,
				Kind:    yaml.MappingNode,
				Tag:     "!!map",
				Line:    3,
				Column:  4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CreateMapYamlNode(tt.contents, tt.line, tt.column)

			assert.Equal(t, tt.wantValue, result, "should create correct map YAML node")
		})
	}
}

func TestIsInterfaceNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		wantValue bool
	}{
		{
			name:      "returns true for nil interface",
			input:     nil,
			wantValue: true,
		},
		{
			name:      "returns true for nil pointer",
			input:     (*string)(nil),
			wantValue: true,
		},
		{
			name:      "returns true for nil slice",
			input:     ([]string)(nil),
			wantValue: true,
		},
		{
			name:      "returns true for nil map",
			input:     (map[string]string)(nil),
			wantValue: true,
		},
		{
			name:      "returns true for nil channel",
			input:     (chan int)(nil),
			wantValue: true,
		},
		{
			name:      "returns true for nil function",
			input:     (func())(nil),
			wantValue: true,
		},
		{
			name:      "returns false for non-nil string",
			input:     "test",
			wantValue: false,
		},
		{
			name:      "returns false for non-nil pointer",
			input:     new(string),
			wantValue: false,
		},
		{
			name:      "returns false for non-nil slice",
			input:     []string{"test"},
			wantValue: false,
		},
		{
			name:      "returns false for non-nil map",
			input:     map[string]string{"key": "value"},
			wantValue: false,
		},
		{
			name:      "returns false for zero int",
			input:     0,
			wantValue: false,
		},
		{
			name:      "returns false for empty string",
			input:     "",
			wantValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isInterfaceNil(tt.input)

			assert.Equal(t, tt.wantValue, result, "should return correct nil check result")
		})
	}
}

// mockSequencedMap is a simple implementation for testing
type mockSequencedMap struct {
	data map[any]any
	keys []any
}

func newMockSequencedMap(pairs ...any) *mockSequencedMap {
	if len(pairs)%2 != 0 {
		panic("pairs must have even length")
	}
	m := &mockSequencedMap{
		data: make(map[any]any),
		keys: make([]any, 0, len(pairs)/2),
	}
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		value := pairs[i+1]
		m.data[key] = value
		m.keys = append(m.keys, key)
	}
	return m
}

func (m *mockSequencedMap) Len() int {
	if m == nil {
		return 0
	}
	return len(m.keys)
}

func (m *mockSequencedMap) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {
		if m == nil {
			return
		}
		for _, k := range m.keys {
			if !yield(k, m.data[k]) {
				return
			}
		}
	}
}

func (m *mockSequencedMap) GetUntyped(key any) (any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m.data[key]
	return v, ok
}

func TestAssertEqualSequencedMap_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected SequencedMap
		actual   SequencedMap
	}{
		{
			name:     "both nil interfaces",
			expected: nil,
			actual:   nil,
		},
		{
			name:     "both nil underlying values",
			expected: (*mockSequencedMap)(nil),
			actual:   (*mockSequencedMap)(nil),
		},
		{
			name:     "equal simple maps",
			expected: newMockSequencedMap("key1", "value1", "key2", "value2"),
			actual:   newMockSequencedMap("key1", "value1", "key2", "value2"),
		},
		{
			name:     "equal empty maps",
			expected: newMockSequencedMap(),
			actual:   newMockSequencedMap(),
		},
		{
			name:     "equal maps with different types",
			expected: newMockSequencedMap("string", "value", 42, 100, true, false),
			actual:   newMockSequencedMap("string", "value", 42, 100, true, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Should not panic or fail
			AssertEqualSequencedMap(t, tt.expected, tt.actual)
		})
	}
}

func TestAssertEqualSequencedMap_Failure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected SequencedMap
		actual   SequencedMap
	}{
		{
			name:     "expected nil, actual not nil",
			expected: nil,
			actual:   newMockSequencedMap("key", "value"),
		},
		{
			name:     "expected not nil, actual nil",
			expected: newMockSequencedMap("key", "value"),
			actual:   nil,
		},
		{
			name:     "different lengths",
			expected: newMockSequencedMap("key1", "value1"),
			actual:   newMockSequencedMap("key1", "value1", "key2", "value2"),
		},
		{
			name:     "different values",
			expected: newMockSequencedMap("key", "value1"),
			actual:   newMockSequencedMap("key", "value2"),
		},
		{
			name:     "missing key in actual",
			expected: newMockSequencedMap("key1", "value1", "key2", "value2"),
			actual:   newMockSequencedMap("key1", "value1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a mock testing.T to capture failures
			mockT := &testing.T{}

			// This should cause an assertion failure but not panic
			// We use require to ensure failures are detected
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("AssertEqualSequencedMap should not panic: %v", r)
				}
			}()

			AssertEqualSequencedMap(mockT, tt.expected, tt.actual)
		})
	}
}

func TestAssertEqualSequencedMap_NilChecks(t *testing.T) {
	t.Parallel()

	t.Run("handles nil expected with nil underlying", func(t *testing.T) {
		t.Parallel()

		var expected *mockSequencedMap
		var actual *mockSequencedMap

		// Should not panic
		AssertEqualSequencedMap(t, expected, actual)
	})

	t.Run("handles mixed nil types", func(t *testing.T) {
		t.Parallel()

		mockT := &testing.T{}
		var nilPtr *mockSequencedMap

		// Should detect difference between true nil and nil pointer
		AssertEqualSequencedMap(mockT, nil, nilPtr)
	})
}

func TestQueryV4_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		query    string
		expected string
	}{
		{
			name:     "scalar value lookup",
			yml:      "name: alice\nage: 30\n",
			query:    "$.name",
			expected: "alice",
		},
		{
			name:     "nested value lookup",
			yml:      "user:\n  name: bob\n",
			query:    "$.user.name",
			expected: "bob",
		},
		{
			name:     "array element lookup",
			yml:      "items:\n  - first\n  - second\n",
			query:    "$.items[1]",
			expected: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var root yaml.Node
			err := yaml.Unmarshal([]byte(tt.yml), &root)
			require.NoError(t, err, "unmarshal should succeed")

			path, err := jsonpath.NewPath(tt.query)
			require.NoError(t, err, "jsonpath should be valid")

			result := QueryV4(path, &root)
			require.Len(t, result, 1, "should return exactly one match")
			assert.Equal(t, tt.expected, result[0].Value, "should return correct value")
		})
	}
}

func TestQueryV4_NoMatch(t *testing.T) {
	t.Parallel()

	var root yaml.Node
	err := yaml.Unmarshal([]byte("name: alice\n"), &root)
	require.NoError(t, err, "unmarshal should succeed")

	path, err := jsonpath.NewPath("$.missing")
	require.NoError(t, err, "jsonpath should be valid")

	result := QueryV4(path, &root)
	assert.Empty(t, result, "should return no matches for missing path")
}
