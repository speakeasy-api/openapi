package core

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type SourceDescription struct {
	Name       marshaller.Node[string] `key:"name"`
	URL        marshaller.Node[string] `key:"url"`
	Type       marshaller.Node[string] `key:"type"`
	Extensions core.Extensions         `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*SourceDescription)(nil)

func (s *SourceDescription) Unmarshal(ctx context.Context, node *yaml.Node) error {
	s.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, s)
}
