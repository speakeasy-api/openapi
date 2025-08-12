package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type RequestBody struct {
	marshaller.CoreModel `model:"requestBody"`

	Description marshaller.Node[*string]                               `key:"description"`
	Content     marshaller.Node[*sequencedmap.Map[string, *MediaType]] `key:"content" required:"true"`
	Required    marshaller.Node[*bool]                                 `key:"required"`
	Extensions  core.Extensions                                        `key:"extensions"`
}
