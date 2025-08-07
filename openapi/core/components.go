package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Components struct {
	marshaller.CoreModel
	Schemas         marshaller.Node[*sequencedmap.Map[string, oascore.JSONSchema]]          `key:"schemas"`
	Responses       marshaller.Node[*sequencedmap.Map[string, *Reference[*Response]]]       `key:"responses"`
	Parameters      marshaller.Node[*sequencedmap.Map[string, *Reference[*Parameter]]]      `key:"parameters"`
	Examples        marshaller.Node[*sequencedmap.Map[string, *Reference[*Example]]]        `key:"examples"`
	RequestBodies   marshaller.Node[*sequencedmap.Map[string, *Reference[*RequestBody]]]    `key:"requestBodies"`
	Headers         marshaller.Node[*sequencedmap.Map[string, *Reference[*Header]]]         `key:"headers"`
	SecuritySchemes marshaller.Node[*sequencedmap.Map[string, *Reference[*SecurityScheme]]] `key:"securitySchemes"`
	Links           marshaller.Node[*sequencedmap.Map[string, *Reference[*Link]]]           `key:"links"`
	Callbacks       marshaller.Node[*sequencedmap.Map[string, *Reference[*Callback]]]       `key:"callbacks"`
	PathItems       marshaller.Node[*sequencedmap.Map[string, *Reference[*PathItem]]]       `key:"pathItems"`

	Extensions core.Extensions `key:"extensions"`
}
