package core

import (
	expression "github.com/speakeasy-api/openapi/expression/core"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type RequestBody struct {
	marshaller.CoreModel `model:"requestBody"`

	ContentType  marshaller.Node[*string]                      `key:"contentType"`
	Payload      marshaller.Node[expression.ValueOrExpression] `key:"payload"`
	Replacements marshaller.Node[[]*PayloadReplacement]        `key:"replacements"`
	Extensions   core.Extensions                               `key:"extensions"`
}
