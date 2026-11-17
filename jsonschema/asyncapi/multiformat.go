package asyncapi

import (
	"fmt"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/asyncapiversion"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// MultiFormatSchema is an AsyncAPI Multi Format Schema Object, a schema written in the language identified by SchemaFormat.
type MultiFormatSchema struct {
	marshaller.Model[core.MultiFormatSchema]

	// SchemaFormat is the media type identifying the schema language, e.g. `application/vnd.apache.avro;version=1.9.0`.
	SchemaFormat string
	// Schema is the schema definition, interpreted according to SchemaFormat.
	Schema *SchemaDefinition
	// Extensions provides a list of extensions to the Multi Format Schema Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.MultiFormatSchema] = (*MultiFormatSchema)(nil)

// NewMultiFormatSchema creates a Multi Format Schema Object for the provided format and definition.
func NewMultiFormatSchema(schemaFormat string, definition *SchemaDefinition) *MultiFormatSchema {
	return &MultiFormatSchema{
		SchemaFormat: schemaFormat,
		Schema:       definition,
	}
}

// GetSchemaFormat returns the authored schema format.
func (m *MultiFormatSchema) GetSchemaFormat() string {
	if m == nil {
		return ""
	}
	return m.SchemaFormat
}

// GetSchema returns the schema definition, or nil if not set.
func (m *MultiFormatSchema) GetSchema() *SchemaDefinition {
	if m == nil {
		return nil
	}
	return m.Schema
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (m *MultiFormatSchema) GetExtensions() *extensions.Extensions {
	if m == nil || m.Extensions == nil {
		return extensions.New()
	}
	return m.Extensions
}

// SchemaDefinition is the `schema` value of a Multi Format Schema Object.
//
// For JSON Schema based formats (the native AsyncAPI Schema Object and JSON Schema Draft 07) Schema holds
// the typed schema. For any other format Raw holds the authored value unchanged: for example a structured
// Avro schema, a Protocol Buffers string or a Reference Object to an external resource in that format.
type SchemaDefinition struct {
	// Schema is the typed schema for JSON Schema based formats.
	Schema *JSONSchema
	// Raw is the authored value for all other formats.
	Raw values.Value

	core core.SchemaDefinition
}

var _ marshaller.Populator = (*SchemaDefinition)(nil)

// NewSchemaDefinitionFromSchema creates a schema definition for a JSON Schema based format.
func NewSchemaDefinitionFromSchema(schema *JSONSchema) *SchemaDefinition {
	return &SchemaDefinition{Schema: schema}
}

// NewSchemaDefinitionFromRaw creates a schema definition holding a raw value for any other format.
func NewSchemaDefinitionFromRaw(raw *yaml.Node) *SchemaDefinition {
	return &SchemaDefinition{Raw: raw}
}

// GetCore returns the low level representation of the schema definition.
func (d *SchemaDefinition) GetCore() *core.SchemaDefinition {
	if d == nil {
		return nil
	}
	return &d.core
}

// SetCore sets the low level representation of the schema definition.
func (d *SchemaDefinition) SetCore(c *core.SchemaDefinition) {
	if d == nil || c == nil {
		return
	}
	d.core = *c
}

// GetRootNode returns the node the schema definition was parsed from.
func (d *SchemaDefinition) GetRootNode() *yaml.Node {
	if d == nil {
		return nil
	}
	return d.core.GetRootNode()
}

// GetSchema returns the typed schema, or nil if the format is not JSON Schema based.
func (d *SchemaDefinition) GetSchema() *JSONSchema {
	if d == nil {
		return nil
	}
	return d.Schema
}

// GetRaw returns the raw value, or nil if the format is JSON Schema based.
func (d *SchemaDefinition) GetRaw() values.Value {
	if d == nil {
		return nil
	}
	return d.Raw
}

// IsReference returns true if the definition is a Reference Object to a schema resource.
func (d *SchemaDefinition) IsReference() bool {
	return d.GetReference() != ""
}

// GetReference returns the reference if the definition is a Reference Object, otherwise an empty reference.
// For formats other than JSON Schema based formats only a Reference Object that is the entire value is recognised,
// references nested within foreign schema languages are not interpreted.
func (d *SchemaDefinition) GetReference() references.Reference {
	if d == nil {
		return ""
	}
	if d.Schema != nil {
		return d.Schema.GetRef()
	}
	raw := yml.ResolveAlias(d.Raw)
	if raw == nil || raw.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(raw.Content); i += 2 {
		key := yml.ResolveAlias(raw.Content[i])
		value := yml.ResolveAlias(raw.Content[i+1])
		if key != nil && key.Value == "$ref" && value != nil && value.Kind == yaml.ScalarNode {
			return references.Reference(value.Value)
		}
	}
	return ""
}

func (d *SchemaDefinition) Populate(source any) error {
	var c core.SchemaDefinition
	switch s := source.(type) {
	case core.SchemaDefinition:
		c = s
	case *core.SchemaDefinition:
		if s == nil {
			return nil
		}
		c = *s
	default:
		return fmt.Errorf("expected core.SchemaDefinition, got %T", source)
	}

	d.Schema = nil
	d.Raw = c.Raw

	if c.Schema != nil {
		d.Schema = &JSONSchema{}
		if err := marshaller.PopulateWithContext(c.Schema, d.Schema, nil); err != nil {
			return err
		}
	}

	d.core = c

	return nil
}

// SchemaUnion is a value AsyncAPI accepts wherever a schema is expected (for example message payloads and
// headers, and component schemas): a Schema Object, a boolean schema, a Reference Object or a Multi Format
// Schema Object. Exactly one of Schema and MultiFormat is set.
type SchemaUnion struct {
	// Schema is a native AsyncAPI Schema Object, boolean schema or Reference Object.
	Schema *JSONSchema
	// MultiFormat is a Multi Format Schema Object.
	MultiFormat *MultiFormatSchema

	core core.SchemaUnion
}

var _ marshaller.Populator = (*SchemaUnion)(nil)

// NewSchemaUnionFromSchema creates a schema union holding a native schema, boolean schema or Reference Object.
func NewSchemaUnionFromSchema(schema *JSONSchema) *SchemaUnion {
	return &SchemaUnion{Schema: schema}
}

// NewSchemaUnionFromMultiFormat creates a schema union holding a Multi Format Schema Object.
func NewSchemaUnionFromMultiFormat(multiFormat *MultiFormatSchema) *SchemaUnion {
	return &SchemaUnion{MultiFormat: multiFormat}
}

// GetCore returns the low level representation of the schema union.
func (u *SchemaUnion) GetCore() *core.SchemaUnion {
	if u == nil {
		return nil
	}
	return &u.core
}

// SetCore sets the low level representation of the schema union.
func (u *SchemaUnion) SetCore(c *core.SchemaUnion) {
	if u == nil || c == nil {
		return
	}
	u.core = *c
}

// GetRootNode returns the node the schema union was parsed from.
func (u *SchemaUnion) GetRootNode() *yaml.Node {
	if u == nil {
		return nil
	}
	return u.core.GetRootNode()
}

// IsSchema returns true if the union holds a native schema, boolean schema or Reference Object.
func (u *SchemaUnion) IsSchema() bool {
	return u != nil && u.Schema != nil
}

// GetSchema returns the native schema, or nil if the union holds a Multi Format Schema Object.
func (u *SchemaUnion) GetSchema() *JSONSchema {
	if u == nil {
		return nil
	}
	return u.Schema
}

// IsMultiFormat returns true if the union holds a Multi Format Schema Object.
func (u *SchemaUnion) IsMultiFormat() bool {
	return u != nil && u.MultiFormat != nil
}

// GetMultiFormat returns the Multi Format Schema Object, or nil if the union holds a native schema.
func (u *SchemaUnion) GetMultiFormat() *MultiFormatSchema {
	if u == nil {
		return nil
	}
	return u.MultiFormat
}

// IsReference returns true if the union is a Reference Object.
func (u *SchemaUnion) IsReference() bool {
	return u.GetSchema().IsReference()
}

// GetRef returns the reference if the union is a Reference Object, otherwise an empty reference.
func (u *SchemaUnion) GetRef() references.Reference {
	return u.GetSchema().GetRef()
}

// GetSchemaFormat returns the effective schema format of the union. A native schema uses the AsyncAPI
// Schema Object format of the provided document version.
func (u *SchemaUnion) GetSchemaFormat(asyncAPIVersion string) string {
	if u.IsMultiFormat() {
		return u.MultiFormat.SchemaFormat
	}
	return asyncapiversion.DefaultSchemaFormat(asyncAPIVersion)
}

func (u *SchemaUnion) Populate(source any) error {
	var c core.SchemaUnion
	switch s := source.(type) {
	case core.SchemaUnion:
		c = s
	case *core.SchemaUnion:
		if s == nil {
			return nil
		}
		c = *s
	default:
		return fmt.Errorf("expected core.SchemaUnion, got %T", source)
	}

	u.Schema = nil
	u.MultiFormat = nil

	switch {
	case c.Schema != nil:
		u.Schema = &JSONSchema{}
		if err := marshaller.PopulateWithContext(c.Schema, u.Schema, nil); err != nil {
			return err
		}
	case c.MultiFormat != nil:
		u.MultiFormat = &MultiFormatSchema{}
		if err := marshaller.PopulateWithContext(c.MultiFormat, u.MultiFormat, nil); err != nil {
			return err
		}
	}

	u.core = c

	return nil
}
