package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type Info struct {
	Title       marshaller.Node[string]  `key:"title"`
	Summary     marshaller.Node[*string] `key:"summary"`
	Description marshaller.Node[*string] `key:"description"`
	Version     marshaller.Node[string]  `key:"version"`
	Extensions  Extensions               `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*Info)(nil)

func (i *Info) Unmarshal(ctx context.Context, node *yaml.Node) error {
	i.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, i)
}
