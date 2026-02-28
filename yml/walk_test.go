package yml_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestWalk_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		node          *yaml.Node
		expectedCalls int
	}{
		{
			name: "simple scalar node",
			node: &yaml.Node{
				Value: "test",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
			expectedCalls: 1,
		},
		{
			name: "document with scalar",
			node: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Value: "test",
						Kind:  yaml.ScalarNode,
						Tag:   "!!str",
					},
				},
			},
			expectedCalls: 2, // document + scalar
		},
		{
			name: "mapping node",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectedCalls: 5, // mapping + 4 scalars
		},
		{
			name: "sequence node",
			node: &yaml.Node{
				Kind: yaml.SequenceNode,
				Tag:  "!!seq",
				Content: []*yaml.Node{
					{Value: "item1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "item2", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "item3", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			expectedCalls: 4, // sequence + 3 scalars
		},
		{
			name: "complex nested structure",
			node: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "array", Kind: yaml.ScalarNode, Tag: "!!str"},
							{
								Kind: yaml.SequenceNode,
								Tag:  "!!seq",
								Content: []*yaml.Node{
									{Value: "item1", Kind: yaml.ScalarNode, Tag: "!!str"},
									{Value: "item2", Kind: yaml.ScalarNode, Tag: "!!str"},
								},
							},
							{Value: "scalar", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str"},
						},
					},
				},
			},
			expectedCalls: 8, // document + mapping + key + sequence + 2 items + key + value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			callCount := 0
			var visitedNodes []*yaml.Node
			var parentNodes []*yaml.Node
			var rootNodes []*yaml.Node

			visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
				callCount++
				visitedNodes = append(visitedNodes, node)
				parentNodes = append(parentNodes, parent)
				rootNodes = append(rootNodes, root)
				return nil
			}

			err := yml.Walk(ctx, tt.node, visitFunc)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCalls, callCount)
			assert.Len(t, visitedNodes, tt.expectedCalls)

			// Verify root is always the same
			for _, root := range rootNodes {
				assert.Equal(t, tt.node, root)
			}
		})
	}
}

func TestWalk_WithAliasNode_Success(t *testing.T) {
	t.Parallel()
	aliasTarget := &yaml.Node{
		Value: "aliased-value",
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
	}

	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Kind: yaml.AliasNode, Alias: aliasTarget},
		},
	}

	ctx := t.Context()
	callCount := 0
	var visitedNodes []*yaml.Node

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		visitedNodes = append(visitedNodes, node)
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err)
	assert.Equal(t, 6, callCount) // mapping + key1 + value1 + key2 + alias + aliased-value

	// Verify the alias target was visited
	found := false
	for _, visited := range visitedNodes {
		if visited == aliasTarget {
			found = true
			break
		}
	}
	assert.True(t, found, "alias target should be visited")
}

func TestWalk_WithNilNode_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	callCount := 0

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		return nil
	}

	err := yml.Walk(ctx, nil, visitFunc)

	require.NoError(t, err)
	assert.Equal(t, 0, callCount)
}

func TestWalk_WithVisitError_Error(t *testing.T) {
	t.Parallel()
	node := &yaml.Node{
		Value: "test",
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
	}

	ctx := t.Context()
	expectedError := errors.New("visit error")

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		return expectedError
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestWalk_WithTerminateError_Success(t *testing.T) {
	t.Parallel()
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str"},
		},
	}

	ctx := t.Context()
	callCount := 0

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		if callCount == 2 { // Terminate after visiting the mapping and first key
			return yml.ErrTerminate
		}
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err) // ErrTerminate should be handled and not returned as error
	assert.Equal(t, 2, callCount)
}

func TestWalk_DocumentNode_Success(t *testing.T) {
	t.Parallel()
	node := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			{
				Value: "second-doc",
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
			},
		},
	}

	ctx := t.Context()
	callCount := 0
	var nodeKinds []yaml.Kind

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		nodeKinds = append(nodeKinds, node.Kind)
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err)
	assert.Equal(t, 5, callCount) // document + mapping + key + value + second-doc

	expectedKinds := []yaml.Kind{
		yaml.DocumentNode,
		yaml.MappingNode,
		yaml.ScalarNode, // key
		yaml.ScalarNode, // value
		yaml.ScalarNode, // second-doc
	}
	assert.Equal(t, expectedKinds, nodeKinds)
}

func TestWalk_SequenceNode_Success(t *testing.T) {
	t.Parallel()
	node := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "nested-key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "nested-value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
			{Value: "simple-item", Kind: yaml.ScalarNode, Tag: "!!str"},
		},
	}

	ctx := t.Context()
	callCount := 0
	var nodeKinds []yaml.Kind

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		nodeKinds = append(nodeKinds, node.Kind)
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err)
	assert.Equal(t, 5, callCount) // sequence + mapping + nested-key + nested-value + simple-item

	expectedKinds := []yaml.Kind{
		yaml.SequenceNode,
		yaml.MappingNode,
		yaml.ScalarNode, // nested-key
		yaml.ScalarNode, // nested-value
		yaml.ScalarNode, // simple-item
	}
	assert.Equal(t, expectedKinds, nodeKinds)
}

func TestWalk_MappingNode_Success(t *testing.T) {
	t.Parallel()
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Value: "simple-key", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "simple-value", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "nested-key", Kind: yaml.ScalarNode, Tag: "!!str"},
			{
				Kind: yaml.SequenceNode,
				Tag:  "!!seq",
				Content: []*yaml.Node{
					{Value: "item1", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "item2", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
		},
	}

	ctx := t.Context()
	callCount := 0
	var nodeKinds []yaml.Kind

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		nodeKinds = append(nodeKinds, node.Kind)
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err)
	assert.Equal(t, 7, callCount) // mapping + simple-key + simple-value + nested-key + sequence + item1 + item2

	expectedKinds := []yaml.Kind{
		yaml.MappingNode,
		yaml.ScalarNode, // simple-key
		yaml.ScalarNode, // simple-value
		yaml.ScalarNode, // nested-key
		yaml.SequenceNode,
		yaml.ScalarNode, // item1
		yaml.ScalarNode, // item2
	}
	assert.Equal(t, expectedKinds, nodeKinds)
}

func TestWalk_AliasNode_Success(t *testing.T) {
	t.Parallel()
	aliasTarget := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Value: "aliased-key", Kind: yaml.ScalarNode, Tag: "!!str"},
			{Value: "aliased-value", Kind: yaml.ScalarNode, Tag: "!!str"},
		},
	}

	node := &yaml.Node{
		Kind:  yaml.AliasNode,
		Alias: aliasTarget,
	}

	ctx := t.Context()
	callCount := 0
	var nodeKinds []yaml.Kind

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		callCount++
		nodeKinds = append(nodeKinds, node.Kind)
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err)
	assert.Equal(t, 4, callCount) // alias + mapping + aliased-key + aliased-value

	expectedKinds := []yaml.Kind{
		yaml.AliasNode,
		yaml.MappingNode,
		yaml.ScalarNode, // aliased-key
		yaml.ScalarNode, // aliased-value
	}
	assert.Equal(t, expectedKinds, nodeKinds)
}

func TestWalk_ParentTracking_Success(t *testing.T) {
	t.Parallel()
	node := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str"},
				},
			},
		},
	}

	ctx := t.Context()
	var parentTracker []struct {
		nodeKind   yaml.Kind
		parentKind *yaml.Kind
	}

	visitFunc := func(ctx context.Context, node, parent, root *yaml.Node) error {
		entry := struct {
			nodeKind   yaml.Kind
			parentKind *yaml.Kind
		}{
			nodeKind: node.Kind,
		}
		if parent != nil {
			entry.parentKind = &parent.Kind
		}
		parentTracker = append(parentTracker, entry)
		return nil
	}

	err := yml.Walk(ctx, node, visitFunc)

	require.NoError(t, err)
	require.Len(t, parentTracker, 4)

	// Document node has no parent
	assert.Equal(t, yaml.DocumentNode, parentTracker[0].nodeKind)
	assert.Nil(t, parentTracker[0].parentKind)

	// Mapping node has document as parent
	assert.Equal(t, yaml.MappingNode, parentTracker[1].nodeKind)
	require.NotNil(t, parentTracker[1].parentKind)
	assert.Equal(t, yaml.DocumentNode, *parentTracker[1].parentKind)

	// Key scalar has mapping as parent
	assert.Equal(t, yaml.ScalarNode, parentTracker[2].nodeKind)
	require.NotNil(t, parentTracker[2].parentKind)
	assert.Equal(t, yaml.MappingNode, *parentTracker[2].parentKind)

	// Value scalar has mapping as parent
	assert.Equal(t, yaml.ScalarNode, parentTracker[3].nodeKind)
	require.NotNil(t, parentTracker[3].parentKind)
	assert.Equal(t, yaml.MappingNode, *parentTracker[3].parentKind)
}
