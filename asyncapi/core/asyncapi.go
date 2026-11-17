// Package core contains the backing models for AsyncAPI documents, retaining the YAML/JSON nodes they were parsed from.
package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
)

// AsyncAPI is the backing model for the root AsyncAPI Object.
type AsyncAPI struct {
	marshaller.CoreModel `model:"asyncapi"`

	AsyncAPI           marshaller.Node[string]       `key:"asyncapi"`
	ID                 marshaller.Node[*string]      `key:"id"`
	Info               marshaller.Node[Info]         `key:"info"`
	Servers            marshaller.Node[values.Value] `key:"servers"`
	DefaultContentType marshaller.Node[*string]      `key:"defaultContentType"`
	Channels           marshaller.Node[values.Value] `key:"channels"`
	Operations         marshaller.Node[values.Value] `key:"operations"`
	Components         marshaller.Node[values.Value] `key:"components"`

	Extensions core.Extensions `key:"extensions"`
}

// Info is the backing model for the Info Object.
type Info struct {
	marshaller.CoreModel `model:"info"`

	Title          marshaller.Node[string]                            `key:"title"`
	Version        marshaller.Node[string]                            `key:"version"`
	Description    marshaller.Node[*string]                           `key:"description"`
	TermsOfService marshaller.Node[*string]                           `key:"termsOfService"`
	Contact        marshaller.Node[*Contact]                          `key:"contact"`
	License        marshaller.Node[*License]                          `key:"license"`
	Tags           marshaller.Node[[]*Tag]                            `key:"tags"`
	ExternalDocs   marshaller.Node[*schemacore.ExternalDocumentation] `key:"externalDocs"`

	Extensions core.Extensions `key:"extensions"`
}

// Contact is the backing model for the Contact Object.
type Contact struct {
	marshaller.CoreModel `model:"contact"`

	Name  marshaller.Node[*string] `key:"name"`
	URL   marshaller.Node[*string] `key:"url"`
	Email marshaller.Node[*string] `key:"email"`

	Extensions core.Extensions `key:"extensions"`
}

// License is the backing model for the License Object.
type License struct {
	marshaller.CoreModel `model:"license"`

	Name marshaller.Node[string]  `key:"name"`
	URL  marshaller.Node[*string] `key:"url"`

	Extensions core.Extensions `key:"extensions"`
}

// Tag is the backing model for the Tag Object, or a Reference Object to one.
type Tag struct {
	marshaller.CoreModel `model:"tag"`

	Ref          marshaller.Node[*string]                           `key:"$ref"`
	Name         marshaller.Node[*string]                           `key:"name"`
	Description  marshaller.Node[*string]                           `key:"description"`
	ExternalDocs marshaller.Node[*schemacore.ExternalDocumentation] `key:"externalDocs"`

	Extensions core.Extensions `key:"extensions"`
}
