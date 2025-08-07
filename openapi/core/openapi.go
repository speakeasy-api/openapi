package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oas3core "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type OpenAPI struct {
	marshaller.CoreModel

	OpenAPI      marshaller.Node[string]                                           `key:"openapi"`
	Info         marshaller.Node[Info]                                             `key:"info"`
	ExternalDocs marshaller.Node[*oas3core.ExternalDocumentation]                  `key:"externalDocs"`
	Tags         marshaller.Node[[]*Tag]                                           `key:"tags"`
	Servers      marshaller.Node[[]*Server]                                        `key:"servers"`
	Security     marshaller.Node[[]*SecurityRequirement]                           `key:"security"`
	Paths        marshaller.Node[*Paths]                                           `key:"paths"`
	Webhooks     marshaller.Node[*sequencedmap.Map[string, *Reference[*PathItem]]] `key:"webhooks"`

	Components marshaller.Node[*Components] `key:"components"`

	JSONSchemaDialect marshaller.Node[*string] `key:"jsonSchemaDialect"`

	Extensions core.Extensions `key:"extensions"`
}
