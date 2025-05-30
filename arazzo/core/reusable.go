package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Reusable[T marshaller.CoreModeler] struct {
	marshaller.CoreModel
	Reference marshaller.Node[*Expression] `key:"reference"`
	Value     marshaller.Node[Value]       `key:"value"`
	Object    T                            `populatorValue:"true"`
}

var _ interfaces.CoreModel = (*Reusable[*Parameter])(nil)

func (r *Reusable[T]) Unmarshal(ctx context.Context, node *yaml.Node) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node, got %v", node.Kind)
	}

	if _, _, ok := yml.GetMapElementNodes(ctx, node, "reference"); ok {
		return marshaller.UnmarshalModel(ctx, node, r)
	}

	var obj T
	if err := marshaller.Unmarshal(ctx, node, &obj); err != nil {
		return err
	}

	r.Object = obj
	r.SetValid(true)

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
		var err error
		valueNode, err = marshaller.SyncValue(ctx, model, r, valueNode, true)
		if err != nil {
			return nil, err
		}
		r.SetValid(true)
	} else {
		var err error
		valueNode, err = marshaller.SyncValue(ctx, of.Interface(), &r.Object, valueNode, false)
		if err != nil {
			return nil, err
		}

		// We are valid if the object is valid
		r.SetValid(r.Object.GetValid())
	}

	r.SetRootNode(valueNode)
	return valueNode, nil
}
