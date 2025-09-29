package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oas3core "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Tag struct {
	marshaller.CoreModel `model:"tag"`

	Name         marshaller.Node[string]                          `key:"name"`
	Summary      marshaller.Node[*string]                         `key:"summary"`
	Description  marshaller.Node[*string]                         `key:"description"`
	ExternalDocs marshaller.Node[*oas3core.ExternalDocumentation] `key:"externalDocs"`
	Parent       marshaller.Node[*string]                         `key:"parent"`
	Kind         marshaller.Node[*string]                         `key:"kind"`
	Extensions   core.Extensions                                  `key:"extensions"`
}
