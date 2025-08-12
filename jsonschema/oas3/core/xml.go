package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type XML struct {
	marshaller.CoreModel `model:"xml"`

	Name       marshaller.Node[*string] `key:"name"`
	Namespace  marshaller.Node[*string] `key:"namespace"`
	Prefix     marshaller.Node[*string] `key:"prefix"`
	Attribute  marshaller.Node[*bool]   `key:"attribute"`
	Wrapped    marshaller.Node[*bool]   `key:"wrapped"`
	Extensions core.Extensions          `key:"extensions"`
}
