package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	values "github.com/speakeasy-api/openapi/values/core"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v4"
)

type Reusable[T marshaller.CoreModeler] struct {
	marshaller.CoreModel `model:"reusable"`

	Reference marshaller.Node[*string]      `key:"reference"`
	Value     marshaller.Node[values.Value] `key:"value"`
	Object    T                             `populatorValue:"true"`
}

var _ interfaces.CoreModel = (*Reusable[*Parameter])(nil)

func (r *Reusable[T]) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)

	if resolvedNode == nil {
		return nil, errors.New("node is nil")
	}

	if resolvedNode.Kind != yaml.MappingNode {
		r.SetValid(false, false)

		return []error{
			validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "reusable expected `object`, got %s", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode),
		}, nil
	}

	if _, _, ok := yml.GetMapElementNodes(ctx, resolvedNode, "reference"); ok {
		return marshaller.UnmarshalModel(ctx, node, r)
	}

	var obj T
	validationErrs, err := marshaller.UnmarshalCore(ctx, parentName, node, &obj)
	if err != nil {
		return nil, err
	}

	r.Object = obj
	r.DetermineValidity(validationErrs)

	return validationErrs, nil
}

func (r *Reusable[T]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Reusable.SyncChanges expected a struct, got `%s`", mv.Kind())
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
