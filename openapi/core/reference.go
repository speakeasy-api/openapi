package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Reference[T marshaller.CoreModeler] struct {
	marshaller.CoreModel `model:"reference"`

	Reference   marshaller.Node[*string] `key:"$ref"`
	Summary     marshaller.Node[*string] `key:"summary"`
	Description marshaller.Node[*string] `key:"description"`

	Object T `populatorValue:"true"`
}

var _ interfaces.CoreModel = (*Reference[*PathItem])(nil)

func (r *Reference[T]) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, fmt.Errorf("node is nil")
	}

	if resolvedNode.Kind != yaml.MappingNode {
		r.SetValid(false, false)

		return []error{validation.NewValidationError(validation.NewTypeMismatchError("reference expected mapping node, got %s", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode)}, nil
	}

	if _, _, ok := yml.GetMapElementNodes(ctx, resolvedNode, "$ref"); ok {
		return marshaller.UnmarshalModel(ctx, node, r)
	}

	var obj T

	validationErrs, err := marshaller.UnmarshalCore(ctx, parentName, node, &obj)
	if err != nil {
		return nil, err
	}

	r.Object = obj
	r.SetValid(r.Object.GetValid(), r.Object.GetValidYaml() && len(validationErrs) == 0)

	return validationErrs, nil
}

func (r *Reference[T]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Reference.SyncChanges expected a struct, got %s", mv.Kind())
	}

	of := mv.FieldByName("Object")
	if of.IsZero() {
		var err error
		valueNode, err = marshaller.SyncValue(ctx, model, r, valueNode, true)
		if err != nil {
			return nil, err
		}
		r.SetValid(true, true)
	} else {
		var err error
		valueNode, err = marshaller.SyncValue(ctx, of.Interface(), &r.Object, valueNode, false)
		if err != nil {
			return nil, err
		}

		// We are valid if the object is valid
		r.SetValid(r.Object.GetValid(), r.Object.GetValidYaml())
	}

	r.SetRootNode(valueNode)
	return valueNode, nil
}
