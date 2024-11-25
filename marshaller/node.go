package marshaller

import (
	"context"
	"reflect"

	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type NodeMutator interface {
	Unmarshal(ctx context.Context, keyNode, valueNode *yaml.Node) error
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

func (n *Node[V]) Unmarshal(ctx context.Context, keyNode, valueNode *yaml.Node) error {
	n.Key = keyNode.Value
	n.KeyNode = keyNode
	n.ValueNode = valueNode

	return Unmarshal(ctx, valueNode, &n.Value)
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
	return n.KeyNode, valueNode, nil
}

func (n *Node[V]) SetPresent(present bool) {
	n.Present = present
}

func (n Node[V]) GetKeyNodeOrRoot(rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.KeyNode == nil {
		return rootNode
	}
	return n.KeyNode
}

func (n Node[V]) GetValueNodeOrRoot(rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}
	return n.ValueNode
}

// Will return the value node for the slice index, or the slice root node or the provided root node if the node is not present
func (n Node[V]) GetSliceValueNodeOrRoot(idx int, rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}

	if idx < 0 || idx >= len(n.ValueNode.Content) {
		return n.ValueNode
	}

	return n.ValueNode.Content[idx]
}

// Will return the key node for the map key, or the map root node or the provided root node if the node is not present
func (n Node[V]) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}

	for i := 0; i < len(n.ValueNode.Content); i += 2 {
		if n.ValueNode.Content[i].Value == key {
			return n.ValueNode.Content[i]
		}
	}

	return n.ValueNode
}

// Will return the value node for the map key, or the map root node or the provided root node if the node is not present
func (n Node[V]) GetMapValueNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !n.Present || n.ValueNode == nil {
		return rootNode
	}

	for i := 0; i < len(n.ValueNode.Content); i += 2 {
		if n.ValueNode.Content[i].Value == key {
			return n.ValueNode.Content[i+1]
		}
	}

	return n.ValueNode
}

func (n Node[V]) GetNavigableNode() (any, error) {
	return n.Value, nil
}
