package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// Swagger is the root document object for the API specification (Swagger 2.0).
type Swagger struct {
	marshaller.CoreModel `model:"swagger"`

	Swagger             marshaller.Node[string]                                                         `key:"swagger" required:"true"`
	Info                marshaller.Node[Info]                                                           `key:"info"`
	Host                marshaller.Node[*string]                                                        `key:"host"`
	BasePath            marshaller.Node[*string]                                                        `key:"basePath"`
	Schemes             marshaller.Node[[]string]                                                       `key:"schemes"`
	Consumes            marshaller.Node[[]string]                                                       `key:"consumes"`
	Produces            marshaller.Node[[]string]                                                       `key:"produces"`
	Paths               marshaller.Node[Paths]                                                          `key:"paths"`
	Definitions         marshaller.Node[*sequencedmap.Map[string, marshaller.Node[oascore.JSONSchema]]] `key:"definitions"`
	Parameters          marshaller.Node[*sequencedmap.Map[string, marshaller.Node[*Parameter]]]         `key:"parameters"`
	Responses           marshaller.Node[*sequencedmap.Map[string, marshaller.Node[*Response]]]          `key:"responses"`
	SecurityDefinitions marshaller.Node[*sequencedmap.Map[string, marshaller.Node[*SecurityScheme]]]    `key:"securityDefinitions"`
	Security            marshaller.Node[[]marshaller.Node[*SecurityRequirement]]                        `key:"security"`
	Tags                marshaller.Node[[]marshaller.Node[*Tag]]                                        `key:"tags"`
	ExternalDocs        marshaller.Node[*ExternalDocumentation]                                         `key:"externalDocs"`
	Extensions          core.Extensions                                                                 `key:"extensions"`
}
