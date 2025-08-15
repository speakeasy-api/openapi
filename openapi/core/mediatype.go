package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

type MediaType struct {
	marshaller.CoreModel `model:"mediaType"`

	Schema     marshaller.Node[oascore.JSONSchema]                              `key:"schema"`
	Encoding   marshaller.Node[*sequencedmap.Map[string, *Encoding]]            `key:"encoding"`
	Example    marshaller.Node[values.Value]                                    `key:"example"`
	Examples   marshaller.Node[*sequencedmap.Map[string, *Reference[*Example]]] `key:"examples"`
	Extensions core.Extensions                                                  `key:"extensions"`
}
