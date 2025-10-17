package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

// Info provides metadata about the API.
type Info struct {
	marshaller.CoreModel `model:"info"`

	Title          marshaller.Node[string]   `key:"title"`
	Description    marshaller.Node[*string]  `key:"description"`
	TermsOfService marshaller.Node[*string]  `key:"termsOfService"`
	Contact        marshaller.Node[*Contact] `key:"contact"`
	License        marshaller.Node[*License] `key:"license"`
	Version        marshaller.Node[string]   `key:"version"`
	Extensions     core.Extensions           `key:"extensions"`
}

// Contact information for the exposed API.
type Contact struct {
	marshaller.CoreModel `model:"contact"`

	Name       marshaller.Node[*string] `key:"name"`
	URL        marshaller.Node[*string] `key:"url"`
	Email      marshaller.Node[*string] `key:"email"`
	Extensions core.Extensions          `key:"extensions"`
}

// License information for the exposed API.
type License struct {
	marshaller.CoreModel `model:"license"`

	Name       marshaller.Node[string]  `key:"name"`
	URL        marshaller.Node[*string] `key:"url"`
	Extensions core.Extensions          `key:"extensions"`
}
