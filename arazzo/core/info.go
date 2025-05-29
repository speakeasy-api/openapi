package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type Info struct {
	Title       marshaller.Node[string]  `key:"title"`
	Summary     marshaller.Node[*string] `key:"summary"`
	Description marshaller.Node[*string] `key:"description"`
	Version     marshaller.Node[string]  `key:"version"`
	Extensions  core.Extensions          `key:"extensions"`

	RootNode *yaml.Node
}
