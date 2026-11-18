package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
)

// Message is the backing model for the Message Object.
type Message struct {
	marshaller.CoreModel `model:"message"`

	Headers       marshaller.Node[*schemacore.SchemaUnion]           `key:"headers"`
	Payload       marshaller.Node[*schemacore.SchemaUnion]           `key:"payload"`
	CorrelationID marshaller.Node[*Reference[*CorrelationID]]        `key:"correlationId"`
	ContentType   marshaller.Node[*string]                           `key:"contentType"`
	Name          marshaller.Node[*string]                           `key:"name"`
	Title         marshaller.Node[*string]                           `key:"title"`
	Summary       marshaller.Node[*string]                           `key:"summary"`
	Description   marshaller.Node[*string]                           `key:"description"`
	Tags          marshaller.Node[[]*Tag]                            `key:"tags"`
	ExternalDocs  marshaller.Node[*schemacore.ExternalDocumentation] `key:"externalDocs"`
	Bindings      marshaller.Node[*Reference[*Bindings]]             `key:"bindings"`
	Examples      marshaller.Node[[]*MessageExample]                 `key:"examples"`
	Traits        marshaller.Node[[]*Reference[*MessageTrait]]       `key:"traits"`

	Extensions core.Extensions `key:"extensions"`
}

// MessageTrait is the backing model for the Message Trait Object.
type MessageTrait struct {
	marshaller.CoreModel `model:"messageTrait"`

	Headers       marshaller.Node[*schemacore.SchemaUnion]           `key:"headers"`
	CorrelationID marshaller.Node[*Reference[*CorrelationID]]        `key:"correlationId"`
	ContentType   marshaller.Node[*string]                           `key:"contentType"`
	Name          marshaller.Node[*string]                           `key:"name"`
	Title         marshaller.Node[*string]                           `key:"title"`
	Summary       marshaller.Node[*string]                           `key:"summary"`
	Description   marshaller.Node[*string]                           `key:"description"`
	Tags          marshaller.Node[[]*Tag]                            `key:"tags"`
	ExternalDocs  marshaller.Node[*schemacore.ExternalDocumentation] `key:"externalDocs"`
	Bindings      marshaller.Node[*Reference[*Bindings]]             `key:"bindings"`
	Examples      marshaller.Node[[]*MessageExample]                 `key:"examples"`

	Extensions core.Extensions `key:"extensions"`
}

// MessageExample is the backing model for the Message Example Object.
type MessageExample struct {
	marshaller.CoreModel `model:"messageExample"`

	Headers marshaller.Node[values.Value] `key:"headers"`
	Payload marshaller.Node[values.Value] `key:"payload"`
	Name    marshaller.Node[*string]      `key:"name"`
	Summary marshaller.Node[*string]      `key:"summary"`

	Extensions core.Extensions `key:"extensions"`
}

// CorrelationID is the backing model for the Correlation ID Object.
type CorrelationID struct {
	marshaller.CoreModel `model:"correlationId"`

	Description marshaller.Node[*string] `key:"description"`
	Location    marshaller.Node[string]  `key:"location"`

	Extensions core.Extensions `key:"extensions"`
}
