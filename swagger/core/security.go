package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// SecurityScheme defines a security scheme that can be used by the operations.
type SecurityScheme struct {
	marshaller.CoreModel `model:"securityScheme"`

	Type             marshaller.Node[string]                                             `key:"type"`
	Description      marshaller.Node[*string]                                            `key:"description"`
	Name             marshaller.Node[*string]                                            `key:"name"`
	In               marshaller.Node[*string]                                            `key:"in"`
	Flow             marshaller.Node[*string]                                            `key:"flow"`
	AuthorizationURL marshaller.Node[*string]                                            `key:"authorizationUrl"`
	TokenURL         marshaller.Node[*string]                                            `key:"tokenUrl"`
	Scopes           marshaller.Node[*sequencedmap.Map[string, marshaller.Node[string]]] `key:"scopes"`
	Extensions       core.Extensions                                                     `key:"extensions"`
}

// SecurityRequirement lists the required security schemes to execute an operation.
type SecurityRequirement struct {
	marshaller.CoreModel `model:"securityRequirement"`
	*sequencedmap.Map[string, marshaller.Node[[]string]]
}

func NewSecurityRequirement() *SecurityRequirement {
	return &SecurityRequirement{
		Map: sequencedmap.New[string, marshaller.Node[[]string]](),
	}
}
