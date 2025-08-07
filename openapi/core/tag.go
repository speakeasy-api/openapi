package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oas3core "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Tag struct {
	marshaller.CoreModel

	Name         marshaller.Node[string]                          `key:"name"`
	Description  marshaller.Node[*string]                         `key:"description"`
	ExternalDocs marshaller.Node[*oas3core.ExternalDocumentation] `key:"externalDocs"`
	Extensions   core.Extensions                                  `key:"extensions"`
}
