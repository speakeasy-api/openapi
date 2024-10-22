package core

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type RequestBody struct {
	ContentType  marshaller.Node[*string]              `key:"contentType"`
	Payload      marshaller.Node[ValueOrExpression]    `key:"payload"`
	Replacements marshaller.Node[[]PayloadReplacement] `key:"replacements"`
	Extensions   core.Extensions                       `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*RequestBody)(nil)

func (r *RequestBody) Unmarshal(ctx context.Context, node *yaml.Node) error {
	r.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, r)
}
