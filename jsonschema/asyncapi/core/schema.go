// Package core contains the backing models for AsyncAPI schemas, retaining the YAML/JSON nodes they were parsed from.
package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

// JSONSchema is either a Schema Object or a boolean schema.
type JSONSchema = *values.EitherValue[Schema, bool]

// Schema is the backing model for an AsyncAPI Schema Object, a superset of JSON Schema Draft 07.
// It is also used for schemas explicitly declared as JSON Schema Draft 07.
type Schema struct {
	marshaller.CoreModel `model:"schema"`

	Ref                  marshaller.Node[*string]                                                `key:"$ref"`
	ID                   marshaller.Node[*string]                                                `key:"$id"`
	Schema               marshaller.Node[*string]                                                `key:"$schema"`
	Comment              marshaller.Node[*string]                                                `key:"$comment"`
	Title                marshaller.Node[*string]                                                `key:"title"`
	Description          marshaller.Node[*string]                                                `key:"description"`
	Default              marshaller.Node[values.Value]                                           `key:"default"`
	ReadOnly             marshaller.Node[*bool]                                                  `key:"readOnly"`
	WriteOnly            marshaller.Node[*bool]                                                  `key:"writeOnly"`
	Examples             marshaller.Node[[]marshaller.Node[values.Value]]                        `key:"examples"`
	MultipleOf           marshaller.Node[*float64]                                               `key:"multipleOf"`
	Maximum              marshaller.Node[*float64]                                               `key:"maximum"`
	ExclusiveMaximum     marshaller.Node[*float64]                                               `key:"exclusiveMaximum"`
	Minimum              marshaller.Node[*float64]                                               `key:"minimum"`
	ExclusiveMinimum     marshaller.Node[*float64]                                               `key:"exclusiveMinimum"`
	MaxLength            marshaller.Node[*int64]                                                 `key:"maxLength"`
	MinLength            marshaller.Node[*int64]                                                 `key:"minLength"`
	Pattern              marshaller.Node[*string]                                                `key:"pattern"`
	AdditionalItems      marshaller.Node[JSONSchema]                                             `key:"additionalItems"`
	Items                marshaller.Node[*Items]                                                 `key:"items"`
	MaxItems             marshaller.Node[*int64]                                                 `key:"maxItems"`
	MinItems             marshaller.Node[*int64]                                                 `key:"minItems"`
	UniqueItems          marshaller.Node[*bool]                                                  `key:"uniqueItems"`
	Contains             marshaller.Node[JSONSchema]                                             `key:"contains"`
	MaxProperties        marshaller.Node[*int64]                                                 `key:"maxProperties"`
	MinProperties        marshaller.Node[*int64]                                                 `key:"minProperties"`
	Required             marshaller.Node[[]marshaller.Node[string]]                              `key:"required"`
	AdditionalProperties marshaller.Node[JSONSchema]                                             `key:"additionalProperties"`
	Definitions          marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"definitions"`
	Properties           marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"properties"`
	PatternProperties    marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"patternProperties"`
	Dependencies         marshaller.Node[*sequencedmap.Map[string, *Dependency]]                 `key:"dependencies"`
	PropertyNames        marshaller.Node[JSONSchema]                                             `key:"propertyNames"`
	Const                marshaller.Node[values.Value]                                           `key:"const"`
	Enum                 marshaller.Node[[]marshaller.Node[values.Value]]                        `key:"enum"`
	Type                 marshaller.Node[*values.EitherValue[[]marshaller.Node[string], string]] `key:"type"`
	Format               marshaller.Node[*string]                                                `key:"format"`
	ContentMediaType     marshaller.Node[*string]                                                `key:"contentMediaType"`
	ContentEncoding      marshaller.Node[*string]                                                `key:"contentEncoding"`
	If                   marshaller.Node[JSONSchema]                                             `key:"if"`
	Then                 marshaller.Node[JSONSchema]                                             `key:"then"`
	Else                 marshaller.Node[JSONSchema]                                             `key:"else"`
	AllOf                marshaller.Node[[]JSONSchema]                                           `key:"allOf"`
	AnyOf                marshaller.Node[[]JSONSchema]                                           `key:"anyOf"`
	OneOf                marshaller.Node[[]JSONSchema]                                           `key:"oneOf"`
	Not                  marshaller.Node[JSONSchema]                                             `key:"not"`
	Discriminator        marshaller.Node[*string]                                                `key:"discriminator"`
	ExternalDocs         marshaller.Node[*ExternalDocumentation]                                 `key:"externalDocs"`
	Deprecated           marshaller.Node[*bool]                                                  `key:"deprecated"`

	Extensions core.Extensions `key:"extensions"`
}

// ExternalDocumentation is the backing model for an AsyncAPI External Documentation Object (or a Reference Object to one).
type ExternalDocumentation struct {
	marshaller.CoreModel `model:"externalDocumentation"`

	Ref         marshaller.Node[*string] `key:"$ref"`
	Description marshaller.Node[*string] `key:"description"`
	URL         marshaller.Node[*string] `key:"url"`

	Extensions core.Extensions `key:"extensions"`
}
