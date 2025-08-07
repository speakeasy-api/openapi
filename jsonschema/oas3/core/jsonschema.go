package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

type JSONSchema = *values.EitherValue[Schema, bool]

type Schema struct {
	marshaller.CoreModel
	Ref                   marshaller.Node[*string]                                                `key:"$ref"`
	ExclusiveMaximum      marshaller.Node[*values.EitherValue[bool, float64]]                     `key:"exclusiveMaximum"`
	ExclusiveMinimum      marshaller.Node[*values.EitherValue[bool, float64]]                     `key:"exclusiveMinimum"`
	Type                  marshaller.Node[*values.EitherValue[[]marshaller.Node[string], string]] `key:"type"`
	AllOf                 marshaller.Node[[]JSONSchema]                                           `key:"allOf"`
	OneOf                 marshaller.Node[[]JSONSchema]                                           `key:"oneOf"`
	AnyOf                 marshaller.Node[[]JSONSchema]                                           `key:"anyOf"`
	Discriminator         marshaller.Node[*Discriminator]                                         `key:"discriminator"`
	Examples              marshaller.Node[[]marshaller.Node[values.Value]]                        `key:"examples"`
	PrefixItems           marshaller.Node[[]JSONSchema]                                           `key:"prefixItems"`
	Contains              marshaller.Node[JSONSchema]                                             `key:"contains"`
	MinContains           marshaller.Node[*int64]                                                 `key:"minContains"`
	MaxContains           marshaller.Node[*int64]                                                 `key:"maxContains"`
	If                    marshaller.Node[JSONSchema]                                             `key:"if"`
	Else                  marshaller.Node[JSONSchema]                                             `key:"else"`
	Then                  marshaller.Node[JSONSchema]                                             `key:"then"`
	DependentSchemas      marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"dependentSchemas"`
	PatternProperties     marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"patternProperties"`
	PropertyNames         marshaller.Node[JSONSchema]                                             `key:"propertyNames"`
	UnevaluatedItems      marshaller.Node[JSONSchema]                                             `key:"unevaluatedItems"`
	UnevaluatedProperties marshaller.Node[JSONSchema]                                             `key:"unevaluatedProperties"`
	Items                 marshaller.Node[JSONSchema]                                             `key:"items"`
	Anchor                marshaller.Node[*string]                                                `key:"$anchor"`
	Not                   marshaller.Node[JSONSchema]                                             `key:"not"`
	Properties            marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"properties"`
	Defs                  marshaller.Node[*sequencedmap.Map[string, JSONSchema]]                  `key:"$defs"`
	Title                 marshaller.Node[*string]                                                `key:"title"`
	MultipleOf            marshaller.Node[*float64]                                               `key:"multipleOf"`
	Maximum               marshaller.Node[*float64]                                               `key:"maximum"`
	Minimum               marshaller.Node[*float64]                                               `key:"minimum"`
	MaxLength             marshaller.Node[*int64]                                                 `key:"maxLength"`
	MinLength             marshaller.Node[*int64]                                                 `key:"minLength"`
	Pattern               marshaller.Node[*string]                                                `key:"pattern"`
	Format                marshaller.Node[*string]                                                `key:"format"`
	MaxItems              marshaller.Node[*int64]                                                 `key:"maxItems"`
	MinItems              marshaller.Node[*int64]                                                 `key:"minItems"`
	UniqueItems           marshaller.Node[*bool]                                                  `key:"uniqueItems"`
	MaxProperties         marshaller.Node[*int64]                                                 `key:"maxProperties"`
	MinProperties         marshaller.Node[*int64]                                                 `key:"minProperties"`
	Required              marshaller.Node[[]marshaller.Node[string]]                              `key:"required"`
	Enum                  marshaller.Node[[]marshaller.Node[values.Value]]                        `key:"enum"`
	AdditionalProperties  marshaller.Node[JSONSchema]                                             `key:"additionalProperties"`
	Description           marshaller.Node[*string]                                                `key:"description"`
	Default               marshaller.Node[values.Value]                                           `key:"default"`
	Const                 marshaller.Node[values.Value]                                           `key:"const"`
	Nullable              marshaller.Node[*bool]                                                  `key:"nullable"`
	ReadOnly              marshaller.Node[*bool]                                                  `key:"readOnly"`
	WriteOnly             marshaller.Node[*bool]                                                  `key:"writeOnly"`
	ExternalDocs          marshaller.Node[*ExternalDocumentation]                                 `key:"externalDocs"`
	Example               marshaller.Node[values.Value]                                           `key:"example"`
	Deprecated            marshaller.Node[*bool]                                                  `key:"deprecated"`
	Schema                marshaller.Node[*string]                                                `key:"$schema"`
	XML                   marshaller.Node[*XML]                                                   `key:"xml"`

	Extensions core.Extensions `key:"extensions"`
}
