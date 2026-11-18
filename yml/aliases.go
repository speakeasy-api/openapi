package yml

import "go.yaml.in/yaml/v3"

// ResolveOrphanedAliases replaces each alias in the tree whose anchored node is not the latest node defined with
// that anchor earlier in the document with a copy of the anchored node. An alias is written as `*name` and refers
// to the closest preceding `&name`, so without this an alias to a node that was replaced, removed or moved after it
// would point to a different value or to nothing. The copies don't keep anchors, so they don't redefine any.
func ResolveOrphanedAliases(node *yaml.Node) {
	resolveOrphanedAliases(node, map[string]*yaml.Node{})
}

func resolveOrphanedAliases(node *yaml.Node, anchors map[string]*yaml.Node) {
	if node == nil {
		return
	}
	if node.Anchor != "" {
		anchors[node.Anchor] = node
	}

	for i, child := range node.Content {
		if child != nil && child.Kind == yaml.AliasNode && child.Alias != nil && anchors[child.Alias.Anchor] != child.Alias {
			child = copyWithoutAnchors(child.Alias)
			node.Content[i] = child
		}
		resolveOrphanedAliases(child, anchors)
	}
}

// copyWithoutAnchors deep copies a node, removing anchors. Aliases inside it are kept and checked as the copy is
// walked.
func copyWithoutAnchors(node *yaml.Node) *yaml.Node {
	c := *node
	c.Anchor = ""
	if node.Content != nil {
		c.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			if child != nil && child.Kind != yaml.AliasNode {
				child = copyWithoutAnchors(child)
			}
			c.Content[i] = child
		}
	}
	return &c
}
