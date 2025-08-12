package core

import (
	expression "github.com/speakeasy-api/openapi/expression/core"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type PayloadReplacement struct {
	marshaller.CoreModel `model:"payloadReplacement"`

	Target     marshaller.Node[string]                       `key:"target"`
	Value      marshaller.Node[expression.ValueOrExpression] `key:"value" required:"true"`
	Extensions core.Extensions                               `key:"extensions"`
}
