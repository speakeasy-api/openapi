package yml_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCreateOrUpdateKeyNode_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		key      string
		keyNode  *yaml.Node
		expected string
	}{
		{
			name:     "create new key node",
			key:      "test-key",
			keyNode:  nil,
			expected: "test-key",
		},
		{
			name: "update existing key node",
			key:  "updated-key",
			keyNode: &yaml.Node{
				Value: "old-key",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			expected: "updated-key",
		},
		{
			name: "update alias key node",
			key:  "alias-key",
			keyNode: &yaml.Node{
				Kind: yaml.AliasNode,
				Alias: &yaml.Node{
					Value: "original",
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
				},
			},
			expected: "alias-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			result := yml.CreateOrUpdateKeyNode(ctx, tt.key, tt.keyNode)

			require.NotNil(t, result)
			if tt.keyNode != nil && tt.keyNode.Kind == yaml.AliasNode {
				assert.Equal(t, tt.expected, result.Alias.Value)
				assert.Equal(t, yaml.AliasNode, result.Kind)
			} else {
				resolvedNode := yml.ResolveAlias(result)
				assert.Equal(t, tt.expected, resolvedNode.Value)
				assert.Equal(t, yaml.ScalarNode, result.Kind)
			}
		})
	}
}

func TestCreateOrUpdateScalarNode_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		value     interface{}
		valueNode *yaml.Node
		expected  interface{}
	}{
		{
			name:      "create string node",
			value:     "test-value",
			valueNode: nil,
			expected:  "test-value",
		},
		{
			name:      "create int node",
			value:     42,
			valueNode: nil,
			expected:  "42",
		},
		{
			name:      "create bool node",
			value:     true,
			valueNode: nil,
			expected:  "true",
		},
		{
			name:  "update existing node",
			value: "updated-value",
			valueNode: &yaml.Node{
				Value: "old-value",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			expected: "updated-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			result := yml.CreateOrUpdateScalarNode(ctx, tt.value, tt.valueNode)

			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result.Value)
			assert.Equal(t, yaml.ScalarNode, result.Kind)
		})
	}
}

func TestCreateOrUpdateMapNodeElement_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		key       string
		keyNode   *yaml.Node
		valueNode *yaml.Node
		mapNode   *yaml.Node
		expectNew bool
	}{
		{
			name:    "create new map",
			key:     "new-key",
			keyNode: nil,
			valueNode: &yaml.Node{
				Value: "new-value",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			mapNode:   nil,
			expectNew: true,
		},
		{
			name: "add to existing map",
			key:  "new-key",
			keyNode: &yaml.Node{
				Value: "new-key",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			valueNode: &yaml.Node{
				Value: "new-value",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			mapNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "existing-key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "existing-value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectNew: false,
		},
		{
			name: "update existing key in map",
			key:  "existing-key",
			keyNode: &yaml.Node{
				Value: "existing-key",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			valueNode: &yaml.Node{
				Value: "updated-value",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			mapNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "existing-key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "old-value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectNew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			result := yml.CreateOrUpdateMapNodeElement(ctx, tt.key, tt.keyNode, tt.valueNode, tt.mapNode)

			require.NotNil(t, result)
			assert.Equal(t, yaml.MappingNode, result.Kind)

			if tt.expectNew {
				assert.Len(t, result.Content, 2)
			} else {
				assert.GreaterOrEqual(t, len(result.Content), 2)
			}
		})
	}
}

func TestCreateStringNode_Success(t *testing.T) {
	t.Parallel()
	value := "test-string"
	result := yml.CreateStringNode(value)

	require.NotNil(t, result)
	assert.Equal(t, value, result.Value)
	assert.Equal(t, yaml.ScalarNode, result.Kind)
	assert.Equal(t, "!!str", result.Tag)
}

func TestCreateIntNode_Success(t *testing.T) {
	t.Parallel()
	value := int64(42)
	result := yml.CreateIntNode(value)

	require.NotNil(t, result)
	assert.Equal(t, "42", result.Value)
	assert.Equal(t, yaml.ScalarNode, result.Kind)
	assert.Equal(t, "!!int", result.Tag)
}

func TestCreateFloatNode_Success(t *testing.T) {
	t.Parallel()
	value := 3.14
	result := yml.CreateFloatNode(value)

	require.NotNil(t, result)
	assert.Equal(t, "3.14", result.Value)
	assert.Equal(t, yaml.ScalarNode, result.Kind)
	assert.Equal(t, "!!float", result.Tag)
}

func TestCreateBoolNode_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    bool
		expected string
	}{
		{
			name:     "true value",
			value:    true,
			expected: "true",
		},
		{
			name:     "false value",
			value:    false,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.CreateBoolNode(tt.value)

			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result.Value)
			assert.Equal(t, yaml.ScalarNode, result.Kind)
			assert.Equal(t, "!!bool", result.Tag)
		})
	}
}

func TestCreateMapNode_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	content := []*yaml.Node{
		{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
		{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
	}

	result := yml.CreateMapNode(ctx, content)

	require.NotNil(t, result)
	assert.Equal(t, yaml.MappingNode, result.Kind)
	assert.Equal(t, "!!map", result.Tag)
	assert.Equal(t, content, result.Content)
}

func TestDeleteMapNodeElement_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		key            string
		mapNode        *yaml.Node
		expectedLength int
		shouldFind     bool
	}{
		{
			name: "delete existing key",
			key:  "key1",
			mapNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectedLength: 2,
			shouldFind:     true,
		},
		{
			name: "delete non-existing key",
			key:  "nonexistent",
			mapNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectedLength: 2,
			shouldFind:     false,
		},
		{
			name:           "delete from nil map",
			key:            "key1",
			mapNode:        nil,
			expectedLength: 0,
			shouldFind:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			result := yml.DeleteMapNodeElement(ctx, tt.key, tt.mapNode)

			if tt.mapNode == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Len(t, result.Content, tt.expectedLength)
			}
		})
	}
}

func TestCreateOrUpdateSliceNode_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		elements  []*yaml.Node
		valueNode *yaml.Node
		expectNew bool
	}{
		{
			name: "create new slice",
			elements: []*yaml.Node{
				{Value: "item1", Kind: yaml.ScalarNode, Tag: "!!str"},
				{Value: "item2", Kind: yaml.ScalarNode, Tag: "!!str"},
			},
			valueNode: nil,
			expectNew: true,
		},
		{
			name: "update existing slice",
			elements: []*yaml.Node{
				{Value: "new-item", Kind: yaml.ScalarNode, Tag: "!!str"},
			},
			valueNode: &yaml.Node{
				Kind: yaml.SequenceNode,
				Tag:  "!!seq",
				Content: []*yaml.Node{
					{Value: "old-item", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectNew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			result := yml.CreateOrUpdateSliceNode(ctx, tt.elements, tt.valueNode)

			require.NotNil(t, result)
			assert.Equal(t, yaml.SequenceNode, result.Kind)
			assert.Equal(t, "!!seq", result.Tag)
			assert.Equal(t, tt.elements, result.Content)
		})
	}
}

func TestGetMapElementNodes_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		mapNode     *yaml.Node
		key         string
		expectFound bool
		expectKey   string
		expectValue string
	}{
		{
			name: "find existing key",
			mapNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			key:         "key1",
			expectFound: true,
			expectKey:   "key1",
			expectValue: "value1",
		},
		{
			name: "key not found",
			mapNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			key:         "nonexistent",
			expectFound: false,
		},
		{
			name:        "nil map node",
			mapNode:     nil,
			key:         "key1",
			expectFound: false,
		},
		{
			name: "non-mapping node",
			mapNode: &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag:  "!!str",
			},
			key:         "key1",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			keyNode, valueNode, found := yml.GetMapElementNodes(ctx, tt.mapNode, tt.key)

			assert.Equal(t, tt.expectFound, found)
			if tt.expectFound {
				require.NotNil(t, keyNode)
				require.NotNil(t, valueNode)
				assert.Equal(t, tt.expectKey, keyNode.Value)
				assert.Equal(t, tt.expectValue, valueNode.Value)
			} else {
				assert.Nil(t, keyNode)
				assert.Nil(t, valueNode)
			}
		})
	}
}

func TestResolveAlias_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		node     *yaml.Node
		expected *yaml.Node
	}{
		{
			name:     "nil node",
			node:     nil,
			expected: nil,
		},
		{
			name: "non-alias node",
			node: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			expected: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
		},
		{
			name: "alias node",
			node: &yaml.Node{
				Kind: yaml.AliasNode,
				Alias: &yaml.Node{
					Value: "aliased-value",
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
				},
			},
			expected: &yaml.Node{
				Value: "aliased-value",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
		},
		{
			name: "nested alias node",
			node: &yaml.Node{
				Kind: yaml.AliasNode,
				Alias: &yaml.Node{
					Kind: yaml.AliasNode,
					Alias: &yaml.Node{
						Value: "deeply-aliased",
						Kind:  yaml.ScalarNode,
						Tag:   "!!str",
					},
				},
			},
			expected: &yaml.Node{
				Value: "deeply-aliased",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.ResolveAlias(tt.node)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Value, result.Value)
				assert.Equal(t, tt.expected.Kind, result.Kind)
				assert.Equal(t, tt.expected.Tag, result.Tag)
			}
		})
	}
}

func TestEqualNodes_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		nodeA    *yaml.Node
		nodeB    *yaml.Node
		expected bool
	}{
		{
			name:     "both nil",
			nodeA:    nil,
			nodeB:    nil,
			expected: true,
		},
		{
			name:     "one nil",
			nodeA:    nil,
			nodeB:    &yaml.Node{Value: "test", Kind: yaml.ScalarNode, Tag: "!!str"},
			expected: false,
		},
		{
			name: "equal scalar nodes",
			nodeA: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			nodeB: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			expected: true,
		},
		{
			name: "different values",
			nodeA: &yaml.Node{
				Value: "test1",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			nodeB: &yaml.Node{
				Value: "test2",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			expected: false,
		},
		{
			name: "different kinds",
			nodeA: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			nodeB: &yaml.Node{
				Value: "test",
				Kind:  yaml.MappingNode,
				Tag:   "!!str",
			},
			expected: false,
		},
		{
			name: "different tags",
			nodeA: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			nodeB: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!int",
			},
			expected: false,
		},
		{
			name: "equal complex nodes",
			nodeA: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			nodeB: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expected: true,
		},
		{
			name: "different content length",
			nodeA: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			nodeB: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.EqualNodes(tt.nodeA, tt.nodeB)
			assert.Equal(t, tt.expected, result)
		})
	}
}
