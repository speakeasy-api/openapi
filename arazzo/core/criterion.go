package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type CriterionExpressionType struct {
	marshaller.CoreModel
	Type    marshaller.Node[string] `key:"type"`
	Version marshaller.Node[string] `key:"version"`
}

type CriterionTypeUnion struct {
	marshaller.CoreModel
	Type           *string
	ExpressionType *CriterionExpressionType
}

var _ interfaces.CoreModel = (*CriterionTypeUnion)(nil)

func (c *CriterionTypeUnion) Unmarshal(ctx context.Context, node *yaml.Node) error {
	c.SetRootNode(node)
	c.SetValid(true)

	switch node.Kind {
	case yaml.ScalarNode:
		return node.Decode(&c.Type)
	case yaml.MappingNode:
		if c.ExpressionType == nil {
			c.ExpressionType = &CriterionExpressionType{}
		}
		return marshaller.UnmarshalModel(ctx, node, c.ExpressionType)
	default:
		return fmt.Errorf("expected scalar or mapping node, got %v", node.Kind)
	}
}

func (c *CriterionTypeUnion) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("CriterionTypeUnion.SyncChanges expected a struct, got %s", mv.Type())
	}

	tf := mv.FieldByName("Type")
	ef := mv.FieldByName("ExpressionType")

	tv, err := marshaller.SyncValue(ctx, tf.Interface(), &c.Type, valueNode, false)
	if err != nil {
		return nil, err
	}

	ev, err := marshaller.SyncValue(ctx, ef.Interface(), &c.ExpressionType, valueNode, false)
	if err != nil {
		return nil, err
	}

	if tv != nil {
		c.SetRootNode(tv)
		return tv, nil
	} else {
		c.SetRootNode(ev)
		return ev, nil
	}
}

type Criterion struct {
	marshaller.CoreModel
	Context    marshaller.Node[*Expression]        `key:"context"`
	Condition  marshaller.Node[string]             `key:"condition"`
	Type       marshaller.Node[CriterionTypeUnion] `key:"type" required:"false"`
	Extensions core.Extensions                     `key:"extensions"`
}
