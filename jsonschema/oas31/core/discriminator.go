package core

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

type Discriminator struct {
	PropertyName marshaller.Node[string]                            `key:"propertyName"`
	Mapping      marshaller.Node[*sequencedmap.Map[string, string]] `key:"mapping"`
	Extensions   *extensions.Extensions                             `key:"extensions"`

	RootNode *yaml.Node
}

func (d *Discriminator) Unmarshal(ctx context.Context, node *yaml.Node) error {
	d.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, d)
}
