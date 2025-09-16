package navigator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseNode_Success(t *testing.T) {
	// Create a test tree structure
	root := &BaseNode{
		ID:          "root",
		Content:        "Root Node",
		Description: "Root description",
		Type:        NodeTypeRoot,
		Children:    []TreeNode{},
		Details:     make(map[string]interface{}),
	}

	child1 := &BaseNode{
		ID:          "child1",
		Content:        "Child 1",
		Description: "First child",
		Type:        NodeTypeInfo,
		Children:    []TreeNode{},
		Details:     make(map[string]interface{}),
	}

	child2 := &BaseNode{
		ID:          "child2",
		Content:        "Child 2",
		Description: "Second child",
		Type:        NodeTypePaths,
		Children:    []TreeNode{},
		Details:     make(map[string]interface{}),
	}

	// Add children to root
	root.AddChild(child1)
	root.AddChild(child2)

	// Test basic node properties
	assert.Equal(t, "root", root.GetID())
	assert.Equal(t, "Root Node", root.GetDisplayName())
	assert.Equal(t, "Root description", root.GetDescription())
	assert.Equal(t, NodeTypeRoot, root.GetNodeType())
	assert.True(t, root.IsExpandable())
	assert.Len(t, root.GetChildren(), 2)

	// Test parent relationships
	assert.Equal(t, root, child1.GetParent())
	assert.Equal(t, root, child2.GetParent())

	// Test child nodes
	children := root.GetChildren()
	assert.Equal(t, child1, children[0])
	assert.Equal(t, child2, children[1])
}

func TestTree_Navigation_Success(t *testing.T) {
	// Create a test tree structure
	root := &BaseNode{
		ID:       "root",
		Content:     "Root",
		Type:     NodeTypeRoot,
		Children: []TreeNode{},
	}

	child1 := &BaseNode{
		ID:       "child1",
		Content:     "Child 1",
		Type:     NodeTypeInfo,
		Children: []TreeNode{},
	}

	grandchild := &BaseNode{
		ID:       "grandchild",
		Content:     "Grandchild",
		Type:     NodeTypeOperation,
		Children: []TreeNode{},
	}

	// Build the tree
	child1.AddChild(grandchild)
	root.AddChild(child1)

	// Create tree navigator
	tree := NewTree(root)

	// Test initial state
	assert.Equal(t, root, tree.Current)
	assert.Len(t, tree.Path, 1)
	assert.Equal(t, "Root", tree.GetBreadcrumb())

	// Test navigation to child
	success := tree.NavigateToChild(0)
	assert.True(t, success)
	assert.Equal(t, child1, tree.Current)
	assert.Len(t, tree.Path, 2)
	assert.Equal(t, "Root > Child 1", tree.GetBreadcrumb())

	// Test navigation to grandchild
	success = tree.NavigateToChild(0)
	assert.True(t, success)
	assert.Equal(t, grandchild, tree.Current)
	assert.Len(t, tree.Path, 3)
	assert.Equal(t, "Root > Child 1 > Grandchild", tree.GetBreadcrumb())

	// Test navigation back to parent
	success = tree.NavigateToParent()
	assert.True(t, success)
	assert.Equal(t, child1, tree.Current)
	assert.Len(t, tree.Path, 2)
	assert.Equal(t, "Root > Child 1", tree.GetBreadcrumb())

	// Test navigation back to root
	success = tree.NavigateToParent()
	assert.True(t, success)
	assert.Equal(t, root, tree.Current)
	assert.Len(t, tree.Path, 1)
	assert.Equal(t, "Root", tree.GetBreadcrumb())

	// Test can't navigate past root
	success = tree.NavigateToParent()
	assert.False(t, success)
	assert.Equal(t, root, tree.Current)
	assert.Len(t, tree.Path, 1)
}

func TestTree_NavigationErrors_Error(t *testing.T) {
	root := &BaseNode{
		ID:       "root",
		Content:     "Root",
		Type:     NodeTypeRoot,
		Children: []TreeNode{},
	}

	tree := NewTree(root)

	// Test navigation to invalid child index
	success := tree.NavigateToChild(-1)
	assert.False(t, success)

	success = tree.NavigateToChild(0) // No children
	assert.False(t, success)

	success = tree.NavigateToChild(10) // Out of bounds
	assert.False(t, success)

	// Current should remain unchanged
	assert.Equal(t, root, tree.Current)
	assert.Len(t, tree.Path, 1)
}

func TestDisplayItem_Success(t *testing.T) {
	node := &BaseNode{
		ID:          "test",
		Content:        "Test Node",
		Description: "Test description",
		Type:        NodeTypeOperation,
		Children:    []TreeNode{},
	}

	item := DisplayItem{
		Index:       0,
		Node:        node,
		HasChildren: false,
		Preview:     []PreviewItem{},
	}

	displayText := item.GetDisplayText()
	assert.Contains(t, displayText, "⚡") // Operation icon
	assert.Contains(t, displayText, "Test Node")
	assert.Contains(t, displayText, "Test description")
}

func TestGetNodeIcon_Success(t *testing.T) {
	tests := []struct {
		nodeType     NodeType
		expectedIcon string
	}{
		{NodeTypeRoot, "📄"},
		{NodeTypeInfo, "📋"},
		{NodeTypeServers, "🌐"},
		{NodeTypeServer, "🖥️"},
		{NodeTypePaths, "🛣️"},
		{NodeTypePath, "📁"},
		{NodeTypeOperation, "⚡"},
		{NodeTypeComponents, "🧩"},
		{NodeTypeSchemas, "📊"},
		{NodeTypeSchema, "📋"},
		{NodeTypeParameters, "⚙️"},
		{NodeTypeParameter, "🔧"},
		{NodeTypeResponses, "📤"},
		{NodeTypeResponse, "📨"},
		{NodeTypeRequestBody, "📥"},
		{NodeTypeSecurity, "🔒"},
		{NodeTypeTags, "🏷️"},
		{NodeTypeTag, "🏷️"},
	}

	for _, tt := range tests {
		t.Run(string(tt.nodeType), func(t *testing.T) {
			icon := getNodeIcon(tt.nodeType)
			assert.Equal(t, tt.expectedIcon, icon)
		})
	}
}

func TestTree_GetCurrentLevelWithPreview_Success(t *testing.T) {
	// Create a tree with preview items
	root := &BaseNode{
		ID:       "root",
		Content:     "Root",
		Type:     NodeTypeRoot,
		Children: []TreeNode{},
	}

	child := &BaseNode{
		ID:       "child",
		Content:     "Child",
		Type:     NodeTypePaths,
		Children: []TreeNode{},
	}

	grandchild1 := &BaseNode{
		ID:   "gc1",
		Content: "Grandchild 1",
		Type: NodeTypeOperation,
	}

	grandchild2 := &BaseNode{
		ID:   "gc2",
		Content: "Grandchild 2",
		Type: NodeTypeOperation,
	}

	child.AddChild(grandchild1)
	child.AddChild(grandchild2)
	root.AddChild(child)

	tree := NewTree(root)
	items := tree.GetCurrentLevelWithPreview()

	require.Len(t, items, 1)
	assert.Equal(t, child, items[0].Node)
	assert.True(t, items[0].HasChildren)
	assert.Len(t, items[0].Preview, 2)
	assert.Equal(t, "Grandchild 1", items[0].Preview[0].Name)
	assert.Equal(t, "operation", items[0].Preview[0].Type)
}
