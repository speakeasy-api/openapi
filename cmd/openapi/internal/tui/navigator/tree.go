package navigator

import (
	"fmt"
	"strings"
)

// TreeNode represents a node in the navigation tree
type TreeNode interface {
	GetID() string
	GetDisplayName() string
	GetDisplayTitle() string // For top-level display (different from YAML-style name)
	GetDescription() string
	GetChildren() []TreeNode
	GetParent() TreeNode
	SetParent(TreeNode)
	IsExpandable() bool
	GetNodeType() NodeType
	GetDetails() map[string]interface{}
}

// NodeType represents the type of node in the tree
type NodeType string

const (
	NodeTypeRoot        NodeType = "root"
	NodeTypeInfo        NodeType = "info"
	NodeTypeServers     NodeType = "servers"
	NodeTypeServer      NodeType = "server"
	NodeTypePaths       NodeType = "paths"
	NodeTypePath        NodeType = "path"
	NodeTypeOperation   NodeType = "operation"
	NodeTypeComponents  NodeType = "components"
	NodeTypeSchemas     NodeType = "schemas"
	NodeTypeSchema      NodeType = "schema"
	NodeTypeParameters  NodeType = "parameters"
	NodeTypeParameter   NodeType = "parameter"
	NodeTypeResponses   NodeType = "responses"
	NodeTypeResponse    NodeType = "response"
	NodeTypeRequestBody NodeType = "requestBody"
	NodeTypeSecurity    NodeType = "security"
	NodeTypeTags        NodeType = "tags"
	NodeTypeTag         NodeType = "tag"
)

// BaseNode provides a basic implementation of TreeNode
type BaseNode struct {
	ID           string
	Content      string
	DisplayTitle string // Optional display title for top-level view
	Description  string
	Type         NodeType
	Children     []TreeNode
	Parent       TreeNode
	Details      map[string]interface{}
}

func (n *BaseNode) GetID() string {
	return n.ID
}

func (n *BaseNode) GetDisplayName() string {
	return n.Content
}

func (n *BaseNode) GetDisplayTitle() string {
	if n.DisplayTitle != "" {
		return n.DisplayTitle
	}
	return n.Content
}

func (n *BaseNode) GetDescription() string {
	return n.Description
}

func (n *BaseNode) GetChildren() []TreeNode {
	return n.Children
}

func (n *BaseNode) GetParent() TreeNode {
	return n.Parent
}

func (n *BaseNode) SetParent(parent TreeNode) {
	n.Parent = parent
}

func (n *BaseNode) IsExpandable() bool {
	return len(n.Children) > 0
}

func (n *BaseNode) GetNodeType() NodeType {
	return n.Type
}

func (n *BaseNode) GetDetails() map[string]interface{} {
	if n.Details == nil {
		n.Details = make(map[string]interface{})
	}
	return n.Details
}

// AddChild adds a child node and sets the parent relationship
func (n *BaseNode) AddChild(child TreeNode) {
	child.SetParent(n)
	n.Children = append(n.Children, child)
}

// Tree represents the navigation tree structure
type Tree struct {
	Root    TreeNode
	Current TreeNode
	Path    []TreeNode // Breadcrumb path from root to current
}

// NewTree creates a new navigation tree
func NewTree(root TreeNode) *Tree {
	return &Tree{
		Root:    root,
		Current: root,
		Path:    []TreeNode{root},
	}
}

// NavigateToChild moves to a child node
func (t *Tree) NavigateToChild(childIndex int) bool {
	children := t.Current.GetChildren()
	if childIndex < 0 || childIndex >= len(children) {
		return false
	}

	child := children[childIndex]
	t.Current = child
	t.Path = append(t.Path, child)
	return true
}

// NavigateToParent moves to the parent node
func (t *Tree) NavigateToParent() bool {
	if len(t.Path) <= 1 {
		return false // Already at root
	}

	// Remove current from path
	t.Path = t.Path[:len(t.Path)-1]
	t.Current = t.Path[len(t.Path)-1]
	return true
}

// GetBreadcrumb returns the breadcrumb path as a string
func (t *Tree) GetBreadcrumb() string {
	if len(t.Path) == 0 {
		return ""
	}

	var parts []string
	for _, node := range t.Path {
		parts = append(parts, node.GetDisplayName())
	}
	return strings.Join(parts, " > ")
}

// GetCurrentLevel returns the current level nodes for display
func (t *Tree) GetCurrentLevel() []TreeNode {
	return t.Current.GetChildren()
}

// GetCurrentLevelWithPreview returns current level with one level deeper preview
func (t *Tree) GetCurrentLevelWithPreview() []DisplayItem {
	children := t.Current.GetChildren()
	var items []DisplayItem

	for i, child := range children {
		item := DisplayItem{
			Index:       i,
			Node:        child,
			HasChildren: child.IsExpandable(),
		}

		// Add preview of children if expandable
		if child.IsExpandable() {
			grandChildren := child.GetChildren()
			for j, grandChild := range grandChildren {
				if j >= 3 { // Limit preview to first 3 items
					item.Preview = append(item.Preview, PreviewItem{
						Name: "...",
						Type: "more",
					})
					break
				}
				item.Preview = append(item.Preview, PreviewItem{
					Name: grandChild.GetDisplayName(),
					Type: string(grandChild.GetNodeType()),
				})
			}
		}

		items = append(items, item)
	}

	return items
}

// DisplayItem represents an item for display in the TUI
type DisplayItem struct {
	Index       int
	Node        TreeNode
	HasChildren bool
	Preview     []PreviewItem
}

// PreviewItem represents a preview of child items
type PreviewItem struct {
	Name string
	Type string
}

// GetDisplayText returns the formatted display text for the item
func (d *DisplayItem) GetDisplayText() string {
	node := d.Node
	icon := getNodeIcon(node.GetNodeType())

	// Format with consistent spacing and tab stops
	title := node.GetDisplayName()
	description := node.GetDescription()

	if description != "" {
		// Use tab character to align content after title
		text := fmt.Sprintf("%s %s\t%s", icon, title, description)
		return text
	}

	return fmt.Sprintf("%s %s", icon, title)
}

// getNodeIcon returns an appropriate icon for the node type
func getNodeIcon(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeRoot:
		return "📄"
	case NodeTypeInfo:
		return "📋"
	case NodeTypeServers:
		return "🌐"
	case NodeTypeServer:
		return "🖥️"
	case NodeTypePaths:
		return "🛣️"
	case NodeTypePath:
		return "📁"
	case NodeTypeOperation:
		return "⚡"
	case NodeTypeComponents:
		return "🧩"
	case NodeTypeSchemas:
		return "📊"
	case NodeTypeSchema:
		return "📋"
	case NodeTypeParameters:
		return "⚙️"
	case NodeTypeParameter:
		return "🔧"
	case NodeTypeResponses:
		return "📤"
	case NodeTypeResponse:
		return "📨"
	case NodeTypeRequestBody:
		return "📥"
	case NodeTypeSecurity:
		return "🔒"
	case NodeTypeTags:
		return "🏷️"
	case NodeTypeTag:
		return "🏷️"
	default:
		return "📄"
	}
}
