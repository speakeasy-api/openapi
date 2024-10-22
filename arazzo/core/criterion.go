package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type CriterionExpressionType struct {
	Type    marshaller.Node[string] `key:"type"`
	Version marshaller.Node[string] `key:"version"`

	RootNode *yaml.Node
}

var _ CoreModel = (*CriterionExpressionType)(nil)

func (c *CriterionExpressionType) Unmarshal(ctx context.Context, node *yaml.Node) error {
	c.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, c)
}

type CriterionTypeUnion struct {
	Type           *string
	ExpressionType *CriterionExpressionType

	RootNode *yaml.Node
}

var _ CoreModel = (*CriterionTypeUnion)(nil)

func (c *CriterionTypeUnion) Unmarshal(ctx context.Context, node *yaml.Node) error {
	c.RootNode = node

	switch node.Kind {
	case yaml.ScalarNode:
		return node.Decode(&c.Type)
	case yaml.MappingNode:
		if c.ExpressionType == nil {
			c.ExpressionType = &CriterionExpressionType{}
		}
		return marshaller.UnmarshalStruct(ctx, node, c.ExpressionType)
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

	tv, err := marshaller.SyncValue(ctx, tf.Interface(), &c.Type, valueNode)
	if err != nil {
		return nil, err
	}

	ev, err := marshaller.SyncValue(ctx, ef.Interface(), &c.ExpressionType, valueNode)
	if err != nil {
		return nil, err
	}

	if tv != nil {
		c.RootNode = tv
		return tv, nil
	} else {
		c.RootNode = ev
		return ev, nil
	}
}

type Criterion struct {
	Context    marshaller.Node[*Expression]        `key:"context"`
	Condition  marshaller.Node[string]             `key:"condition"`
	Type       marshaller.Node[CriterionTypeUnion] `key:"type" required:"false"`
	Extensions core.Extensions                     `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*Criterion)(nil)

func (c *Criterion) Unmarshal(ctx context.Context, node *yaml.Node) error {
	c.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, c)
}
