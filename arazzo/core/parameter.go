package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Parameter struct {
	marshaller.CoreModel
	Name       marshaller.Node[string]            `key:"name"`
	In         marshaller.Node[*string]           `key:"in"`
	Value      marshaller.Node[ValueOrExpression] `key:"value" required:"true"`
	Extensions core.Extensions                    `key:"extensions"`
}
