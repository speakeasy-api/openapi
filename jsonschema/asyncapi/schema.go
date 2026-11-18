// Package asyncapi provides models for AsyncAPI schemas: the native AsyncAPI Schema Object (a superset of
// JSON Schema Draft 07), explicit JSON Schema Draft 07 schemas, and Multi Format Schema Objects that wrap
// schemas written in other languages such as Avro or Protocol Buffers.
//
// This package is deliberately separate from jsonschema/oas3. AsyncAPI schemas follow Draft 07 rather than
// the OpenAPI dialects, use a string `discriminator`, and `$ref` follows AsyncAPI Reference Object behaviour.
package asyncapi

import (
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
)

// Type is the value of the `type` keyword: either a list of types or a single type.
type Type = values.EitherValue[[]string, []marshaller.Node[string], string, string]

// NewTypeFromString creates a `type` value with a single type.
func NewTypeFromString(typ string) *Type {
	return &Type{Right: pointer.From(typ)}
}

// NewTypeFromArray creates a `type` value with a list of types.
func NewTypeFromArray(types ...string) *Type {
	if types == nil {
		types = []string{}
	}
	return &Type{Left: &types}
}

// Schema is an AsyncAPI Schema Object, a superset of JSON Schema Draft 07. It is also used to represent
// schemas explicitly declared as JSON Schema Draft 07, in which case the AsyncAPI specific fields
// (Discriminator, ExternalDocs and Deprecated) have no special meaning.
//
// When Ref is set the schema is a Reference Object, and any sibling fields are ignored.
type Schema struct {
	marshaller.Model[core.Schema]

	Ref                  *references.Reference
	ID                   *string
	Schema               *string
	Comment              *string
	Title                *string
	Description          *string
	Default              values.Value
	ReadOnly             *bool
	WriteOnly            *bool
	Examples             []values.Value
	MultipleOf           *float64
	Maximum              *float64
	ExclusiveMaximum     *float64
	Minimum              *float64
	ExclusiveMinimum     *float64
	MaxLength            *int64
	MinLength            *int64
	Pattern              *string
	AdditionalItems      *JSONSchema
	Items                *Items
	MaxItems             *int64
	MinItems             *int64
	UniqueItems          *bool
	Contains             *JSONSchema
	MaxProperties        *int64
	MinProperties        *int64
	Required             []string
	AdditionalProperties *JSONSchema
	Definitions          *sequencedmap.Map[string, *JSONSchema]
	Properties           *sequencedmap.Map[string, *JSONSchema]
	PatternProperties    *sequencedmap.Map[string, *JSONSchema]
	Dependencies         *sequencedmap.Map[string, *Dependency]
	PropertyNames        *JSONSchema
	Const                values.Value
	Enum                 []values.Value
	Type                 *Type
	Format               *string
	ContentMediaType     *string
	ContentEncoding      *string
	If                   *JSONSchema
	Then                 *JSONSchema
	Else                 *JSONSchema
	AllOf                []*JSONSchema
	AnyOf                []*JSONSchema
	OneOf                []*JSONSchema
	Not                  *JSONSchema
	// Discriminator is the name of the property used to differentiate between schemas that inherit this schema (AsyncAPI only).
	Discriminator *string
	// ExternalDocs is additional external documentation for this schema (AsyncAPI only).
	ExternalDocs *ExternalDocumentation
	// Deprecated specifies that the schema is deprecated (AsyncAPI only).
	Deprecated *bool
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Schema] = (*Schema)(nil)

// IsReference returns true if the schema is a Reference Object.
func (s *Schema) IsReference() bool {
	return s != nil && s.Ref != nil
}

// GetRef returns the reference of the schema, or an empty reference if it is not a Reference Object.
func (s *Schema) GetRef() references.Reference {
	if s == nil || s.Ref == nil {
		return ""
	}
	return *s.Ref
}

// GetType returns the declared types of the schema, or nil if no type is declared.
func (s *Schema) GetType() []string {
	if s == nil || s.Type == nil {
		return nil
	}
	if s.Type.IsLeft() {
		return s.Type.LeftValue()
	}
	// IsLeft is only false when Right is set.
	return []string{*s.Type.Right}
}

// GetRequired returns the required property names, or nil if not set.
func (s *Schema) GetRequired() []string {
	if s == nil {
		return nil
	}
	return s.Required
}

// GetProperties returns the properties of the schema, or nil if not set.
func (s *Schema) GetProperties() *sequencedmap.Map[string, *JSONSchema] {
	if s == nil {
		return nil
	}
	return s.Properties
}

// GetDiscriminator returns the discriminator property name, or an empty string if not set.
func (s *Schema) GetDiscriminator() string {
	if s == nil || s.Discriminator == nil {
		return ""
	}
	return *s.Discriminator
}

// GetExternalDocs returns the external documentation for the schema, or nil if not set.
func (s *Schema) GetExternalDocs() *ExternalDocumentation {
	if s == nil {
		return nil
	}
	return s.ExternalDocs
}

// GetDeprecated returns true if the schema is marked as deprecated.
func (s *Schema) GetDeprecated() bool {
	if s == nil || s.Deprecated == nil {
		return false
	}
	return *s.Deprecated
}

// GetExtensions returns the extensions of the schema. Returns an empty extensions map if not set.
func (s *Schema) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// JSONSchema is either a Schema Object (which may be a Reference Object) or a boolean schema.
type JSONSchema struct {
	values.EitherValue[Schema, core.Schema, bool, bool]
}

// NewJSONSchemaFromSchema creates a JSONSchema from a Schema Object.
func NewJSONSchemaFromSchema(schema *Schema) *JSONSchema {
	return &JSONSchema{
		EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{Left: schema},
	}
}

// NewJSONSchemaFromBool creates a boolean JSONSchema, where true allows any instance and false allows none.
func NewJSONSchemaFromBool(value bool) *JSONSchema {
	return &JSONSchema{
		EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{Right: pointer.From(value)},
	}
}

// NewJSONSchemaFromReference creates a JSONSchema that is a Reference Object.
func NewJSONSchemaFromReference(ref references.Reference) *JSONSchema {
	return NewJSONSchemaFromSchema(&Schema{Ref: pointer.From(ref)})
}

// IsSchema returns true if the JSONSchema is a Schema Object (including a Reference Object).
func (j *JSONSchema) IsSchema() bool {
	return j != nil && j.IsLeft()
}

// GetSchema returns the Schema Object, or nil if the JSONSchema is a boolean schema.
func (j *JSONSchema) GetSchema() *Schema {
	if j == nil {
		return nil
	}
	return j.GetLeft()
}

// IsBool returns true if the JSONSchema is a boolean schema.
func (j *JSONSchema) IsBool() bool {
	return j != nil && j.IsRight()
}

// GetBool returns the boolean schema value, or nil if the JSONSchema is a Schema Object.
func (j *JSONSchema) GetBool() *bool {
	if j == nil {
		return nil
	}
	return j.GetRight()
}

// IsReference returns true if the JSONSchema is a Reference Object.
func (j *JSONSchema) IsReference() bool {
	return j.GetSchema().IsReference()
}

// GetRef returns the reference if the JSONSchema is a Reference Object, otherwise an empty reference.
func (j *JSONSchema) GetRef() references.Reference {
	return j.GetSchema().GetRef()
}

// ExternalDocumentation allows referencing an external resource for extended documentation, or is a Reference Object to one.
type ExternalDocumentation struct {
	marshaller.Model[core.ExternalDocumentation]

	// Ref is set when this is a Reference Object, in which case the other fields are ignored.
	Ref *references.Reference
	// Description is a description of the target documentation. May contain CommonMark syntax.
	Description *string
	// URL is the URL for the target documentation. Required unless this is a Reference Object.
	URL *string
	// Extensions provides a list of extensions to the External Documentation Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.ExternalDocumentation] = (*ExternalDocumentation)(nil)

// IsReference returns true if the external documentation is a Reference Object.
func (e *ExternalDocumentation) IsReference() bool {
	return e != nil && e.Ref != nil
}

// GetDescription returns the description, or an empty string if not set.
func (e *ExternalDocumentation) GetDescription() string {
	if e == nil || e.Description == nil {
		return ""
	}
	return *e.Description
}

// GetURL returns the URL, or an empty string if not set.
func (e *ExternalDocumentation) GetURL() string {
	if e == nil || e.URL == nil {
		return ""
	}
	return *e.URL
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (e *ExternalDocumentation) GetExtensions() *extensions.Extensions {
	if e == nil || e.Extensions == nil {
		return extensions.New()
	}
	return e.Extensions
}
