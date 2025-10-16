package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

// Operation describes a single API operation on a path.
type Operation struct {
	marshaller.CoreModel `model:"operation"`

	Tags         marshaller.Node[[]string]                                  `key:"tags"`
	Summary      marshaller.Node[*string]                                   `key:"summary"`
	Description  marshaller.Node[*string]                                   `key:"description"`
	ExternalDocs marshaller.Node[*ExternalDocumentation]                    `key:"externalDocs"`
	OperationID  marshaller.Node[*string]                                   `key:"operationId"`
	Consumes     marshaller.Node[[]string]                                  `key:"consumes"`
	Produces     marshaller.Node[[]string]                                  `key:"produces"`
	Parameters   marshaller.Node[[]marshaller.Node[*Reference[*Parameter]]] `key:"parameters"`
	Responses    marshaller.Node[Responses]                                 `key:"responses"`
	Schemes      marshaller.Node[[]string]                                  `key:"schemes"`
	Deprecated   marshaller.Node[*bool]                                     `key:"deprecated"`
	Security     marshaller.Node[[]marshaller.Node[*SecurityRequirement]]   `key:"security"`
	Extensions   core.Extensions                                            `key:"extensions"`
}
