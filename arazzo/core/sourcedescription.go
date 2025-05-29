package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type SourceDescription struct {
	Name       marshaller.Node[string] `key:"name"`
	URL        marshaller.Node[string] `key:"url"`
	Type       marshaller.Node[string] `key:"type"`
	Extensions core.Extensions         `key:"extensions"`

	RootNode *yaml.Node
}
