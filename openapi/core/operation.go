package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oas3core "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Operation struct {
	marshaller.CoreModel `model:"operation"`

	OperationID  marshaller.Node[*string]                                          `key:"operationId"`
	Summary      marshaller.Node[*string]                                          `key:"summary"`
	Description  marshaller.Node[*string]                                          `key:"description"`
	Tags         marshaller.Node[[]marshaller.Node[string]]                        `key:"tags"`
	Servers      marshaller.Node[[]*Server]                                        `key:"servers"`
	Security     marshaller.Node[[]*SecurityRequirement]                           `key:"security"`
	Parameters   marshaller.Node[[]*Reference[*Parameter]]                         `key:"parameters"`
	RequestBody  marshaller.Node[*Reference[*RequestBody]]                         `key:"requestBody"`
	Responses    marshaller.Node[Responses]                                        `key:"responses"`
	Callbacks    marshaller.Node[*sequencedmap.Map[string, *Reference[*Callback]]] `key:"callbacks"`
	Deprecated   marshaller.Node[*bool]                                            `key:"deprecated"`
	ExternalDocs marshaller.Node[*oas3core.ExternalDocumentation]                  `key:"externalDocs"`
	Extensions   core.Extensions                                                   `key:"extensions"`
}
