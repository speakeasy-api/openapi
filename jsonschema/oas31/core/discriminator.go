package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Discriminator struct {
	marshaller.CoreModel
	PropertyName marshaller.Node[string]                            `key:"propertyName"`
	Mapping      marshaller.Node[*sequencedmap.Map[string, string]] `key:"mapping"`
	Extensions   core.Extensions                                    `key:"extensions"`
}
