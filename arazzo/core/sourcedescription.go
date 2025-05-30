package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type SourceDescription struct {
	marshaller.CoreModel
	Name       marshaller.Node[string] `key:"name"`
	URL        marshaller.Node[string] `key:"url"`
	Type       marshaller.Node[string] `key:"type"`
	Extensions core.Extensions         `key:"extensions"`
}
