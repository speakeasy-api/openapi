package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

type Discriminator struct {
	PropertyName marshaller.Node[string]                            `key:"propertyName"`
	Mapping      marshaller.Node[*sequencedmap.Map[string, string]] `key:"mapping"`
	Extensions   core.Extensions                                    `key:"extensions"`

	RootNode *yaml.Node
}
