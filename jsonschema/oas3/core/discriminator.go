package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Discriminator struct {
	marshaller.CoreModel `model:"discriminator"`

	PropertyName   marshaller.Node[string]                                             `key:"propertyName"`
	Mapping        marshaller.Node[*sequencedmap.Map[string, marshaller.Node[string]]] `key:"mapping"`
	DefaultMapping marshaller.Node[*string]                                            `key:"defaultMapping"`
	Extensions     core.Extensions                                                     `key:"extensions"`
}
