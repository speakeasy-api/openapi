package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type Parameter struct {
	Name       marshaller.Node[string]            `key:"name"`
	In         marshaller.Node[*string]           `key:"in"`
	Value      marshaller.Node[ValueOrExpression] `key:"value" required:"true"`
	Extensions Extensions                         `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*Parameter)(nil)

func (p *Parameter) Unmarshal(ctx context.Context, node *yaml.Node) error {
	p.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, p)
}
