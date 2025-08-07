package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Info struct {
	marshaller.CoreModel
	Title          marshaller.Node[string]   `key:"title"`
	Version        marshaller.Node[string]   `key:"version"`
	Summary        marshaller.Node[*string]  `key:"summary"`
	Description    marshaller.Node[*string]  `key:"description"`
	TermsOfService marshaller.Node[*string]  `key:"termsOfService"`
	Contact        marshaller.Node[*Contact] `key:"contact"`
	License        marshaller.Node[*License] `key:"license"`
	Extensions     core.Extensions           `key:"extensions"`
}

type Contact struct {
	marshaller.CoreModel
	Name       marshaller.Node[*string] `key:"name"`
	URL        marshaller.Node[*string] `key:"url"`
	Email      marshaller.Node[*string] `key:"email"`
	Extensions core.Extensions          `key:"extensions"`
}

type License struct {
	marshaller.CoreModel
	Name       marshaller.Node[string]  `key:"name"`
	Identifier marshaller.Node[*string] `key:"identifier"`
	URL        marshaller.Node[*string] `key:"url"`
	Extensions core.Extensions          `key:"extensions"`
}
