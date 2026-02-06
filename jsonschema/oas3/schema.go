// Package oas3 contains an implementation of the OAS v3.1 JSON Schema specification https://spec.openapis.org/oas/v3.1.0#schema-object
package oas3

import (
	_ "embed"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
)

type Schema struct {
	marshaller.Model[core.Schema]

	Ref              *references.Reference
	ExclusiveMaximum ExclusiveMaximum
	ExclusiveMinimum ExclusiveMinimum
	// Type represents the type of a schema either an array of types or a single type.
	Type                  Type
	AllOf                 []*JSONSchema[Referenceable]
	OneOf                 []*JSONSchema[Referenceable]
	AnyOf                 []*JSONSchema[Referenceable]
	Discriminator         *Discriminator
	Examples              []values.Value
	PrefixItems           []*JSONSchema[Referenceable]
	Contains              *JSONSchema[Referenceable]
	MinContains           *int64
	MaxContains           *int64
	If                    *JSONSchema[Referenceable]
	Else                  *JSONSchema[Referenceable]
	Then                  *JSONSchema[Referenceable]
	DependentSchemas      *sequencedmap.Map[string, *JSONSchema[Referenceable]]
	PatternProperties     *sequencedmap.Map[string, *JSONSchema[Referenceable]]
	PropertyNames         *JSONSchema[Referenceable]
	UnevaluatedItems      *JSONSchema[Referenceable]
	UnevaluatedProperties *JSONSchema[Referenceable]
	Items                 *JSONSchema[Referenceable]
	Anchor                *string
	ID                    *string
	Not                   *JSONSchema[Referenceable]
	Properties            *sequencedmap.Map[string, *JSONSchema[Referenceable]]
	Defs                  *sequencedmap.Map[string, *JSONSchema[Referenceable]]
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
	AdditionalProperties  *JSONSchema[Referenceable]
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

	// Parent reference links - private fields to avoid serialization
	// These are set when the schema was populated as a child of another schema.
	// Used for circular reference detection during resolution.
	parent *JSONSchema[Referenceable] // Immediate parent schema in the hierarchy

	// owningDocument stores a reference to the owning document (e.g., OpenAPI document).
	// This is used to access the SchemaRegistry for $id and $anchor resolution.
	// The document must implement SchemaRegistryProvider to enable $id and $anchor lookups.
	owningDocument SchemaRegistryProvider

	// effectiveBaseURI is the computed base URI for this schema, used for resolving
	// relative $id values and $anchor references in nested schemas.
	effectiveBaseURI string
}

// ShallowCopy creates a shallow copy of the Schema.
// This copies all struct fields and creates new slices/maps but does not deep copy the referenced objects.
// The elements within slices and maps are copied by reference, not deep copied.
func (s *Schema) ShallowCopy() *Schema {
	if s == nil {
		return nil
	}

	result := &Schema{
		Model:                 s.Model,
		Ref:                   s.Ref,
		ExclusiveMaximum:      s.ExclusiveMaximum,
		ExclusiveMinimum:      s.ExclusiveMinimum,
		Type:                  s.Type,
		Discriminator:         s.Discriminator,
		Contains:              s.Contains,
		MinContains:           s.MinContains,
		MaxContains:           s.MaxContains,
		If:                    s.If,
		Else:                  s.Else,
		Then:                  s.Then,
		PropertyNames:         s.PropertyNames,
		UnevaluatedItems:      s.UnevaluatedItems,
		UnevaluatedProperties: s.UnevaluatedProperties,
		Items:                 s.Items,
		Anchor:                s.Anchor,
		ID:                    s.ID,
		Not:                   s.Not,
		Title:                 s.Title,
		MultipleOf:            s.MultipleOf,
		Maximum:               s.Maximum,
		Minimum:               s.Minimum,
		MaxLength:             s.MaxLength,
		MinLength:             s.MinLength,
		Pattern:               s.Pattern,
		Format:                s.Format,
		MaxItems:              s.MaxItems,
		MinItems:              s.MinItems,
		UniqueItems:           s.UniqueItems,
		MaxProperties:         s.MaxProperties,
		MinProperties:         s.MinProperties,
		AdditionalProperties:  s.AdditionalProperties,
		Description:           s.Description,
		Default:               s.Default,
		Const:                 s.Const,
		Nullable:              s.Nullable,
		ReadOnly:              s.ReadOnly,
		WriteOnly:             s.WriteOnly,
		ExternalDocs:          s.ExternalDocs,
		Example:               s.Example,
		Deprecated:            s.Deprecated,
		Schema:                s.Schema,
		XML:                   s.XML,
		Extensions:            s.Extensions,
		parent:                s.parent,
		owningDocument:        s.owningDocument,
		effectiveBaseURI:      s.effectiveBaseURI,
	}

	// Shallow copy slices - create new slice but reference same elements
	if s.AllOf != nil {
		result.AllOf = make([]*JSONSchema[Referenceable], len(s.AllOf))
		copy(result.AllOf, s.AllOf)
	}
	if s.OneOf != nil {
		result.OneOf = make([]*JSONSchema[Referenceable], len(s.OneOf))
		copy(result.OneOf, s.OneOf)
	}
	if s.AnyOf != nil {
		result.AnyOf = make([]*JSONSchema[Referenceable], len(s.AnyOf))
		copy(result.AnyOf, s.AnyOf)
	}
	if s.Examples != nil {
		result.Examples = make([]values.Value, len(s.Examples))
		copy(result.Examples, s.Examples)
	}
	if s.PrefixItems != nil {
		result.PrefixItems = make([]*JSONSchema[Referenceable], len(s.PrefixItems))
		copy(result.PrefixItems, s.PrefixItems)
	}
	if s.Required != nil {
		result.Required = make([]string, len(s.Required))
		copy(result.Required, s.Required)
	}
	if s.Enum != nil {
		result.Enum = make([]values.Value, len(s.Enum))
		copy(result.Enum, s.Enum)
	}

	// Shallow copy maps - create new map but reference same elements
	if s.DependentSchemas != nil {
		result.DependentSchemas = sequencedmap.From(s.DependentSchemas.All())
	}
	if s.PatternProperties != nil {
		result.PatternProperties = sequencedmap.From(s.PatternProperties.All())
	}
	if s.Properties != nil {
		result.Properties = sequencedmap.From(s.Properties.All())
	}
	if s.Defs != nil {
		result.Defs = sequencedmap.From(s.Defs.All())
	}

	return result
}

// GetRef returns the value of the Ref field. Returns empty string if not set.
func (s *Schema) GetRef() references.Reference {
	if s == nil || s.Ref == nil {
		return ""
	}
	return *s.Ref
}

// IsReference returns true if the schema is a reference (via $ref) to another schema.
func (s *Schema) IsReference() bool {
	if s == nil {
		return false
	}
	return s.Ref != nil && *s.Ref != ""
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
	if s == nil {
		return nil
	}

	if s.Type == nil {
		return []SchemaType{}
	}

	if s.Type.IsLeft() {
		return *s.Type.Left
	}

	return []SchemaType{*s.Type.Right}
}

// GetAllOf returns the value of the AllOf field. Returns nil if not set.
func (s *Schema) GetAllOf() []*JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.AllOf
}

// GetOneOf returns the value of the OneOf field. Returns nil if not set.
func (s *Schema) GetOneOf() []*JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.OneOf
}

// GetAnyOf returns the value of the AnyOf field. Returns nil if not set.
func (s *Schema) GetAnyOf() []*JSONSchema[Referenceable] {
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
func (s *Schema) GetPrefixItems() []*JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.PrefixItems
}

// GetContains returns the value of the Contains field. Returns nil if not set.
func (s *Schema) GetContains() *JSONSchema[Referenceable] {
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
func (s *Schema) GetIf() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.If
}

// GetElse returns the value of the Else field. Returns nil if not set.
func (s *Schema) GetElse() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.Else
}

// GetThen returns the value of the Then field. Returns nil if not set.
func (s *Schema) GetThen() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.Then
}

// GetDependentSchemas returns the value of the DependentSchemas field. Returns nil if not set.
func (s *Schema) GetDependentSchemas() *sequencedmap.Map[string, *JSONSchema[Referenceable]] {
	if s == nil {
		return nil
	}
	return s.DependentSchemas
}

// GetPatternProperties returns the value of the PatternProperties field. Returns nil if not set.
func (s *Schema) GetPatternProperties() *sequencedmap.Map[string, *JSONSchema[Referenceable]] {
	if s == nil {
		return nil
	}
	return s.PatternProperties
}

// GetPropertyNames returns the value of the PropertyNames field. Returns nil if not set.
func (s *Schema) GetPropertyNames() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.PropertyNames
}

// GetUnevaluatedItems returns the value of the UnevaluatedItems field. Returns nil if not set.
func (s *Schema) GetUnevaluatedItems() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.UnevaluatedItems
}

// GetUnevaluatedProperties returns the value of the UnevaluatedProperties field. Returns nil if not set.
func (s *Schema) GetUnevaluatedProperties() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.UnevaluatedProperties
}

// GetItems returns the value of the Items field. Returns nil if not set.
func (s *Schema) GetItems() *JSONSchema[Referenceable] {
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

// GetID returns the value of the ID field ($id keyword). Returns empty string if not set.
// The $id keyword identifies a schema resource with its canonical URI.
func (s *Schema) GetID() string {
	if s == nil || s.ID == nil {
		return ""
	}
	return *s.ID
}

// GetNot returns the value of the Not field. Returns nil if not set.
func (s *Schema) GetNot() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.Not
}

// GetProperties returns the value of the Properties field. Returns nil if not set.
func (s *Schema) GetProperties() *sequencedmap.Map[string, *JSONSchema[Referenceable]] {
	if s == nil {
		return nil
	}
	return s.Properties
}

// GetDefs returns the value of the Defs field. Returns nil if not set.
func (s *Schema) GetDefs() *sequencedmap.Map[string, *JSONSchema[Referenceable]] {
	if s == nil {
		return nil
	}
	return s.Defs
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

// IsReferenceOnly returns true if this schema only contains a $ref and no other properties.
// This is used for the no-ref-siblings linter rule in OAS 3.0.x (in OAS 3.1+, $ref can have siblings).
func (s *Schema) IsReferenceOnly() bool {
	if !s.IsReference() {
		return false
	}

	// Check all schema fields - if any are set, it's not reference-only
	return s.Type == nil &&
		len(s.AllOf) == 0 &&
		len(s.OneOf) == 0 &&
		len(s.AnyOf) == 0 &&
		s.Discriminator == nil &&
		len(s.Examples) == 0 &&
		len(s.PrefixItems) == 0 &&
		s.Contains == nil &&
		s.MinContains == nil &&
		s.MaxContains == nil &&
		s.If == nil &&
		s.Else == nil &&
		s.Then == nil &&
		(s.DependentSchemas == nil || s.DependentSchemas.Len() == 0) &&
		(s.PatternProperties == nil || s.PatternProperties.Len() == 0) &&
		s.PropertyNames == nil &&
		s.UnevaluatedItems == nil &&
		s.UnevaluatedProperties == nil &&
		s.Items == nil &&
		s.Anchor == nil &&
		s.ID == nil &&
		s.Not == nil &&
		(s.Properties == nil || s.Properties.Len() == 0) &&
		(s.Defs == nil || s.Defs.Len() == 0) &&
		s.Title == nil &&
		s.MultipleOf == nil &&
		s.Maximum == nil &&
		s.Minimum == nil &&
		s.MaxLength == nil &&
		s.MinLength == nil &&
		s.Pattern == nil &&
		s.Format == nil &&
		s.MaxItems == nil &&
		s.MinItems == nil &&
		s.UniqueItems == nil &&
		s.MaxProperties == nil &&
		s.MinProperties == nil &&
		len(s.Required) == 0 &&
		len(s.Enum) == 0 &&
		s.AdditionalProperties == nil &&
		s.Description == nil &&
		s.Default == nil &&
		s.Const == nil &&
		s.Nullable == nil &&
		s.ReadOnly == nil &&
		s.WriteOnly == nil &&
		s.ExternalDocs == nil &&
		s.Example == nil &&
		s.Deprecated == nil &&
		s.Schema == nil &&
		s.XML == nil &&
		(s.Extensions == nil || s.Extensions.Len() == 0)
}

// GetMaxItems returns the value of the MaxItems field. Returns nil if not set.
func (s *Schema) GetMaxItems() *int64 {
	if s == nil {
		return nil
	}
	return s.MaxItems
}

// GetMinItems returns the value of the MinItems field. Returns 0 if not set.
func (s *Schema) GetMinItems() int64 {
	if s == nil || s.MinItems == nil {
		return 0
	}
	return *s.MinItems
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
func (s *Schema) GetAdditionalProperties() *JSONSchema[Referenceable] {
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

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (s *Schema) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// IsEqual compares two Schema instances for equality.
// It performs a deep comparison of all fields, using IsEqual methods
// on custom types where available.
func (s *Schema) IsEqual(other *Schema) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	// Compare reference using reflect.DeepEqual
	if !reflect.DeepEqual(s.Ref, other.Ref) {
		return false
	}

	// Compare ExclusiveMaximum and ExclusiveMinimum (EitherValue types)
	switch {
	case s.ExclusiveMaximum == nil && other.ExclusiveMaximum == nil:
		// Both nil, continue
	case s.ExclusiveMaximum == nil || other.ExclusiveMaximum == nil:
		return false
	case !s.ExclusiveMaximum.IsEqual(other.ExclusiveMaximum):
		return false
	}

	switch {
	case s.ExclusiveMinimum == nil && other.ExclusiveMinimum == nil:
		// Both nil, continue
	case s.ExclusiveMinimum == nil || other.ExclusiveMinimum == nil:
		return false
	case !s.ExclusiveMinimum.IsEqual(other.ExclusiveMinimum):
		return false
	}

	// Compare Type (EitherValue type)
	switch {
	case s.Type == nil && other.Type == nil:
		// Both nil, continue
	case s.Type == nil || other.Type == nil:
		return false
	}
	// Compare both type arrays have the same types
	if !equalSlices(s.GetType(), other.GetType()) {
		return false
	}

	// Compare schema arrays
	if !equalJSONSchemaSlices(s.AllOf, other.AllOf) {
		return false
	}
	if !equalJSONSchemaSlices(s.OneOf, other.OneOf) {
		return false
	}
	if !equalJSONSchemaSlices(s.AnyOf, other.AnyOf) {
		return false
	}
	if !equalJSONSchemaSlices(s.PrefixItems, other.PrefixItems) {
		return false
	}

	// Compare single JSONSchema pointers
	if !equalJSONSchemas(s.Contains, other.Contains) {
		return false
	}
	if !equalJSONSchemas(s.If, other.If) {
		return false
	}
	if !equalJSONSchemas(s.Else, other.Else) {
		return false
	}
	if !equalJSONSchemas(s.Then, other.Then) {
		return false
	}
	if !equalJSONSchemas(s.Not, other.Not) {
		return false
	}
	if !equalJSONSchemas(s.PropertyNames, other.PropertyNames) {
		return false
	}
	if !equalJSONSchemas(s.UnevaluatedItems, other.UnevaluatedItems) {
		return false
	}
	if !equalJSONSchemas(s.UnevaluatedProperties, other.UnevaluatedProperties) {
		return false
	}
	if !equalJSONSchemas(s.Items, other.Items) {
		return false
	}
	if !equalJSONSchemas(s.AdditionalProperties, other.AdditionalProperties) {
		return false
	}

	// Compare sequenced maps using their IsEqualFunc method
	if !equalSequencedMaps(s.DependentSchemas, other.DependentSchemas) {
		return false
	}
	if !equalSequencedMaps(s.PatternProperties, other.PatternProperties) {
		return false
	}
	if !equalSequencedMaps(s.Properties, other.Properties) {
		return false
	}
	if !equalSequencedMaps(s.Defs, other.Defs) {
		return false
	}

	// Compare pointer fields using reflect.DeepEqual
	if !reflect.DeepEqual(s.MinContains, other.MinContains) {
		return false
	}
	if !reflect.DeepEqual(s.MaxContains, other.MaxContains) {
		return false
	}
	if !reflect.DeepEqual(s.Anchor, other.Anchor) {
		return false
	}
	if !reflect.DeepEqual(s.ID, other.ID) {
		return false
	}
	if !reflect.DeepEqual(s.Title, other.Title) {
		return false
	}
	if !reflect.DeepEqual(s.MultipleOf, other.MultipleOf) {
		return false
	}
	if !reflect.DeepEqual(s.Maximum, other.Maximum) {
		return false
	}
	if !reflect.DeepEqual(s.Minimum, other.Minimum) {
		return false
	}
	if !reflect.DeepEqual(s.MaxLength, other.MaxLength) {
		return false
	}
	if !reflect.DeepEqual(s.MinLength, other.MinLength) {
		return false
	}
	if !reflect.DeepEqual(s.Pattern, other.Pattern) {
		return false
	}
	if !reflect.DeepEqual(s.Format, other.Format) {
		return false
	}
	if !reflect.DeepEqual(s.MaxItems, other.MaxItems) {
		return false
	}
	if !reflect.DeepEqual(s.MinItems, other.MinItems) {
		return false
	}
	if !reflect.DeepEqual(s.UniqueItems, other.UniqueItems) {
		return false
	}
	if !reflect.DeepEqual(s.MaxProperties, other.MaxProperties) {
		return false
	}
	if !reflect.DeepEqual(s.MinProperties, other.MinProperties) {
		return false
	}
	if !reflect.DeepEqual(s.Description, other.Description) {
		return false
	}
	if !reflect.DeepEqual(s.Nullable, other.Nullable) {
		return false
	}
	if !reflect.DeepEqual(s.ReadOnly, other.ReadOnly) {
		return false
	}
	if !reflect.DeepEqual(s.WriteOnly, other.WriteOnly) {
		return false
	}
	if !reflect.DeepEqual(s.Deprecated, other.Deprecated) {
		return false
	}
	if !reflect.DeepEqual(s.Schema, other.Schema) {
		return false
	}

	// Compare string slices
	if !equalSlices(s.Required, other.Required) {
		return false
	}

	// Compare values.Value slices
	if !equalValueSlices(s.Examples, other.Examples) {
		return false
	}
	if !equalValueSlices(s.Enum, other.Enum) {
		return false
	}

	// Compare values.Value fields
	if !yml.EqualNodes(s.Default, other.Default) {
		return false
	}
	if !yml.EqualNodes(s.Const, other.Const) {
		return false
	}
	if !yml.EqualNodes(s.Example, other.Example) {
		return false
	}

	// Compare complex struct pointers using their IsEqual methods
	switch {
	case s.Discriminator == nil && other.Discriminator == nil:
		// Both nil, continue
	case s.Discriminator == nil || other.Discriminator == nil:
		return false
	case !s.Discriminator.IsEqual(other.Discriminator):
		return false
	}

	switch {
	case s.ExternalDocs == nil && other.ExternalDocs == nil:
		// Both nil, continue
	case s.ExternalDocs == nil || other.ExternalDocs == nil:
		return false
	case !s.ExternalDocs.IsEqual(other.ExternalDocs):
		return false
	}

	switch {
	case s.XML == nil && other.XML == nil:
		// Both nil, continue
	case s.XML == nil || other.XML == nil:
		return false
	case !s.XML.IsEqual(other.XML):
		return false
	}

	// Compare Extensions using the Extensions.IsEqual method which handles nil/empty equality
	switch {
	case s.Extensions == nil && other.Extensions == nil:
		// Both nil, continue
	case (s.Extensions == nil && other.Extensions != nil && other.Extensions.Len() > 0) ||
		(other.Extensions == nil && s.Extensions != nil && s.Extensions.Len() > 0):
		// One is nil and the other is non-empty
		return false
	case s.Extensions != nil && other.Extensions != nil:
		// Both non-nil, use IsEqual method
		if !s.Extensions.IsEqual(other.Extensions) {
			return false
		}
	}
	// If we reach here, either both are nil, or one is nil and the other is empty, or both are equal

	return true
}

// GetParent returns the immediate parent JSONSchema if this schema was populated as a child of another schema.
// Returns nil if this schema has no parent or was not populated via parent-aware population.
func (s *Schema) GetParent() *JSONSchema[Referenceable] {
	if s == nil {
		return nil
	}
	return s.parent
}

// SetParent sets the immediate parent JSONSchema for this schema.
// This is used during parent-aware population to establish parent relationships.
func (s *Schema) SetParent(parent *JSONSchema[Referenceable]) {
	if s == nil {
		return
	}
	s.parent = parent
}

// PopulateWithContext implements the ContextAwarePopulator interface for full context-aware population.
// This method receives the owning document and can register the schema with the document's registry.
func (s *Schema) PopulateWithContext(source any, ctx *marshaller.PopulationContext) error {
	// Extract parent JSONSchema if available
	if ctx != nil && ctx.Parent != nil {
		if parentSchema, ok := ctx.Parent.(*JSONSchema[Referenceable]); ok {
			s.SetParent(parentSchema)
		}
	}

	// Store owning document reference
	if ctx != nil && ctx.OwningDocument != nil {
		s.SetOwningDocument(ctx.OwningDocument)
	}

	var coreSchema *core.Schema
	switch src := source.(type) {
	case *core.Schema:
		coreSchema = src
	case core.Schema:
		coreSchema = &src
	default:
		return fmt.Errorf("expected *core.Schema or core.Schema, got %T", source)
	}

	// CRITICAL: Extract $id and $anchor from core BEFORE populating children
	// This allows us to compute effective base URI before children try to access it
	if coreSchema.ID.Present && coreSchema.ID.Value != nil {
		s.ID = coreSchema.ID.Value
	}
	if coreSchema.Anchor.Present && coreSchema.Anchor.Value != nil {
		s.Anchor = coreSchema.Anchor.Value
	}

	// CRITICAL: Compute and register effective base URI BEFORE populating children
	// This ensures parent's effective base URI is available when children try to resolve relative $ids
	s.registerWithRegistry(ctx)

	s.SetCore(coreSchema)

	// Create context for nested schemas - pass THIS Schema as parent
	// so that child JSONSchemas can store it as their enclosingSchema
	// This allows children to access our effective base URI for relative $id resolution
	nestedCtx := &marshaller.PopulationContext{
		Parent:         s, // Pass the Schema so children can access our effective base URI
		OwningDocument: s.owningDocument,
	}

	// Perform the standard population with context propagation
	if err := marshaller.PopulateModelWithContext(source, s, nestedCtx); err != nil {
		return err
	}

	return nil
}

// registerWithRegistry computes the effective base URI and registers the schema with the registry.
// This uses s.owningDocument directly (already typed as SchemaRegistryProvider) rather than
// type-asserting from context - SetOwningDocument handles that conversion.
func (s *Schema) registerWithRegistry(_ *marshaller.PopulationContext) {
	// Use s.owningDocument directly - it's already strongly typed as SchemaRegistryProvider
	// SetOwningDocument was called earlier in PopulateWithContext to handle the type assertion
	if s.owningDocument == nil {
		return
	}

	registry := s.owningDocument.GetSchemaRegistry()
	if registry == nil {
		return
	}

	// Get the containing JSONSchema from parent - this was set in PopulateWithContext
	// We need to use the actual JSONSchema from the document tree, not create a new wrapper
	containingJS := s.parent
	if containingJS == nil {
		return
	}

	// Determine parent's effective base URI by traversing the enclosingSchema chain
	// This is critical for relative $id resolution:
	// - containingJS is the JSONSchema that wraps THIS Schema
	// - containingJS.GetEnclosingSchema() is the Schema that contains containingJS as a field
	// - That enclosing Schema's effective base URI is what we need for relative $id resolution
	var parentBaseURI string
	if enclosingSchema := containingJS.GetEnclosingSchema(); enclosingSchema != nil {
		// Get the effective base URI from the enclosing schema (the parent in the document tree)
		parentBaseURI = enclosingSchema.GetEffectiveBaseURI()
	}

	// Register the actual JSONSchema from the document tree
	if err := registry.RegisterSchema(containingJS, parentBaseURI); err != nil {
		// Log error but don't fail population
		return
	}

	// Store the effective base URI from the registry
	s.effectiveBaseURI = registry.GetBaseURI(containingJS)
}

// GetOwningDocument returns the owning document (e.g., OpenAPI document) for this schema.
// Returns nil if no owning document is set.
// The owning document implements SchemaRegistryProvider to enable $id and $anchor resolution.
func (s *Schema) GetOwningDocument() SchemaRegistryProvider {
	if s == nil {
		return nil
	}
	return s.owningDocument
}

// SetOwningDocument sets the owning document reference for this schema.
// The document must implement SchemaRegistryProvider; if not, the owning document is set to nil.
func (s *Schema) SetOwningDocument(doc any) {
	if s == nil {
		return
	}
	if doc == nil {
		s.owningDocument = nil
		return
	}
	// Type assert to SchemaRegistryProvider - only store if it implements the interface
	if provider, ok := doc.(SchemaRegistryProvider); ok {
		s.owningDocument = provider
	}
	// If doc doesn't implement SchemaRegistryProvider, owningDocument remains unchanged
	// This silently ignores non-SchemaRegistryProvider types to maintain backward compatibility
}

// GetEffectiveBaseURI returns the computed base URI for this schema.
// This is the URI against which relative references should be resolved.
// If not computed, returns an empty string.
func (s *Schema) GetEffectiveBaseURI() string {
	if s == nil {
		return ""
	}
	return s.effectiveBaseURI
}

// SetEffectiveBaseURI sets the effective base URI for this schema.
func (s *Schema) SetEffectiveBaseURI(uri string) {
	if s == nil {
		return
	}
	s.effectiveBaseURI = uri
}

// GetSchemaRegistry returns the schema registry from the owning document.
// Returns nil if no owning document is set or if it doesn't provide a registry.
func (s *Schema) GetSchemaRegistry() SchemaRegistry {
	if s == nil || s.owningDocument == nil {
		return nil
	}
	return s.owningDocument.GetSchemaRegistry()
}

// Helper functions for equality comparison

func equalJSONSchemas(a, b *JSONSchema[Referenceable]) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.IsEqual(b)
}

func equalJSONSchemaSlices(a, b []*JSONSchema[Referenceable]) bool {
	// Treat nil and empty slices as equal
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i, itemA := range a {
		if !equalJSONSchemas(itemA, b[i]) {
			return false
		}
	}
	return true
}

func equalSequencedMaps(a, b *sequencedmap.Map[string, *JSONSchema[Referenceable]]) bool {
	// The sequencedmap.IsEqualFunc method now handles nil/empty equality,
	// so we can use it directly
	if a == nil && b == nil {
		return true
	}

	// Treat nil and empty maps as equal
	aLen := 0
	if a != nil {
		aLen = a.Len()
	}
	bLen := 0
	if b != nil {
		bLen = b.Len()
	}

	if aLen == 0 && bLen == 0 {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	// Use IsEqualFunc with custom comparison for JSONSchema values
	return a.IsEqualFunc(b, equalJSONSchemas)
}

func equalSlices[T any](a, b []T) bool {
	// Treat nil and empty slices as equal
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func equalValueSlices(a, b []values.Value) bool {
	// Treat nil and empty slices as equal
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i, itemA := range a {
		if !yml.EqualNodes(itemA, b[i]) {
			return false
		}
	}
	return true
}
