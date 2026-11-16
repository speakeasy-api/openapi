package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

type CriterionExpressionType struct {
	marshaller.CoreModel `model:"criterionExpressionType"`

	Type    marshaller.Node[string] `key:"type"`
	Version marshaller.Node[string] `key:"version"`
}

type CriterionTypeUnion struct {
	marshaller.CoreModel `model:"criterionTypeUnion"`

	Type           *string
	ExpressionType *CriterionExpressionType
}

var _ interfaces.CoreModel = (*CriterionTypeUnion)(nil)

func (c *CriterionTypeUnion) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)

	if resolvedNode == nil {
		return nil, errors.New("node is nil")
	}

	c.SetRootNode(node)

	var validationErrs []error

	switch resolvedNode.Kind {
	case yaml.ScalarNode:
		var err error
		validationErrs, err = marshaller.DecodeNode(ctx, parentName, resolvedNode, &c.Type)
		if err != nil {
			return nil, err
		}

		c.DetermineValidity(validationErrs)
	case yaml.MappingNode:
		if c.ExpressionType == nil {
			c.ExpressionType = &CriterionExpressionType{}
		}
		var err error
		validationErrs, err = marshaller.UnmarshalModel(ctx, node, c.ExpressionType)
		if err != nil {
			return nil, err
		}

		c.DetermineValidity(validationErrs)
	default:
		return []error{
			validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "criterionTypeUnion expected string or object, got %s", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode),
		}, nil
	}

	return validationErrs, nil
}

func (c *CriterionTypeUnion) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("CriterionTypeUnion.SyncChanges expected a struct, got `%s`", mv.Type())
	}

	tf := mv.FieldByName("Type")
	ef := mv.FieldByName("ExpressionType")

	// Only reuse the existing node for the side of the union of the same kind, so switching between a string and
	// an Expression Type Object (or vice versa) creates a new node rather than updating the old one.
	resolvedNode := yml.ResolveAlias(valueNode)
	var scalarNode, mappingNode *yaml.Node
	if resolvedNode != nil {
		switch resolvedNode.Kind {
		case yaml.ScalarNode:
			scalarNode = valueNode
		case yaml.MappingNode:
			mappingNode = valueNode
		case yaml.DocumentNode, yaml.SequenceNode, yaml.AliasNode:
		}
	}

	if tf.Kind() == reflect.Ptr && !tf.IsNil() {
		c.ExpressionType = nil

		tv, err := marshaller.SyncValue(ctx, tf.Interface(), &c.Type, scalarNode, false)
		if err != nil {
			return nil, err
		}
		c.SetRootNode(tv)
		return tv, nil
	}

	c.Type = nil

	if ef.Kind() == reflect.Ptr && ef.IsNil() {
		c.ExpressionType = nil
		c.SetRootNode(nil)
		return nil, nil
	}

	ev, err := marshaller.SyncValue(ctx, ef.Interface(), &c.ExpressionType, mappingNode, false)
	if err != nil {
		return nil, err
	}
	c.SetRootNode(ev)
	return ev, nil
}

type Criterion struct {
	marshaller.CoreModel `model:"criterion"`

	Context    marshaller.Node[*string]            `key:"context"`
	Condition  marshaller.Node[string]             `key:"condition"`
	Type       marshaller.Node[CriterionTypeUnion] `key:"type" required:"false"`
	Extensions core.Extensions                     `key:"extensions"`
}
