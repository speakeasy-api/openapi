package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type RequestBody struct {
	ContentType  marshaller.Node[*string]               `key:"contentType"`
	Payload      marshaller.Node[ValueOrExpression]     `key:"payload"`
	Replacements marshaller.Node[[]*PayloadReplacement] `key:"replacements"`
	Extensions   core.Extensions                        `key:"extensions"`

	RootNode *yaml.Node
}
