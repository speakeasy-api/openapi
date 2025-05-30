package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type ExternalDoc struct {
	marshaller.CoreModel
	Description marshaller.Node[*string] `key:"description"`
	URL         marshaller.Node[string]  `key:"url"`
	Extensions  core.Extensions          `key:"extensions"`
}
