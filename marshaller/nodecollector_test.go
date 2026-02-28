package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// Test models to verify CollectLeafNodes behavior

// SimpleModel has only primitive leaf fields - all should be collected
type SimpleModel struct {
	marshaller.CoreModel

	StringField marshaller.Node[*string] `key:"stringField"`
	IntField    marshaller.Node[*int]    `key:"intField"`
	BoolField   marshaller.Node[*bool]   `key:"boolField"`
}

// ModelWithSlice has a slice of primitives - all items should be collected
type ModelWithSlice struct {
	marshaller.CoreModel

	Items marshaller.Node[[]string] `key:"items"`
}

// ModelWithNodeSlice has a slice of Node[string] - all items should be collected
type ModelWithNodeSlice struct {
	marshaller.CoreModel

	Tags marshaller.Node[[]marshaller.Node[string]] `key:"tags"`
}

// NestedCoreModel represents a model that would be walked separately
type NestedCoreModel struct {
	marshaller.CoreModel

	Name marshaller.Node[*string] `key:"name"`
}

func (n *NestedCoreModel) GetRootNode() *yaml.Node {
	return n.RootNode
}

// ModelWithNestedCore has a nested core model - the nested model's nodes should NOT be collected
type ModelWithNestedCore struct {
	marshaller.CoreModel

	Title  marshaller.Node[*string]          `key:"title"`
	Nested marshaller.Node[*NestedCoreModel] `key:"nested"`
}

// ModelWithSliceOfCoreModels has a slice of core models - those nodes should NOT be collected
type ModelWithSliceOfCoreModels struct {
	marshaller.CoreModel

	Description marshaller.Node[*string]            `key:"description"`
	Children    marshaller.Node[[]*NestedCoreModel] `key:"children"`
}

func TestCollectLeafNodes_NilInput_Success(t *testing.T) {
	t.Parallel()

	nodes := marshaller.CollectLeafNodes(nil)
	assert.Nil(t, nodes, "should return nil for nil input")
}

func TestCollectLeafNodes_SimpleModel_CollectsAllNodes(t *testing.T) {
	t.Parallel()

	// Create YAML nodes
	stringKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "stringField"}
	stringValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "hello"}
	intKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "intField"}
	intValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "42"}
	boolKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "boolField"}
	boolValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "true"}

	str := "hello"
	intVal := 42
	boolVal := true

	model := &SimpleModel{
		StringField: marshaller.Node[*string]{
			KeyNode:   stringKeyNode,
			ValueNode: stringValueNode,
			Value:     &str,
			Present:   true,
		},
		IntField: marshaller.Node[*int]{
			KeyNode:   intKeyNode,
			ValueNode: intValueNode,
			Value:     &intVal,
			Present:   true,
		},
		BoolField: marshaller.Node[*bool]{
			KeyNode:   boolKeyNode,
			ValueNode: boolValueNode,
			Value:     &boolVal,
			Present:   true,
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	// Should have 6 nodes (KeyNode + ValueNode for each of 3 fields)
	require.Len(t, nodes, 6, "should collect all key and value nodes")

	// Verify all nodes are collected
	nodeSet := make(map[*yaml.Node]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}

	assert.True(t, nodeSet[stringKeyNode], "should include stringField key node")
	assert.True(t, nodeSet[stringValueNode], "should include stringField value node")
	assert.True(t, nodeSet[intKeyNode], "should include intField key node")
	assert.True(t, nodeSet[intValueNode], "should include intField value node")
	assert.True(t, nodeSet[boolKeyNode], "should include boolField key node")
	assert.True(t, nodeSet[boolValueNode], "should include boolField value node")
}

func TestCollectLeafNodes_NotPresent_SkipsField(t *testing.T) {
	t.Parallel()

	stringKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "stringField"}
	stringValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "hello"}
	intKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "intField"}
	intValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "42"}

	str := "hello"

	model := &SimpleModel{
		StringField: marshaller.Node[*string]{
			KeyNode:   stringKeyNode,
			ValueNode: stringValueNode,
			Value:     &str,
			Present:   true,
		},
		IntField: marshaller.Node[*int]{
			KeyNode:   intKeyNode,
			ValueNode: intValueNode,
			Value:     nil,
			Present:   false, // Not present - should be skipped
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	// Should have 2 nodes (only StringField)
	require.Len(t, nodes, 2, "should only collect present fields")

	nodeSet := make(map[*yaml.Node]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}

	assert.True(t, nodeSet[stringKeyNode], "should include present field key node")
	assert.True(t, nodeSet[stringValueNode], "should include present field value node")
	assert.False(t, nodeSet[intKeyNode], "should not include non-present field key node")
	assert.False(t, nodeSet[intValueNode], "should not include non-present field value node")
}

func TestCollectLeafNodes_SliceOfPrimitives_CollectsChildren(t *testing.T) {
	t.Parallel()

	itemsKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "items"}
	item1Node := &yaml.Node{Kind: yaml.ScalarNode, Value: "item1"}
	item2Node := &yaml.Node{Kind: yaml.ScalarNode, Value: "item2"}
	item3Node := &yaml.Node{Kind: yaml.ScalarNode, Value: "item3"}
	itemsValueNode := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{item1Node, item2Node, item3Node},
	}

	model := &ModelWithSlice{
		Items: marshaller.Node[[]string]{
			KeyNode:   itemsKeyNode,
			ValueNode: itemsValueNode,
			Value:     []string{"item1", "item2", "item3"},
			Present:   true,
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	// Should have: keyNode + valueNode + 3 child nodes = 5
	require.Len(t, nodes, 5, "should collect key, value, and child nodes")

	nodeSet := make(map[*yaml.Node]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}

	assert.True(t, nodeSet[itemsKeyNode], "should include items key node")
	assert.True(t, nodeSet[itemsValueNode], "should include items value node")
	assert.True(t, nodeSet[item1Node], "should include item1 node")
	assert.True(t, nodeSet[item2Node], "should include item2 node")
	assert.True(t, nodeSet[item3Node], "should include item3 node")
}

func TestCollectLeafNodes_NestedCoreModel_DoesNotCollectNestedNodes(t *testing.T) {
	t.Parallel()

	// Parent's leaf field
	titleKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "title"}
	titleValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "My Title"}

	// Nested model's field - should NOT be collected
	nestedNameKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "name"}
	nestedNameValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "Nested Name"}
	nestedKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "nested"}
	nestedValueNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nestedNameKeyNode,
			nestedNameValueNode,
		},
	}

	nestedName := "Nested Name"
	title := "My Title"

	nestedCore := &NestedCoreModel{
		Name: marshaller.Node[*string]{
			KeyNode:   nestedNameKeyNode,
			ValueNode: nestedNameValueNode,
			Value:     &nestedName,
			Present:   true,
		},
	}
	nestedCore.RootNode = nestedValueNode

	model := &ModelWithNestedCore{
		Title: marshaller.Node[*string]{
			KeyNode:   titleKeyNode,
			ValueNode: titleValueNode,
			Value:     &title,
			Present:   true,
		},
		Nested: marshaller.Node[*NestedCoreModel]{
			KeyNode:   nestedKeyNode,
			ValueNode: nestedValueNode,
			Value:     nestedCore,
			Present:   true,
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	nodeSet := make(map[*yaml.Node]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}

	// Should collect Title field nodes (leaf)
	assert.True(t, nodeSet[titleKeyNode], "should include title key node")
	assert.True(t, nodeSet[titleValueNode], "should include title value node")

	// Should NOT collect nested model's internal field nodes
	// (the nested model itself will be walked separately)
	assert.False(t, nodeSet[nestedNameKeyNode], "should NOT include nested model's internal key node")
	assert.False(t, nodeSet[nestedNameValueNode], "should NOT include nested model's internal value node")
}

func TestCollectLeafNodes_SliceOfCoreModels_DoesNotCollectNestedNodes(t *testing.T) {
	t.Parallel()

	// Parent's leaf field
	descKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "description"}
	descValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "A description"}

	// Child 1 - should NOT be collected
	child1NameKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "name"}
	child1NameValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "Child 1"}
	child1RootNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			child1NameKeyNode,
			child1NameValueNode,
		},
	}

	// Child 2 - should NOT be collected
	child2NameKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "name"}
	child2NameValueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "Child 2"}
	child2RootNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			child2NameKeyNode,
			child2NameValueNode,
		},
	}

	childrenKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "children"}
	childrenValueNode := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{child1RootNode, child2RootNode},
	}

	desc := "A description"
	child1Name := "Child 1"
	child2Name := "Child 2"

	child1 := &NestedCoreModel{
		Name: marshaller.Node[*string]{
			KeyNode:   child1NameKeyNode,
			ValueNode: child1NameValueNode,
			Value:     &child1Name,
			Present:   true,
		},
	}
	child1.RootNode = child1RootNode

	child2 := &NestedCoreModel{
		Name: marshaller.Node[*string]{
			KeyNode:   child2NameKeyNode,
			ValueNode: child2NameValueNode,
			Value:     &child2Name,
			Present:   true,
		},
	}
	child2.RootNode = child2RootNode

	model := &ModelWithSliceOfCoreModels{
		Description: marshaller.Node[*string]{
			KeyNode:   descKeyNode,
			ValueNode: descValueNode,
			Value:     &desc,
			Present:   true,
		},
		Children: marshaller.Node[[]*NestedCoreModel]{
			KeyNode:   childrenKeyNode,
			ValueNode: childrenValueNode,
			Value:     []*NestedCoreModel{child1, child2},
			Present:   true,
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	nodeSet := make(map[*yaml.Node]bool)
	for _, n := range nodes {
		nodeSet[n] = true
	}

	// Should collect Description field nodes (leaf)
	assert.True(t, nodeSet[descKeyNode], "should include description key node")
	assert.True(t, nodeSet[descValueNode], "should include description value node")

	// Should NOT collect Children array's child model nodes
	// (they will be walked separately)
	assert.False(t, nodeSet[child1NameKeyNode], "should NOT include child1's name key node")
	assert.False(t, nodeSet[child1NameValueNode], "should NOT include child1's name value node")
	assert.False(t, nodeSet[child2NameKeyNode], "should NOT include child2's name key node")
	assert.False(t, nodeSet[child2NameValueNode], "should NOT include child2's name value node")
}

func TestCollectLeafNodes_NilKeyNode_SkipsKeyNode(t *testing.T) {
	t.Parallel()

	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "hello"}
	str := "hello"

	model := &SimpleModel{
		StringField: marshaller.Node[*string]{
			KeyNode:   nil, // No key node
			ValueNode: valueNode,
			Value:     &str,
			Present:   true,
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	require.Len(t, nodes, 1, "should only collect value node")
	assert.Equal(t, valueNode, nodes[0], "should collect value node")
}

func TestCollectLeafNodes_NilValueNode_SkipsValueNode(t *testing.T) {
	t.Parallel()

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "stringField"}
	str := "hello"

	model := &SimpleModel{
		StringField: marshaller.Node[*string]{
			KeyNode:   keyNode,
			ValueNode: nil, // No value node
			Value:     &str,
			Present:   true,
		},
	}

	nodes := marshaller.CollectLeafNodes(model)

	require.Len(t, nodes, 1, "should only collect key node")
	assert.Equal(t, keyNode, nodes[0], "should collect key node")
}

func TestCollectLeafNodes_EmptyModel_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	model := &SimpleModel{}

	nodes := marshaller.CollectLeafNodes(model)

	assert.Empty(t, nodes, "should return empty for model with no present fields")
}
