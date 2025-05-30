package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func Test_Node_GetValue_Success(t *testing.T) {
	node := marshaller.Node[string]{
		Value: "test-value",
	}

	result := node.GetValue()
	assert.Equal(t, "test-value", result)
}

func Test_Node_GetKeyNodeOrRoot_Success(t *testing.T) {
	rootNode := testutils.CreateStringYamlNode("root", 1, 1)
	keyNode := testutils.CreateStringYamlNode("key", 2, 2)

	tests := []struct {
		name     string
		node     marshaller.Node[string]
		expected *yaml.Node
	}{
		{
			name: "present with key node",
			node: marshaller.Node[string]{
				Present: true,
				KeyNode: keyNode,
			},
			expected: keyNode,
		},
		{
			name: "present without key node",
			node: marshaller.Node[string]{
				Present: true,
				KeyNode: nil,
			},
			expected: rootNode,
		},
		{
			name: "not present",
			node: marshaller.Node[string]{
				Present: false,
				KeyNode: keyNode,
			},
			expected: rootNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetKeyNodeOrRoot(rootNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_Node_GetValueNodeOrRoot_Success(t *testing.T) {
	rootNode := testutils.CreateStringYamlNode("root", 1, 1)
	valueNode := testutils.CreateStringYamlNode("value", 2, 2)

	tests := []struct {
		name     string
		node     marshaller.Node[string]
		expected *yaml.Node
	}{
		{
			name: "present with value node",
			node: marshaller.Node[string]{
				Present:   true,
				ValueNode: valueNode,
			},
			expected: valueNode,
		},
		{
			name: "present without value node",
			node: marshaller.Node[string]{
				Present:   true,
				ValueNode: nil,
			},
			expected: rootNode,
		},
		{
			name: "not present",
			node: marshaller.Node[string]{
				Present:   false,
				ValueNode: valueNode,
			},
			expected: rootNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetValueNodeOrRoot(rootNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_Node_GetSliceValueNodeOrRoot_Success(t *testing.T) {
	rootNode := testutils.CreateStringYamlNode("root", 1, 1)
	item1 := testutils.CreateStringYamlNode("item1", 2, 2)
	item2 := testutils.CreateStringYamlNode("item2", 3, 3)
	sliceNode := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{item1, item2},
	}

	tests := []struct {
		name     string
		node     marshaller.Node[[]string]
		idx      int
		expected *yaml.Node
	}{
		{
			name: "valid index",
			node: marshaller.Node[[]string]{
				Present:   true,
				ValueNode: sliceNode,
			},
			idx:      0,
			expected: item1,
		},
		{
			name: "valid index 2",
			node: marshaller.Node[[]string]{
				Present:   true,
				ValueNode: sliceNode,
			},
			idx:      1,
			expected: item2,
		},
		{
			name: "negative index",
			node: marshaller.Node[[]string]{
				Present:   true,
				ValueNode: sliceNode,
			},
			idx:      -1,
			expected: sliceNode,
		},
		{
			name: "index out of bounds",
			node: marshaller.Node[[]string]{
				Present:   true,
				ValueNode: sliceNode,
			},
			idx:      5,
			expected: sliceNode,
		},
		{
			name: "not present",
			node: marshaller.Node[[]string]{
				Present:   false,
				ValueNode: sliceNode,
			},
			idx:      0,
			expected: rootNode,
		},
		{
			name: "nil value node",
			node: marshaller.Node[[]string]{
				Present:   true,
				ValueNode: nil,
			},
			idx:      0,
			expected: rootNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetSliceValueNodeOrRoot(tt.idx, rootNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_Node_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	rootNode := testutils.CreateStringYamlNode("root", 1, 1)
	key1 := testutils.CreateStringYamlNode("key1", 2, 2)
	value1 := testutils.CreateStringYamlNode("value1", 2, 8)
	key2 := testutils.CreateStringYamlNode("key2", 3, 2)
	value2 := testutils.CreateStringYamlNode("value2", 3, 8)
	mapNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{key1, value1, key2, value2},
	}

	tests := []struct {
		name     string
		node     marshaller.Node[map[string]string]
		key      string
		expected *yaml.Node
	}{
		{
			name: "existing key",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: mapNode,
			},
			key:      "key1",
			expected: key1,
		},
		{
			name: "existing key 2",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: mapNode,
			},
			key:      "key2",
			expected: key2,
		},
		{
			name: "non-existing key",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: mapNode,
			},
			key:      "key3",
			expected: mapNode,
		},
		{
			name: "not present",
			node: marshaller.Node[map[string]string]{
				Present:   false,
				ValueNode: mapNode,
			},
			key:      "key1",
			expected: rootNode,
		},
		{
			name: "nil value node",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: nil,
			},
			key:      "key1",
			expected: rootNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_Node_GetMapValueNodeOrRoot_Success(t *testing.T) {
	rootNode := testutils.CreateStringYamlNode("root", 1, 1)
	key1 := testutils.CreateStringYamlNode("key1", 2, 2)
	value1 := testutils.CreateStringYamlNode("value1", 2, 8)
	key2 := testutils.CreateStringYamlNode("key2", 3, 2)
	value2 := testutils.CreateStringYamlNode("value2", 3, 8)
	mapNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{key1, value1, key2, value2},
	}

	tests := []struct {
		name     string
		node     marshaller.Node[map[string]string]
		key      string
		expected *yaml.Node
	}{
		{
			name: "existing key",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: mapNode,
			},
			key:      "key1",
			expected: value1,
		},
		{
			name: "existing key 2",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: mapNode,
			},
			key:      "key2",
			expected: value2,
		},
		{
			name: "non-existing key",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: mapNode,
			},
			key:      "key3",
			expected: mapNode,
		},
		{
			name: "not present",
			node: marshaller.Node[map[string]string]{
				Present:   false,
				ValueNode: mapNode,
			},
			key:      "key1",
			expected: rootNode,
		},
		{
			name: "nil value node",
			node: marshaller.Node[map[string]string]{
				Present:   true,
				ValueNode: nil,
			},
			key:      "key1",
			expected: rootNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetMapValueNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_Node_GetNavigableNode_Success(t *testing.T) {
	node := marshaller.Node[string]{
		Value: "test-value",
	}

	result, err := node.GetNavigableNode()
	assert.NoError(t, err)
	assert.Equal(t, "test-value", result)
}