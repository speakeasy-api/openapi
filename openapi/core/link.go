package core

import (
	expression "github.com/speakeasy-api/openapi/expression/core"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Link struct {
	marshaller.CoreModel `model:"link"`

	OperationID  marshaller.Node[*string]                                                                  `key:"operationId"`
	OperationRef marshaller.Node[*string]                                                                  `key:"operationRef"`
	Parameters   marshaller.Node[*sequencedmap.Map[string, marshaller.Node[expression.ValueOrExpression]]] `key:"parameters"`
	RequestBody  marshaller.Node[expression.ValueOrExpression]                                             `key:"requestBody"`
	Description  marshaller.Node[*string]                                                                  `key:"description"`
	Server       marshaller.Node[*Server]                                                                  `key:"server"`
	Extensions   core.Extensions                                                                           `key:"extensions"`
}
