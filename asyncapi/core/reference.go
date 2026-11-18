package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// Reference is the backing model for a value that is either an inline object or an AsyncAPI Reference Object.
// AsyncAPI Reference Objects only contain `$ref`; any sibling properties are ignored.
type Reference[T marshaller.CoreModeler] struct {
	marshaller.CoreModel `model:"reference"`

	Reference marshaller.Node[*string] `key:"$ref"`

	Object T `populatorValue:"true"`
}

var _ interfaces.CoreModel = (*Reference[*Server])(nil)

func (r *Reference[T]) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, errors.New("node is nil")
	}

	if resolvedNode.Kind != yaml.MappingNode {
		r.SetValid(false, false)

		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "reference expected `object`, got %s", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode)}, nil
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
	rf := mv.FieldByName("Reference")

	hasObject := !of.IsZero()
	hasReference := !rf.IsZero() && !rf.IsNil()

	if hasObject && !hasReference {
		// Inline object: clear the reference and sync only the object.
		r.Reference = marshaller.Node[*string]{}

		var err error
		valueNode, err = marshaller.SyncValue(ctx, of.Interface(), &r.Object, valueNode, false)
		if err != nil {
			return nil, err
		}

		if valueNode != nil && valueNode.Kind == yaml.MappingNode {
			valueNode.Content = filterMappingKeys(valueNode.Content, func(key string) bool { return key != "$ref" })
		}

		r.SetValid(r.Object.GetValid(), r.Object.GetValidYaml())
	} else {
		// Reference Object: clear the object and sync only `$ref`.
		var zero T
		r.Object = zero

		var err error
		valueNode, err = marshaller.SyncValue(ctx, model, r, valueNode, true)
		if err != nil {
			return nil, err
		}

		if valueNode != nil && valueNode.Kind == yaml.MappingNode {
			valueNode.Content = filterMappingKeys(valueNode.Content, func(key string) bool { return key == "$ref" })
		}

		r.SetValid(true, true)
	}

	r.SetRootNode(valueNode)
	return valueNode, nil
}

// ReferenceObject is the backing model for a field that must be an AsyncAPI Reference Object.
type ReferenceObject struct {
	marshaller.CoreModel `model:"referenceObject"`

	Reference marshaller.Node[string] `key:"$ref"`
}

func filterMappingKeys(content []*yaml.Node, keep func(key string) bool) []*yaml.Node {
	out := make([]*yaml.Node, 0, len(content))
	for i := 0; i+1 < len(content); i += 2 {
		if keep(content[i].Value) {
			out = append(out, content[i], content[i+1])
		}
	}
	return out
}
