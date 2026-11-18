package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// Channel is the backing model for the Channel Object.
//
// A null address and an omitted address both mean the address is unknown. An authored `address: null` is kept
// when syncing while the public address is still unknown, as with any explicit null.
type Channel struct {
	marshaller.CoreModel `model:"channel"`

	Address      marshaller.Node[*string]                                           `key:"address"`
	Messages     marshaller.Node[*sequencedmap.Map[string, *Reference[*Message]]]   `key:"messages"`
	Title        marshaller.Node[*string]                                           `key:"title"`
	Summary      marshaller.Node[*string]                                           `key:"summary"`
	Description  marshaller.Node[*string]                                           `key:"description"`
	Servers      marshaller.Node[[]*ReferenceObject]                                `key:"servers"`
	Parameters   marshaller.Node[*sequencedmap.Map[string, *Reference[*Parameter]]] `key:"parameters"`
	Tags         marshaller.Node[[]*Tag]                                            `key:"tags"`
	ExternalDocs marshaller.Node[*schemacore.ExternalDocumentation]                 `key:"externalDocs"`
	Bindings     marshaller.Node[*Reference[*Bindings]]                             `key:"bindings"`

	Extensions core.Extensions `key:"extensions"`
}

// Parameter is the backing model for the Parameter Object.
type Parameter struct {
	marshaller.CoreModel `model:"parameter"`

	Enum        marshaller.Node[[]marshaller.Node[string]] `key:"enum"`
	Default     marshaller.Node[*string]                   `key:"default"`
	Description marshaller.Node[*string]                   `key:"description"`
	Examples    marshaller.Node[[]marshaller.Node[string]] `key:"examples"`
	Location    marshaller.Node[*string]                   `key:"location"`

	Extensions core.Extensions `key:"extensions"`
}
