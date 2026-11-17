package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Selector struct {
	marshaller.CoreModel `model:"selector"`

	Context    marshaller.Node[string]             `key:"context" required:"true"`
	Selector   marshaller.Node[string]             `key:"selector" required:"true"`
	Type       marshaller.Node[CriterionTypeUnion] `key:"type" required:"true"`
	Extensions core.Extensions                     `key:"extensions"`
}
