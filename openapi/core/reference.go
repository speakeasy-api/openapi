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
		return nil, errors.New("node is nil")
	}

	if resolvedNode.Kind != yaml.MappingNode {
		r.SetValid(false, false)

		return []error{validation.NewValidationError(validation.NewTypeMismatchError(parentName, "reference expected object, got %s", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode)}, nil
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

	// Check if we have an object but no reference (inlined case)
	hasObject := !of.IsZero()
	hasReference := !rf.IsZero() && !rf.IsNil()

	if hasObject && !hasReference {
		// Inlined case: we have an object but no reference
		// Clear the reference in the core model and sync only the object
		r.Reference = marshaller.Node[*string]{}

		var err error
		valueNode, err = marshaller.SyncValue(ctx, of.Interface(), &r.Object, valueNode, false)
		if err != nil {
			return nil, err
		}

		// Also manually remove $ref from the YAML node if it exists
		if valueNode != nil && valueNode.Kind == yaml.MappingNode {
			// Remove $ref key-value pair from the mapping node
			newContent := make([]*yaml.Node, 0, len(valueNode.Content))
			for i := 0; i < len(valueNode.Content); i += 2 {
				if i+1 < len(valueNode.Content) && valueNode.Content[i].Value != "$ref" {
					newContent = append(newContent, valueNode.Content[i], valueNode.Content[i+1])
				}
			}
			valueNode.Content = newContent
		}

		// We are valid if the object is valid
		r.SetValid(r.Object.GetValid(), r.Object.GetValidYaml())
	} else {
		// Reference case: we have a reference but no object
		// Clear the object and sync the reference
		var zero T
		r.Object = zero

		var err error
		valueNode, err = marshaller.SyncValue(ctx, model, r, valueNode, true)
		if err != nil {
			return nil, err
		}

		// Also manually remove object fields from the YAML node if it exists
		// Keep only $ref, summary, and description fields
		if valueNode != nil && valueNode.Kind == yaml.MappingNode {
			newContent := make([]*yaml.Node, 0, len(valueNode.Content))
			for i := 0; i < len(valueNode.Content); i += 2 {
				if i+1 < len(valueNode.Content) {
					key := valueNode.Content[i].Value
					if key == "$ref" || key == "summary" || key == "description" {
						newContent = append(newContent, valueNode.Content[i], valueNode.Content[i+1])
					}
				}
			}
			valueNode.Content = newContent
		}

		r.SetValid(true, true)
	}

	r.SetRootNode(valueNode)
	return valueNode, nil
}
