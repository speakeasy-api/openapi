package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Server struct {
	marshaller.CoreModel

	URL         marshaller.Node[string]                                     `key:"url"`
	Description marshaller.Node[*string]                                    `key:"description"`
	Variables   marshaller.Node[*sequencedmap.Map[string, *ServerVariable]] `key:"variables"`
	Extensions  core.Extensions                                             `key:"extensions"`
}

type ServerVariable struct {
	marshaller.CoreModel

	Default     marshaller.Node[string]                    `key:"default"`
	Enum        marshaller.Node[[]marshaller.Node[string]] `key:"enum"`
	Description marshaller.Node[*string]                   `key:"description"`
	Extensions  core.Extensions                            `key:"extensions"`
}
