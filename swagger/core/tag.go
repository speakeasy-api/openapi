package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

// Tag allows adding metadata to a single tag that is used by operations.
type Tag struct {
	marshaller.CoreModel `model:"tag"`

	Name         marshaller.Node[string]                 `key:"name"`
	Description  marshaller.Node[*string]                `key:"description"`
	ExternalDocs marshaller.Node[*ExternalDocumentation] `key:"externalDocs"`
	Extensions   core.Extensions                         `key:"extensions"`
}
