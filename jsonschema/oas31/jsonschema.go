// Package oas31 contains an implementation of the OAS v3.1 JSON Schema specification https://spec.openapis.org/oas/v3.1.0#schema-object
package oas31

import (
	_ "embed"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
)

type JSONSchema = *values.EitherValue[Schema, core.Schema, bool, bool]

func NewJSONSchemaFromSchema(value *Schema) JSONSchema {
	return &values.EitherValue[Schema, core.Schema, bool, bool]{
		Left:  value,
		Right: nil,
	}
}

func NewJSONSchemaFromBool(value bool) JSONSchema {
	return &values.EitherValue[Schema, core.Schema, bool, bool]{
		Left:  nil,
		Right: pointer.From(value),
	}
}

type Schema struct {
	marshaller.Model[core.Schema]

	Ref              *string
	ExclusiveMaximum ExclusiveMaximum
	ExclusiveMinimum ExclusiveMinimum
	// Type represents the type of a schema either an array of types or a single type.
	Type                  Type
	AllOf                 []JSONSchema
	OneOf                 []JSONSchema
	AnyOf                 []JSONSchema
	Discriminator         *Discriminator
	Examples              []values.Value
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
	Enum                  []values.Value
	AdditionalProperties  JSONSchema
	Description           *string
	Default               values.Value
	Const                 values.Value
	Nullable              *bool
	ReadOnly              *bool
	WriteOnly             *bool
	ExternalDocs          *ExternalDocumentation
	Example               values.Value
	Deprecated            *bool
	Schema                *string
	XML                   *XML
	Extensions            *extensions.Extensions
}

// GetRef returns the value of the Ref field. Returns empty string if not set.
func (s *Schema) GetRef() string {
	if s == nil || s.Ref == nil {
		return ""
	}
	return *s.Ref
}

// GetExclusiveMaximum returns the value of the ExclusiveMaximum field. Returns nil if not set.
func (s *Schema) GetExclusiveMaximum() ExclusiveMaximum {
	if s == nil {
		return nil
	}
	return s.ExclusiveMaximum
}

// GetExclusiveMinimum returns the value of the ExclusiveMinimum field. Returns nil if not set.
func (s *Schema) GetExclusiveMinimum() ExclusiveMinimum {
	if s == nil {
		return nil
	}
	return s.ExclusiveMinimum
}

// GetType will resolve the type of the schema to an array of the types represented by this schema.
func (s *Schema) GetType() []SchemaType {
	if s.Type == nil {
		return []SchemaType{}
	}

	if s.Type.IsLeft() {
		return *s.Type.Left
	}

	return []SchemaType{*s.Type.Right}
}

// GetAllOf returns the value of the AllOf field. Returns nil if not set.
func (s *Schema) GetAllOf() []JSONSchema {
	if s == nil {
		return nil
	}
	return s.AllOf
}

// GetOneOf returns the value of the OneOf field. Returns nil if not set.
func (s *Schema) GetOneOf() []JSONSchema {
	if s == nil {
		return nil
	}
	return s.OneOf
}

// GetAnyOf returns the value of the AnyOf field. Returns nil if not set.
func (s *Schema) GetAnyOf() []JSONSchema {
	if s == nil {
		return nil
	}
	return s.AnyOf
}

// GetDiscriminator returns the value of the Discriminator field. Returns nil if not set.
func (s *Schema) GetDiscriminator() *Discriminator {
	if s == nil {
		return nil
	}
	return s.Discriminator
}

// GetExamples returns the value of the Examples field. Returns nil if not set.
func (s *Schema) GetExamples() []values.Value {
	if s == nil {
		return nil
	}
	return s.Examples
}

// GetPrefixItems returns the value of the PrefixItems field. Returns nil if not set.
func (s *Schema) GetPrefixItems() []JSONSchema {
	if s == nil {
		return nil
	}
	return s.PrefixItems
}

// GetContains returns the value of the Contains field. Returns nil if not set.
func (s *Schema) GetContains() JSONSchema {
	if s == nil {
		return nil
	}
	return s.Contains
}

// GetMinContains returns the value of the MinContains field. Returns nil if not set.
func (s *Schema) GetMinContains() *int64 {
	if s == nil {
		return nil
	}
	return s.MinContains
}

// GetMaxContains returns the value of the MaxContains field. Returns nil if not set.
func (s *Schema) GetMaxContains() *int64 {
	if s == nil {
		return nil
	}
	return s.MaxContains
}

// GetIf returns the value of the If field. Returns nil if not set.
func (s *Schema) GetIf() JSONSchema {
	if s == nil {
		return nil
	}
	return s.If
}

// GetElse returns the value of the Else field. Returns nil if not set.
func (s *Schema) GetElse() JSONSchema {
	if s == nil {
		return nil
	}
	return s.Else
}

// GetThen returns the value of the Then field. Returns nil if not set.
func (s *Schema) GetThen() JSONSchema {
	if s == nil {
		return nil
	}
	return s.Then
}

// GetDependentSchemas returns the value of the DependentSchemas field. Returns nil if not set.
func (s *Schema) GetDependentSchemas() *sequencedmap.Map[string, JSONSchema] {
	if s == nil {
		return nil
	}
	return s.DependentSchemas
}

// GetPatternProperties returns the value of the PatternProperties field. Returns nil if not set.
func (s *Schema) GetPatternProperties() *sequencedmap.Map[string, JSONSchema] {
	if s == nil {
		return nil
	}
	return s.PatternProperties
}

// GetPropertyNames returns the value of the PropertyNames field. Returns nil if not set.
func (s *Schema) GetPropertyNames() JSONSchema {
	if s == nil {
		return nil
	}
	return s.PropertyNames
}

// GetUnevaluatedItems returns the value of the UnevaluatedItems field. Returns nil if not set.
func (s *Schema) GetUnevaluatedItems() JSONSchema {
	if s == nil {
		return nil
	}
	return s.UnevaluatedItems
}

// GetUnevaluatedProperties returns the value of the UnevaluatedProperties field. Returns nil if not set.
func (s *Schema) GetUnevaluatedProperties() JSONSchema {
	if s == nil {
		return nil
	}
	return s.UnevaluatedProperties
}

// GetItems returns the value of the Items field. Returns nil if not set.
func (s *Schema) GetItems() JSONSchema {
	if s == nil {
		return nil
	}
	return s.Items
}

// GetAnchor returns the value of the Anchor field. Returns empty string if not set.
func (s *Schema) GetAnchor() string {
	if s == nil || s.Anchor == nil {
		return ""
	}
	return *s.Anchor
}

// GetNot returns the value of the Not field. Returns nil if not set.
func (s *Schema) GetNot() JSONSchema {
	if s == nil {
		return nil
	}
	return s.Not
}

// GetProperties returns the value of the Properties field. Returns nil if not set.
func (s *Schema) GetProperties() *sequencedmap.Map[string, JSONSchema] {
	if s == nil {
		return nil
	}
	return s.Properties
}

// GetTitle returns the value of the Title field. Returns empty string if not set.
func (s *Schema) GetTitle() string {
	if s == nil || s.Title == nil {
		return ""
	}
	return *s.Title
}

// GetMultipleOf returns the value of the MultipleOf field. Returns nil if not set.
func (s *Schema) GetMultipleOf() *float64 {
	if s == nil {
		return nil
	}
	return s.MultipleOf
}

// GetMaximum returns the value of the Maximum field. Returns nil if not set.
func (s *Schema) GetMaximum() *float64 {
	if s == nil {
		return nil
	}
	return s.Maximum
}

// GetMinimum returns the value of the Minimum field. Returns nil if not set.
func (s *Schema) GetMinimum() *float64 {
	if s == nil {
		return nil
	}
	return s.Minimum
}

// GetMaxLength returns the value of the MaxLength field. Returns nil if not set.
func (s *Schema) GetMaxLength() *int64 {
	if s == nil {
		return nil
	}
	return s.MaxLength
}

// GetMinLength returns the value of the MinLength field. Returns nil if not set.
func (s *Schema) GetMinLength() *int64 {
	if s == nil {
		return nil
	}
	return s.MinLength
}

// GetPattern returns the value of the Pattern field. Returns empty string if not set.
func (s *Schema) GetPattern() string {
	if s == nil || s.Pattern == nil {
		return ""
	}
	return *s.Pattern
}

// GetFormat returns the value of the Format field. Returns empty string if not set.
func (s *Schema) GetFormat() string {
	if s == nil || s.Format == nil {
		return ""
	}
	return *s.Format
}

// GetMaxItems returns the value of the MaxItems field. Returns nil if not set.
func (s *Schema) GetMaxItems() *int64 {
	if s == nil {
		return nil
	}
	return s.MaxItems
}

// GetMinItems returns the value of the MinItems field. Returns nil if not set.
func (s *Schema) GetMinItems() *int64 {
	if s == nil {
		return nil
	}
	return s.MinItems
}

// GetUniqueItems returns the value of the UniqueItems field. Returns false if not set.
func (s *Schema) GetUniqueItems() bool {
	if s == nil || s.UniqueItems == nil {
		return false
	}
	return *s.UniqueItems
}

// GetMaxProperties returns the value of the MaxProperties field. Returns nil if not set.
func (s *Schema) GetMaxProperties() *int64 {
	if s == nil {
		return nil
	}
	return s.MaxProperties
}

// GetMinProperties returns the value of the MinProperties field. Returns nil if not set.
func (s *Schema) GetMinProperties() *int64 {
	if s == nil {
		return nil
	}
	return s.MinProperties
}

// GetRequired returns the value of the Required field. Returns nil if not set.
func (s *Schema) GetRequired() []string {
	if s == nil {
		return nil
	}
	return s.Required
}

// GetEnum returns the value of the Enum field. Returns nil if not set.
func (s *Schema) GetEnum() []values.Value {
	if s == nil {
		return nil
	}
	return s.Enum
}

// GetAdditionalProperties returns the value of the AdditionalProperties field. Returns nil if not set.
func (s *Schema) GetAdditionalProperties() JSONSchema {
	if s == nil {
		return nil
	}
	return s.AdditionalProperties
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (s *Schema) GetDescription() string {
	if s == nil || s.Description == nil {
		return ""
	}
	return *s.Description
}

// GetDefault returns the value of the Default field. Returns nil if not set.
func (s *Schema) GetDefault() values.Value {
	if s == nil {
		return nil
	}
	return s.Default
}

// GetConst returns the value of the Const field. Returns nil if not set.
func (s *Schema) GetConst() values.Value {
	if s == nil {
		return nil
	}
	return s.Const
}

// GetNullable returns the value of the Nullable field. Returns false if not set.
func (s *Schema) GetNullable() bool {
	if s == nil || s.Nullable == nil {
		return false
	}
	return *s.Nullable
}

// GetReadOnly returns the value of the ReadOnly field. Returns false if not set.
func (s *Schema) GetReadOnly() bool {
	if s == nil || s.ReadOnly == nil {
		return false
	}
	return *s.ReadOnly
}

// GetWriteOnly returns the value of the WriteOnly field. Returns false if not set.
func (s *Schema) GetWriteOnly() bool {
	if s == nil || s.WriteOnly == nil {
		return false
	}
	return *s.WriteOnly
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (s *Schema) GetExternalDocs() *ExternalDocumentation {
	if s == nil {
		return nil
	}
	return s.ExternalDocs
}

// GetExample returns the value of the Example field. Returns nil if not set.
func (s *Schema) GetExample() values.Value {
	if s == nil {
		return nil
	}
	return s.Example
}

// GetDeprecated returns the value of the Deprecated field. Returns false if not set.
func (s *Schema) GetDeprecated() bool {
	if s == nil || s.Deprecated == nil {
		return false
	}
	return *s.Deprecated
}

// GetSchema returns the value of the Schema field. Returns empty string if not set.
func (s *Schema) GetSchema() string {
	if s == nil || s.Schema == nil {
		return ""
	}
	return *s.Schema
}

// GetXML returns the value of the XML field. Returns nil if not set.
func (s *Schema) GetXML() *XML {
	if s == nil {
		return nil
	}
	return s.XML
}

// GetExtensions returns the value of the Extensions field. Returns nil if not set.
func (s *Schema) GetExtensions() *extensions.Extensions {
	if s == nil {
		return nil
	}
	return s.Extensions
}
