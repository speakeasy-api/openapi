package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type PayloadReplacement struct {
	Target     marshaller.Node[string]            `key:"target"`
	Value      marshaller.Node[ValueOrExpression] `key:"value" required:"true"`
	Extensions core.Extensions                    `key:"extensions"`

	RootNode *yaml.Node
}
