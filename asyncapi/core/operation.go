package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

// Operation is the backing model for the Operation Object.
type Operation struct {
	marshaller.CoreModel `model:"operation"`

	Action       marshaller.Node[string]                            `key:"action"`
	Channel      marshaller.Node[ReferenceObject]                   `key:"channel"`
	Title        marshaller.Node[*string]                           `key:"title"`
	Summary      marshaller.Node[*string]                           `key:"summary"`
	Description  marshaller.Node[*string]                           `key:"description"`
	Security     marshaller.Node[[]*Reference[*SecurityScheme]]     `key:"security"`
	Tags         marshaller.Node[[]*Tag]                            `key:"tags"`
	ExternalDocs marshaller.Node[*schemacore.ExternalDocumentation] `key:"externalDocs"`
	Bindings     marshaller.Node[*Reference[*Bindings]]             `key:"bindings"`
	Traits       marshaller.Node[[]*Reference[*OperationTrait]]     `key:"traits"`
	Messages     marshaller.Node[[]*ReferenceObject]                `key:"messages"`
	Reply        marshaller.Node[*Reference[*OperationReply]]       `key:"reply"`

	Extensions core.Extensions `key:"extensions"`
}

// OperationTrait is the backing model for the Operation Trait Object.
//
// Reply is modelled because the specification's prose allows a trait to hold any Operation Object property other
// than action, channel, messages and traits, even though its fixed field table omits reply.
type OperationTrait struct {
	marshaller.CoreModel `model:"operationTrait"`

	Title        marshaller.Node[*string]                           `key:"title"`
	Summary      marshaller.Node[*string]                           `key:"summary"`
	Description  marshaller.Node[*string]                           `key:"description"`
	Security     marshaller.Node[[]*Reference[*SecurityScheme]]     `key:"security"`
	Tags         marshaller.Node[[]*Tag]                            `key:"tags"`
	ExternalDocs marshaller.Node[*schemacore.ExternalDocumentation] `key:"externalDocs"`
	Bindings     marshaller.Node[*Reference[*Bindings]]             `key:"bindings"`
	Reply        marshaller.Node[*Reference[*OperationReply]]       `key:"reply"`

	Extensions core.Extensions `key:"extensions"`
}

// OperationReply is the backing model for the Operation Reply Object.
type OperationReply struct {
	marshaller.CoreModel `model:"operationReply"`

	Address  marshaller.Node[*Reference[*OperationReplyAddress]] `key:"address"`
	Channel  marshaller.Node[*ReferenceObject]                   `key:"channel"`
	Messages marshaller.Node[[]*ReferenceObject]                 `key:"messages"`

	Extensions core.Extensions `key:"extensions"`
}

// OperationReplyAddress is the backing model for the Operation Reply Address Object.
type OperationReplyAddress struct {
	marshaller.CoreModel `model:"operationReplyAddress"`

	Description marshaller.Node[*string] `key:"description"`
	Location    marshaller.Node[string]  `key:"location"`

	Extensions core.Extensions `key:"extensions"`
}
