package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Reusable[T any] struct {
	Reference marshaller.Node[*Expression] `key:"reference"`
	Value     marshaller.Node[Value]       `key:"value"`
	Object    *T                           `populatorValue:"true"`

	RootNode *yaml.Node
}

var _ CoreModel = (*Reusable[any])(nil)

func (r *Reusable[T]) Unmarshal(ctx context.Context, node *yaml.Node) error {
	r.RootNode = node

	if node == nil {
		return fmt.Errorf("node is nil")
	}

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node, got %v", node.Kind)
	}

	if _, _, ok := yml.GetMapElementNodes(ctx, node, "reference"); ok {
		return marshaller.UnmarshalStruct(ctx, node, r)
	}

	var obj T
	if err := marshaller.Unmarshal(ctx, node, &obj); err != nil {
		return err
	}

	r.Object = &obj

	return nil
}

func (r *Reusable[T]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Reusable.SyncChanges expected a struct, got %s", mv.Kind())
	}

	of := mv.FieldByName("Object")
	if of.IsZero() {
		type reusable[T any] struct {
			Reference marshaller.Node[*Expression] `key:"reference"`
			Value     marshaller.Node[Value]       `key:"value"`

			RootNode *yaml.Node
		}

		rl := reusable[T]{
			Reference: r.Reference,
			Value:     r.Value,
			RootNode:  r.RootNode,
		}

		var err error
		valueNode, err = marshaller.SyncValue(ctx, model, &rl, valueNode)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		valueNode, err = marshaller.SyncValue(ctx, of.Interface(), &r.Object, valueNode)
		if err != nil {
			return nil, err
		}
	}

	r.RootNode = valueNode
	return valueNode, nil
}
