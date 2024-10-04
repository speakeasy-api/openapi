package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type ExternalDoc struct {
	Description marshaller.Node[*string] `key:"description"`
	URL         marshaller.Node[string]  `key:"url"`
	Extensions  Extensions               `key:"extensions"`

	RootNode *yaml.Node
}

func (e *ExternalDoc) Unmarshal(ctx context.Context, node *yaml.Node) error {
	e.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, e)
}
