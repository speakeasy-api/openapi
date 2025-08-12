package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

type Parameter struct {
	marshaller.CoreModel `model:"parameter"`

	Name            marshaller.Node[string]                                          `key:"name"`
	In              marshaller.Node[string]                                          `key:"in"`
	Description     marshaller.Node[*string]                                         `key:"description"`
	Required        marshaller.Node[*bool]                                           `key:"required"`
	Deprecated      marshaller.Node[*bool]                                           `key:"deprecated"`
	AllowEmptyValue marshaller.Node[*bool]                                           `key:"allowEmptyValue"`
	Style           marshaller.Node[*string]                                         `key:"style"`
	Explode         marshaller.Node[*bool]                                           `key:"explode"`
	AllowReserved   marshaller.Node[*bool]                                           `key:"allowReserved"`
	Schema          marshaller.Node[oascore.JSONSchema]                              `key:"schema"`
	Content         marshaller.Node[*sequencedmap.Map[string, *MediaType]]           `key:"content"`
	Example         marshaller.Node[values.Value]                                    `key:"example"`
	Examples        marshaller.Node[*sequencedmap.Map[string, *Reference[*Example]]] `key:"examples"`
	Extensions      core.Extensions                                                  `key:"extensions"`
}
