package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

// ExternalDocumentation allows referencing an external resource for extended documentation.
type ExternalDocumentation struct {
	marshaller.CoreModel `model:"externalDocumentation"`

	Description marshaller.Node[*string] `key:"description"`
	URL         marshaller.Node[string]  `key:"url"`
	Extensions  core.Extensions          `key:"extensions"`
}
