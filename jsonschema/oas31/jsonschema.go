// Package oas31 contains an implementation of the OAS v3.1 JSON Schema specification https://spec.openapis.org/oas/v3.1.0#schema-object
package oas31

import (
	_ "embed"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type JSONSchema = *EitherValue[Schema, core.Schema, bool, bool]

func NewJSONSchemaFromSchema(value *Schema) JSONSchema {
	return &EitherValue[Schema, core.Schema, bool, bool]{
		Left:  value,
		Right: nil,
	}
}

func NewJSONSchemaFromBool(value bool) JSONSchema {
	return &EitherValue[Schema, core.Schema, bool, bool]{
		Left:  nil,
		Right: pointer.From(value),
	}
}

type Schema struct {
	Ref                   *string
	ExclusiveMaximum      ExclusiveMaximum
	ExclusiveMinimum      ExclusiveMinimum
	Type                  Type
	AllOf                 []JSONSchema
	OneOf                 []JSONSchema
	AnyOf                 []JSONSchema
	Discriminator         *Discriminator
	Examples              []Value
	PrefixItems           []JSONSchema
	Contains              JSONSchema
	MinContains           *int64
	MaxContains           *int64
	If                    JSONSchema
	Else                  JSONSchema
	Then                  JSONSchema
	DependentSchemas      *sequencedmap.Map[string, JSONSchema]
	PatternProperties     *sequencedmap.Map[string, JSONSchema]
	PropertyNames         JSONSchema
	UnevaluatedItems      JSONSchema
	UnevaluatedProperties JSONSchema
	Items                 JSONSchema
	Anchor                *string
	Not                   JSONSchema
	Properties            *sequencedmap.Map[string, JSONSchema]
	Title                 *string
	MultipleOf            *float64
	Maximum               *float64
	Minimum               *float64
	MaxLength             *int64
	MinLength             *int64
	Pattern               *string
	Format                *string
	MaxItems              *int64
	MinItems              *int64
	UniqueItems           *bool
	MaxProperties         *int64
	MinProperties         *int64
	Required              []string
	Enum                  []Value
	AdditionalProperties  JSONSchema
	Description           *string
	Default               Value
	Const                 Value
	Nullable              *bool
	ReadOnly              *bool
	WriteOnly             *bool
	ExternalDocs          *ExternalDoc
	Example               Value
	Deprecated            *bool
	Schema                *string
	Extensions            *extensions.Extensions

	core core.Schema
}

func (js *Schema) GetCore() *core.Schema {
	return &js.core
}
