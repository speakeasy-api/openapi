package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
)

type Example struct {
	marshaller.CoreModel

	Summary       marshaller.Node[*string]      `key:"summary"`
	Description   marshaller.Node[*string]      `key:"description"`
	Value         marshaller.Node[values.Value] `key:"value"`
	ExternalValue marshaller.Node[*string]      `key:"externalValue"`
	Extensions    core.Extensions               `key:"extensions"`
}
