package yml

import (
	"context"

	"gopkg.in/yaml.v3"
)

func CreateOrUpdateKeyNode(ctx context.Context, key string, keyNode *yaml.Node) *yaml.Node {
	if keyNode != nil {
		keyNode.Value = key
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

	if valueNode != nil {
		valueNode.Value = convNode.Value
		return valueNode
	}

	cfg := GetConfigFromContext(ctx)

	if convNode.Kind == yaml.ScalarNode && convNode.Tag == "!!str" {
		convNode.Style = cfg.ValueStringStyle
	}

	return &convNode
}

func CreateOrUpdateMapNodeElement(ctx context.Context, key string, keyNode, valueNode, mapNode *yaml.Node) *yaml.Node {
	if mapNode != nil {
		for i := 0; i < len(mapNode.Content); i += 2 {
			if mapNode.Content[i].Value == key {
				mapNode.Content[i+1] = valueNode
				return mapNode
			}
		}

		mapNode.Content = append(mapNode.Content, CreateOrUpdateKeyNode(ctx, key, keyNode))
		mapNode.Content = append(mapNode.Content, valueNode)

		return mapNode
	}

	return &yaml.Node{
		Content: []*yaml.Node{
			CreateOrUpdateKeyNode(ctx, key, keyNode),
			valueNode,
		},
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
}

func DeleteMapNodeElement(ctx context.Context, key string, mapNode *yaml.Node) *yaml.Node {
	if mapNode == nil {
		return nil
	}

	for i := 0; i < len(mapNode.Content); i += 2 {
		if mapNode.Content[i].Value == key {
			mapNode.Content = append(mapNode.Content[:i], mapNode.Content[i+2:]...)
			return mapNode
		}
	}

	return mapNode
}

func CreateOrUpdateSliceNode(ctx context.Context, elements []*yaml.Node, valueNode *yaml.Node) *yaml.Node {
	if valueNode != nil {
		valueNode.Content = elements
		return valueNode
	}

	return &yaml.Node{
		Content: elements,
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
	}
}

func GetMapElementNodes(ctx context.Context, mapNode *yaml.Node, key string) (*yaml.Node, *yaml.Node, bool) {
	if mapNode == nil {
		return nil, nil, false
	}

	if mapNode.Kind != yaml.MappingNode {
		return nil, nil, false
	}

	for i := 0; i < len(mapNode.Content); i += 2 {
		if mapNode.Content[i].Value == key {
			return mapNode.Content[i], mapNode.Content[i+1], true
		}
	}

	return nil, nil, false
}
