package yml

import (
	"context"
	"strconv"

	"gopkg.in/yaml.v3"
)

func CreateOrUpdateKeyNode(ctx context.Context, key string, keyNode *yaml.Node) *yaml.Node {
	if keyNode != nil {
		resolvedKeyNode := ResolveAlias(keyNode)
		if resolvedKeyNode == nil {
			resolvedKeyNode = keyNode
		}

		resolvedKeyNode.Value = key
		return keyNode
	}

	cfg := GetConfigFromContext(ctx)

	return &yaml.Node{
		Value: key,
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Style: cfg.KeyStringStyle,
	}
}

func CreateOrUpdateScalarNode(ctx context.Context, value any, valueNode *yaml.Node) *yaml.Node {
	var convNode yaml.Node
	if err := convNode.Encode(value); err != nil {
		return nil
	}

	resolvedValueNode := ResolveAlias(valueNode)

	if resolvedValueNode != nil {
		resolvedValueNode.Value = convNode.Value
		return valueNode
	}

	cfg := GetConfigFromContext(ctx)

	if convNode.Kind == yaml.ScalarNode && convNode.Tag == "!!str" {
		convNode.Style = cfg.ValueStringStyle
	}

	return &convNode
}

func CreateOrUpdateMapNodeElement(ctx context.Context, key string, keyNode, valueNode, mapNode *yaml.Node) *yaml.Node {
	resolvedMapNode := ResolveAlias(mapNode)

	if resolvedMapNode != nil {
		for i := 0; i < len(resolvedMapNode.Content); i += 2 {
			keyNode := resolvedMapNode.Content[i]
			// Check direct match first
			if keyNode.Value == key {
				resolvedMapNode.Content[i+1] = valueNode
				return mapNode
			}
			// Check alias resolution match for alias keys like *keyAlias
			if resolvedKeyNode := ResolveAlias(keyNode); resolvedKeyNode != nil && resolvedKeyNode.Value == key {
				resolvedMapNode.Content[i+1] = valueNode
				return mapNode
			}
		}

		resolvedMapNode.Content = append(resolvedMapNode.Content, CreateOrUpdateKeyNode(ctx, key, keyNode))
		resolvedMapNode.Content = append(resolvedMapNode.Content, valueNode)

		return mapNode
	}

	return CreateMapNode(ctx, []*yaml.Node{
		CreateOrUpdateKeyNode(ctx, key, keyNode),
		valueNode,
	})
}

func CreateStringNode(value string) *yaml.Node {
	return &yaml.Node{
		Value: value,
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
	}
}

func CreateIntNode(value int64) *yaml.Node {
	return &yaml.Node{
		Value: strconv.FormatInt(value, 10),
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
	}
}

func CreateFloatNode(value float64) *yaml.Node {
	return &yaml.Node{
		Value: strconv.FormatFloat(value, 'f', -1, 64),
		Kind:  yaml.ScalarNode,
		Tag:   "!!float",
	}
}

func CreateBoolNode(value bool) *yaml.Node {
	return &yaml.Node{
		Value: strconv.FormatBool(value),
		Kind:  yaml.ScalarNode,
		Tag:   "!!bool",
	}
}

func CreateMapNode(ctx context.Context, content []*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Content: content,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
	}
}

func DeleteMapNodeElement(ctx context.Context, key string, mapNode *yaml.Node) *yaml.Node {
	if mapNode == nil {
		return nil
	}

	resolvedMapNode := ResolveAlias(mapNode)
	if resolvedMapNode == nil {
		return nil
	}

	for i := 0; i < len(resolvedMapNode.Content); i += 2 {
		if resolvedMapNode.Content[i].Value == key {
			mapNode.Content = append(resolvedMapNode.Content[:i], resolvedMapNode.Content[i+2:]...) //nolint:gocritic
			return mapNode
		}
	}

	return mapNode
}

func CreateOrUpdateSliceNode(ctx context.Context, elements []*yaml.Node, valueNode *yaml.Node) *yaml.Node {
	resolvedValueNode := ResolveAlias(valueNode)
	if resolvedValueNode != nil {
		resolvedValueNode.Content = elements
		return valueNode
	}

	return &yaml.Node{
		Content: elements,
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
	}
}

func GetMapElementNodes(ctx context.Context, mapNode *yaml.Node, key string) (*yaml.Node, *yaml.Node, bool) {
	resolvedMapNode := ResolveAlias(mapNode)
	if resolvedMapNode == nil {
		return nil, nil, false
	}

	if resolvedMapNode.Kind != yaml.MappingNode {
		return nil, nil, false
	}

	for i := 0; i < len(resolvedMapNode.Content); i += 2 {
		keyNode := resolvedMapNode.Content[i]
		// Check direct match first
		if keyNode.Value == key {
			return keyNode, resolvedMapNode.Content[i+1], true
		}
		// Check alias resolution match for alias keys like *keyAlias
		if resolvedKeyNode := ResolveAlias(keyNode); resolvedKeyNode != nil && resolvedKeyNode.Value == key {
			return keyNode, resolvedMapNode.Content[i+1], true
		}
	}

	return nil, nil, false
}

func ResolveAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.AliasNode:
		return ResolveAlias(node.Alias)
	default:
		return node
	}
}

// EqualNodes compares two yaml.Node instances for equality.
// It performs a deep comparison of the essential fields.
func EqualNodes(a, b *yaml.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Resolve aliases before comparison
	resolvedA := ResolveAlias(a)
	resolvedB := ResolveAlias(b)

	if resolvedA == nil && resolvedB == nil {
		return true
	}
	if resolvedA == nil || resolvedB == nil {
		return false
	}

	// Compare essential fields
	if resolvedA.Kind != resolvedB.Kind {
		return false
	}
	if resolvedA.Tag != resolvedB.Tag {
		return false
	}
	if resolvedA.Value != resolvedB.Value {
		return false
	}

	// Compare content for complex nodes
	if len(resolvedA.Content) != len(resolvedB.Content) {
		return false
	}
	for i, contentA := range resolvedA.Content {
		if !EqualNodes(contentA, resolvedB.Content[i]) {
			return false
		}
	}

	return true
}
