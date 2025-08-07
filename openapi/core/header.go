package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

type Header struct {
	marshaller.CoreModel

	Description marshaller.Node[*string]                                         `key:"description"`
	Required    marshaller.Node[*bool]                                           `key:"required"`
	Deprecated  marshaller.Node[*bool]                                           `key:"deprecated"`
	Style       marshaller.Node[*string]                                         `key:"style"`
	Explode     marshaller.Node[*bool]                                           `key:"explode"`
	Schema      marshaller.Node[oascore.JSONSchema]                              `key:"schema"`
	Content     marshaller.Node[*sequencedmap.Map[string, *MediaType]]           `key:"content"`
	Example     marshaller.Node[values.Value]                                    `key:"example"`
	Examples    marshaller.Node[*sequencedmap.Map[string, *Reference[*Example]]] `key:"examples"`
	Extensions  core.Extensions                                                  `key:"extensions"`
}
