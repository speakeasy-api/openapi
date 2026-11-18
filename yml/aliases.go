package yml

import "go.yaml.in/yaml/v3"

// ResolveOrphanedAliases replaces each alias in the tree whose anchored node is not the latest node defined with
// that anchor earlier in the document with a copy of the anchored node. An alias is written as `*name` and refers
// to the closest preceding `&name`, so without this an alias to a node that was replaced, removed or moved after it
// would point to a different value or to nothing. The copies don't keep anchors, so they don't redefine any, unless
// the copied node contains an alias to itself, which then points to the copy. The alias's comments are kept.
func ResolveOrphanedAliases(node *yaml.Node) {
	r := orphanResolver{anchors: map[string]*yaml.Node{}, expanding: map[*yaml.Node]*yaml.Node{}}
	r.walk(node)
}

type orphanResolver struct {
	// anchors is the latest node defined with each anchor so far in document order.
	anchors map[string]*yaml.Node
	// expanding maps each anchored node being copied to its copy, while the copy is walked.
	expanding map[*yaml.Node]*yaml.Node
}

func (r *orphanResolver) walk(node *yaml.Node) {
	if node == nil {
		return
	}
	if node.Anchor != "" {
		r.anchors[node.Anchor] = node
	}

	for i, child := range node.Content {
		if child == nil || child.Kind != yaml.AliasNode || child.Alias == nil || r.anchors[child.Alias.Anchor] == child.Alias {
			r.walk(child)
			continue
		}

		target := child.Alias
		// An alias inside the copy of its own target points to that copy, which is an ancestor.
		if c, ok := r.expanding[target]; ok {
			if c.Anchor == "" {
				c.Anchor = target.Anchor
				r.anchors[c.Anchor] = c
			}
			child.Alias = c
			child.Value = c.Anchor
			continue
		}

		c := copyWithoutAnchors(target)
		c.HeadComment, c.LineComment, c.FootComment = child.HeadComment, child.LineComment, child.FootComment
		node.Content[i] = c

		r.expanding[target] = c
		r.walk(c)
		delete(r.expanding, target)
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
