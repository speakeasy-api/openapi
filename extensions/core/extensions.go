package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

type (
	Extension  = *yaml.Node
	Extensions = *sequencedmap.Map[string, marshaller.Node[Extension]]
)

func UnmarshalExtensionModel[L any](ctx context.Context, e Extensions, ext string) (*L, error) {
	if !e.Has(ext) {
		return nil, nil
	}

	node := e.GetOrZero(ext)

	var l L
	if err := marshaller.Unmarshal(ctx, node.Value, &l); err != nil {
		return nil, err
	}

	return &l, nil
}
