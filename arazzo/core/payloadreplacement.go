package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type PayloadReplacement struct {
	marshaller.CoreModel
	Target     marshaller.Node[string]            `key:"target"`
	Value      marshaller.Node[ValueOrExpression] `key:"value" required:"true"`
	Extensions core.Extensions                    `key:"extensions"`
}
