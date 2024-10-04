package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type PayloadReplacement struct {
	Target     marshaller.Node[string]            `key:"target"`
	Value      marshaller.Node[ValueOrExpression] `key:"value" required:"true"`
	Extensions Extensions                         `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*PayloadReplacement)(nil)

func (p *PayloadReplacement) Unmarshal(ctx context.Context, node *yaml.Node) error {
	p.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, p)
}
