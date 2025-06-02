package marshaller_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_UnmarshalMapping_Error(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		target      any
		expectedErr string
	}{
		{
			name:        "unsupported map type",
			yaml:        `key: value`,
			target:      &map[string]string{},
			expectedErr: "currently unsupported out kind: map",
		},
		{
			name:        "invalid target type",
			yaml:        `key: value`,
			target:      &[]string{},
			expectedErr: "expected struct or map, got slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			err = marshaller.Unmarshal(context.Background(), node.Content[0], tt.target)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// Simple model for testing
type SimpleModel struct {
	Name  string `yaml:"name"`
	Value int    `yaml:"value"`
}

func Test_UnmarshalMapping_SequencedMap_Success(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		target   any
		validate func(t *testing.T, target any)
	}{
		{
			name: "simple string values",
			yaml: `key1: value1
key2: value2
key3: value3`,
			target: sequencedmap.New[string, string](),
			validate: func(t *testing.T, target any) {
				sm := target.(*sequencedmap.Map[string, string])

				assert.Equal(t, 3, sm.Len())

				// Check values
				val1, ok := sm.Get("key1")
				require.True(t, ok)
				assert.Equal(t, "value1", val1)

				val2, ok := sm.Get("key2")
				require.True(t, ok)
				assert.Equal(t, "value2", val2)

				val3, ok := sm.Get("key3")
				require.True(t, ok)
				assert.Equal(t, "value3", val3)

				// Verify order is maintained
				keys := make([]string, 0)
				for key := range sm.Keys() {
					keys = append(keys, key)
				}
				assert.Equal(t, []string{"key1", "key2", "key3"}, keys)
			},
		},
		{
			name: "integer values",
			yaml: `item1: 42
item2: 84
item3: 126`,
			target: sequencedmap.New[string, int](),
			validate: func(t *testing.T, target any) {
				sm := target.(*sequencedmap.Map[string, int])

				assert.Equal(t, 3, sm.Len())

				// Check values
				val1, ok := sm.Get("item1")
				require.True(t, ok)
				assert.Equal(t, 42, val1)

				val2, ok := sm.Get("item2")
				require.True(t, ok)
				assert.Equal(t, 84, val2)

				val3, ok := sm.Get("item3")
				require.True(t, ok)
				assert.Equal(t, 126, val3)

				// Verify order is maintained
				keys := make([]string, 0)
				for key := range sm.Keys() {
					keys = append(keys, key)
				}
				assert.Equal(t, []string{"item1", "item2", "item3"}, keys)
			},
		},
		{
			name:   "empty map",
			yaml:   `{}`,
			target: sequencedmap.New[string, string](),
			validate: func(t *testing.T, target any) {
				sm := target.(*sequencedmap.Map[string, string])
				assert.Equal(t, 0, sm.Len())
			},
		},
		{
			name:   "single key-value pair",
			yaml:   `single: value`,
			target: sequencedmap.New[string, string](),
			validate: func(t *testing.T, target any) {
				sm := target.(*sequencedmap.Map[string, string])
				assert.Equal(t, 1, sm.Len())

				val, ok := sm.Get("single")
				require.True(t, ok)
				assert.Equal(t, "value", val)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			// Use a pointer to the target to avoid the interface detection issue
			err = marshaller.Unmarshal(context.Background(), node.Content[0], &tt.target)
			require.NoError(t, err)

			tt.validate(t, tt.target)
		})
	}
}

func Test_UnmarshalMapping_SequencedMap_PlainStruct_Success(t *testing.T) {
	t.Skip("Skipping until we can figure out how to handle plain structs")

	yamlData := `model1:
  name: "test1"
  value: 42`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	require.NoError(t, err)

	target := sequencedmap.New[string, SimpleModel]()
	err = marshaller.Unmarshal(context.Background(), node.Content[0], &target)

	// Should now work for plain structs
	require.NoError(t, err)

	// Verify the data was unmarshaled correctly
	assert.Equal(t, 1, target.Len())

	model, ok := target.Get("model1")
	require.True(t, ok)
	assert.Equal(t, "test1", model.Name)
	assert.Equal(t, 42, model.Value)
}

func Test_UnmarshalSequence_Error(t *testing.T) {
	testYaml := `
- item1
- item2
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	target := &map[string]string{}
	err = marshaller.Unmarshal(context.Background(), node.Content[0], target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected slice, got map")
}

type TestUnmarshallableStruct struct {
	Value string
}

func (t *TestUnmarshallableStruct) Unmarshal(ctx context.Context, node *yaml.Node) error {
	t.Value = "unmarshalled-" + node.Value
	return nil
}

func Test_UnmarshalNode_CustomUnmarshaller_Success(t *testing.T) {
	testYaml := `test-value`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	target := &TestUnmarshallableStruct{}
	err = marshaller.Unmarshal(context.Background(), node.Content[0], target)
	require.NoError(t, err)

	assert.Equal(t, "unmarshalled-test-value", target.Value)
}

type TestCoreModelStruct struct {
	marshaller.Model[TestCoreModelStructCore]
	Value string
}

type TestCoreModelStructCore struct {
	marshaller.CoreModel
	Value marshaller.Node[string] `key:"value"`
}

func Test_UnmarshalNode_ScalarTypes_Success(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		target   any
		expected any
	}{
		{
			name:     "string",
			yaml:     `"test-string"`,
			target:   new(string),
			expected: "test-string",
		},
		{
			name:     "int",
			yaml:     `42`,
			target:   new(int),
			expected: 42,
		},
		{
			name:     "bool",
			yaml:     `true`,
			target:   new(bool),
			expected: true,
		},
		{
			name:     "float",
			yaml:     `3.14`,
			target:   new(float64),
			expected: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			err = marshaller.Unmarshal(context.Background(), node.Content[0], tt.target)
			require.NoError(t, err)

			switch target := tt.target.(type) {
			case *string:
				assert.Equal(t, tt.expected, *target)
			case *int:
				assert.Equal(t, tt.expected, *target)
			case *bool:
				assert.Equal(t, tt.expected, *target)
			case *float64:
				assert.Equal(t, tt.expected, *target)
			}
		})
	}
}

func Test_UnmarshalNode_InvalidNode_Error(t *testing.T) {
	// Create an invalid node type
	node := &yaml.Node{
		Kind:  yaml.Kind(99), // Invalid kind
		Value: "test",
	}

	target := new(string)
	err := marshaller.Unmarshal(context.Background(), node, target)
	require.Error(t, err)
}
