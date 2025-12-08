package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// MediaType provides a schema and examples for the associated media type.
type MediaType struct {
	marshaller.Model[core.MediaType]

	// Schema is the schema defining the type used for the media type.
	Schema *oas3.JSONSchema[oas3.Referenceable]
	// ItemSchema is a schema describing each item within a sequential media type like text/event-stream.
	ItemSchema *oas3.JSONSchema[oas3.Referenceable]
	// Encoding is a map allowing for more complex encoding scenarios.
	Encoding *sequencedmap.Map[string, *Encoding]
	// Example is an example of the media type's value.
	Example values.Value
	// Examples is a map of examples of the media type's value.
	Examples *sequencedmap.Map[string, *ReferencedExample]

	// Extensions provides a list of extensions to the MediaType object.
	Extensions *extensions.Extensions
}

// GetSchema returns the value of the Schema field. Returns nil if not set.
func (m *MediaType) GetSchema() *oas3.JSONSchema[oas3.Referenceable] {
	if m == nil {
		return nil
	}
	return m.Schema
}

// GetItemSchema returns the value of the ItemSchema field. Returns nil if not set.
func (m *MediaType) GetItemSchema() *oas3.JSONSchema[oas3.Referenceable] {
	if m == nil {
		return nil
	}
	return m.ItemSchema
}

// GetEncoding returns the value of the Encoding field. Returns nil if not set.
func (m *MediaType) GetEncoding() *sequencedmap.Map[string, *Encoding] {
	if m == nil {
		return nil
	}
	return m.Encoding
}

// GetExamples returns the value of the Examples field. Returns nil if not set.
func (m *MediaType) GetExamples() *sequencedmap.Map[string, *ReferencedExample] {
	if m == nil {
		return nil
	}
	return m.Examples
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (m *MediaType) GetExtensions() *extensions.Extensions {
	if m == nil || m.Extensions == nil {
		return extensions.New()
	}
	return m.Extensions
}

// Validate will validate the MediaType object against the OpenAPI Specification.
func (m *MediaType) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := m.GetCore()
	errs := []error{}

	if core.Schema.Present {
		errs = append(errs, m.Schema.Validate(ctx, opts...)...)
	}

	if core.ItemSchema.Present {
		errs = append(errs, m.ItemSchema.Validate(ctx, opts...)...)
	}

	for _, obj := range m.Examples.All() {
		errs = append(errs, obj.Validate(ctx, opts...)...)
	}

	for _, obj := range m.Encoding.All() {
		errs = append(errs, obj.Validate(ctx, opts...)...)
	}

	m.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// GetExample returns the value of the Example field. Returns nil if not set.
func (m *MediaType) GetExample() values.Value {
	if m == nil {
		return nil
	}
	return m.Example
}
