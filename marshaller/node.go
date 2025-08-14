package marshaller

import (
	"context"
	"reflect"

	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type NodeMutator interface {
	Unmarshal(ctx context.Context, parentName string, keyNode, valueNode *yaml.Node) ([]error, error)
	SetPresent(present bool)
	SyncValue(ctx context.Context, key string, value any) (*yaml.Node, *yaml.Node, error)
}

type NodeAccessor interface {
	GetValue() any
	GetValueType() reflect.Type
}

type Node[V any] struct {
	Key       string
	KeyNode   *yaml.Node
	Value     V
	ValueNode *yaml.Node
	Present   bool
}

var (
	_ NodeAccessor = Node[any]{}
	_ NodeMutator  = (*Node[any])(nil)
)

func (n *Node[V]) Unmarshal(ctx context.Context, parentName string, keyNode, valueNode *yaml.Node) ([]error, error) {
	resolvedKeyNode := yml.ResolveAlias(keyNode)
	if resolvedKeyNode != nil {
		n.Key = resolvedKeyNode.Value
		n.KeyNode = keyNode
	}
	n.ValueNode = valueNode

	validationErrs, err := UnmarshalCore(ctx, parentName, n.ValueNode, &n.Value)

	n.SetPresent(err == nil && len(validationErrs) == 0)

	return validationErrs, err
}

func (n Node[V]) GetValue() any {
	return n.Value
}

func (n Node[V]) GetValueType() reflect.Type {
	return reflect.TypeOf(n.Value)
}

func (n *Node[V]) SyncValue(ctx context.Context, key string, value any) (*yaml.Node, *yaml.Node, error) {
	n.Key = key
	n.KeyNode = yml.CreateOrUpdateKeyNode(ctx, key, n.KeyNode)

	valueNode, err := SyncValue(ctx, value, &n.Value, n.ValueNode, false)
	if err != nil {
		return nil, nil, err
	}

	n.ValueNode = valueNode

	return n.KeyNode, n.ValueNode, nil
}

func (n *Node[V]) SetPresent(present bool) {
	n.Present = present
}

func (n Node[V]) GetKeyNode() *yaml.Node {
	return n.KeyNode
}

func (n Node[V]) GetKeyNodeOrRoot(rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.KeyNode == nil {
		return rootNode
	}
	return n.KeyNode
}

func (n Node[V]) GetKeyNodeOrRootLine(rootNode *yaml.Node) int {
	keyNode := n.GetKeyNodeOrRoot(rootNode)
	if keyNode == nil {
		return -1
	}
	return keyNode.Line
}

func (n Node[V]) GetValueNode() *yaml.Node {
	return n.ValueNode
}

func (n Node[V]) GetValueNodeOrRoot(rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}
	return n.ValueNode
}

func (n Node[V]) GetValueNodeOrRootLine(rootNode *yaml.Node) int {
	valueNode := n.GetValueNodeOrRoot(rootNode)
	if valueNode == nil {
		return -1
	}
	return valueNode.Line
}

// Will return the value node for the slice index, or the slice root node or the provided root node if the node is not present
func (n Node[V]) GetSliceValueNodeOrRoot(idx int, rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}

	resolvedNode := yml.ResolveAlias(n.ValueNode)
	if resolvedNode == nil {
		return rootNode
	}

	if idx < 0 || idx >= len(resolvedNode.Content) {
		return n.ValueNode
	}

	return resolvedNode.Content[idx]
}

// Will return the key node for the map key, or the map root node or the provided root node if the node is not present
func (n Node[V]) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}

	resolvedNode := yml.ResolveAlias(n.ValueNode)
	if resolvedNode == nil {
		return rootNode
	}

	for i := 0; i < len(resolvedNode.Content); i += 2 {
		if resolvedNode.Content[i].Value == key {
			return resolvedNode.Content[i]
		}
	}

	return n.ValueNode
}

func (n Node[V]) GetMapKeyNodeOrRootLine(key string, rootNode *yaml.Node) int {
	keyNode := n.GetMapKeyNodeOrRoot(key, rootNode)
	if keyNode == nil {
		return -1
	}
	return keyNode.Line
}

// Will return the value node for the map key, or the map root node or the provided root node if the node is not present
func (n Node[V]) GetMapValueNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}

	resolvedNode := yml.ResolveAlias(n.ValueNode)
	if resolvedNode == nil {
		return rootNode
	}

	for i := 0; i < len(resolvedNode.Content); i += 2 {
		if resolvedNode.Content[i].Value == key {
			return resolvedNode.Content[i+1]
		}
	}

	return n.ValueNode
}

func (n Node[V]) GetNavigableNode() (any, error) {
	return n.Value, nil
}
