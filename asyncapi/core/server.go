package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// Server is the backing model for the Server Object.
type Server struct {
	marshaller.CoreModel `model:"server"`

	Host            marshaller.Node[string]                                                 `key:"host"`
	Protocol        marshaller.Node[string]                                                 `key:"protocol"`
	ProtocolVersion marshaller.Node[*string]                                                `key:"protocolVersion"`
	Pathname        marshaller.Node[*string]                                                `key:"pathname"`
	Description     marshaller.Node[*string]                                                `key:"description"`
	Title           marshaller.Node[*string]                                                `key:"title"`
	Summary         marshaller.Node[*string]                                                `key:"summary"`
	Variables       marshaller.Node[*sequencedmap.Map[string, *Reference[*ServerVariable]]] `key:"variables"`
	Security        marshaller.Node[[]*Reference[*SecurityScheme]]                          `key:"security"`
	Tags            marshaller.Node[[]*Tag]                                                 `key:"tags"`
	ExternalDocs    marshaller.Node[*schemacore.ExternalDocumentation]                      `key:"externalDocs"`
	Bindings        marshaller.Node[*Reference[*Bindings]]                                  `key:"bindings"`

	Extensions core.Extensions `key:"extensions"`
}

// ServerVariable is the backing model for the Server Variable Object.
type ServerVariable struct {
	marshaller.CoreModel `model:"serverVariable"`

	Enum        marshaller.Node[[]marshaller.Node[string]] `key:"enum"`
	Default     marshaller.Node[*string]                   `key:"default"`
	Description marshaller.Node[*string]                   `key:"description"`
	Examples    marshaller.Node[[]marshaller.Node[string]] `key:"examples"`

	Extensions core.Extensions `key:"extensions"`
}
