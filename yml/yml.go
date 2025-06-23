package yml

import (
	"context"

	"gopkg.in/yaml.v3"
)

func CreateOrUpdateKeyNode(ctx context.Context, key string, keyNode *yaml.Node) *yaml.Node {
	if keyNode != nil {
		resolvedKeyNode := ResolveAlias(keyNode)

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
			if resolvedMapNode.Content[i].Value == key {
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

func CreateMapNode(ctx context.Context, content []*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Content: content,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
	}
}

func DeleteMapNodeElement(ctx context.Context, key string, mapNode *yaml.Node) *yaml.Node {
	resolvedMapNode := ResolveAlias(mapNode)
	if resolvedMapNode == nil {
		return nil
	}

	for i := 0; i < len(resolvedMapNode.Content); i += 2 {
		if resolvedMapNode.Content[i].Value == key {
			mapNode.Content = append(resolvedMapNode.Content[:i], resolvedMapNode.Content[i+2:]...)
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
		if resolvedMapNode.Content[i].Value == key {
			return resolvedMapNode.Content[i], resolvedMapNode.Content[i+1], true
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
