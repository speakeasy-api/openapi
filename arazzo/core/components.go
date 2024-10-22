package core

import (
	"context"

	coreExtensions "github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

type Components struct {
	Inputs         marshaller.Node[*sequencedmap.Map[string, core.JSONSchema]] `key:"inputs"`
	Parameters     marshaller.Node[*sequencedmap.Map[string, Parameter]]       `key:"parameters"`
	SuccessActions marshaller.Node[*sequencedmap.Map[string, SuccessAction]]   `key:"successActions"`
	FailureActions marshaller.Node[*sequencedmap.Map[string, FailureAction]]   `key:"failureActions"`
	Extensions     coreExtensions.Extensions                                   `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*Components)(nil)

func (c *Components) Unmarshal(ctx context.Context, node *yaml.Node) error {
	c.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, c)
}
