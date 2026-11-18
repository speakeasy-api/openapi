package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// Components is the backing model for the Components Object.
type Components struct {
	marshaller.CoreModel `model:"components"`

	Schemas           marshaller.Node[*sequencedmap.Map[string, *schemacore.SchemaUnion]]            `key:"schemas"`
	Servers           marshaller.Node[*sequencedmap.Map[string, *Reference[*Server]]]                `key:"servers"`
	Channels          marshaller.Node[*sequencedmap.Map[string, *Reference[*Channel]]]               `key:"channels"`
	Operations        marshaller.Node[*sequencedmap.Map[string, *Reference[*Operation]]]             `key:"operations"`
	Messages          marshaller.Node[*sequencedmap.Map[string, *Reference[*Message]]]               `key:"messages"`
	SecuritySchemes   marshaller.Node[*sequencedmap.Map[string, *Reference[*SecurityScheme]]]        `key:"securitySchemes"`
	ServerVariables   marshaller.Node[*sequencedmap.Map[string, *Reference[*ServerVariable]]]        `key:"serverVariables"`
	Parameters        marshaller.Node[*sequencedmap.Map[string, *Reference[*Parameter]]]             `key:"parameters"`
	CorrelationIDs    marshaller.Node[*sequencedmap.Map[string, *Reference[*CorrelationID]]]         `key:"correlationIds"`
	Replies           marshaller.Node[*sequencedmap.Map[string, *Reference[*OperationReply]]]        `key:"replies"`
	ReplyAddresses    marshaller.Node[*sequencedmap.Map[string, *Reference[*OperationReplyAddress]]] `key:"replyAddresses"`
	ExternalDocs      marshaller.Node[*sequencedmap.Map[string, *schemacore.ExternalDocumentation]]  `key:"externalDocs"`
	Tags              marshaller.Node[*sequencedmap.Map[string, *Tag]]                               `key:"tags"`
	OperationTraits   marshaller.Node[*sequencedmap.Map[string, *Reference[*OperationTrait]]]        `key:"operationTraits"`
	MessageTraits     marshaller.Node[*sequencedmap.Map[string, *Reference[*MessageTrait]]]          `key:"messageTraits"`
	ServerBindings    marshaller.Node[*sequencedmap.Map[string, *Reference[*Bindings]]]              `key:"serverBindings"`
	ChannelBindings   marshaller.Node[*sequencedmap.Map[string, *Reference[*Bindings]]]              `key:"channelBindings"`
	OperationBindings marshaller.Node[*sequencedmap.Map[string, *Reference[*Bindings]]]              `key:"operationBindings"`
	MessageBindings   marshaller.Node[*sequencedmap.Map[string, *Reference[*Bindings]]]              `key:"messageBindings"`

	Extensions core.Extensions `key:"extensions"`
}
