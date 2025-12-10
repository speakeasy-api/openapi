package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Server struct {
	marshaller.CoreModel `model:"server"`

	URL         marshaller.Node[string]                                     `key:"url"`
	Description marshaller.Node[*string]                                    `key:"description"`
	Name        marshaller.Node[*string]                                    `key:"name"`
	Variables   marshaller.Node[*sequencedmap.Map[string, *ServerVariable]] `key:"variables"`
	Extensions  core.Extensions                                             `key:"extensions"`
}

type ServerVariable struct {
	marshaller.CoreModel `model:"serverVariable"`

	Default     marshaller.Node[string]                    `key:"default"`
	Enum        marshaller.Node[[]marshaller.Node[string]] `key:"enum"`
	Description marshaller.Node[*string]                   `key:"description"`
	Extensions  core.Extensions                            `key:"extensions"`
}
