package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Encoding struct {
	marshaller.CoreModel

	ContentType   marshaller.Node[*string]                                        `key:"contentType"`
	Headers       marshaller.Node[*sequencedmap.Map[string, *Reference[*Header]]] `key:"headers"`
	Style         marshaller.Node[*string]                                        `key:"style"`
	Explode       marshaller.Node[*bool]                                          `key:"explode"`
	AllowReserved marshaller.Node[*bool]                                          `key:"allowReserved"`
	Extensions    core.Extensions                                                 `key:"extensions"`
}
