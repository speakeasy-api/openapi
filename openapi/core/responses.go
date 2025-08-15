package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Responses struct {
	marshaller.CoreModel `model:"responses"`
	*sequencedmap.Map[string, *Reference[*Response]]

	Default    marshaller.Node[*Reference[*Response]] `key:"default"`
	Extensions core.Extensions                        `key:"extensions"`
}

type Response struct {
	marshaller.CoreModel `model:"response"`

	Description marshaller.Node[string]                                         `key:"description"`
	Headers     marshaller.Node[*sequencedmap.Map[string, *Reference[*Header]]] `key:"headers"`
	Content     marshaller.Node[*sequencedmap.Map[string, *MediaType]]          `key:"content"`
	Links       marshaller.Node[*sequencedmap.Map[string, *Reference[*Link]]]   `key:"links"`
	Extensions  core.Extensions                                                 `key:"extensions"`
}
