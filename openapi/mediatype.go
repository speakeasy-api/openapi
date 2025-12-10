package openapi

import (
	"context"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// MediaTypeContext holds the media type string for validation purposes
type MediaTypeContext struct {
	MediaType string
}

// MediaType provides a schema and examples for the associated media type.
type MediaType struct {
	marshaller.Model[core.MediaType]

	// Schema is the schema defining the type used for the media type.
	Schema *oas3.JSONSchema[oas3.Referenceable]
	// ItemSchema is a schema describing each item within a sequential media type like text/event-stream.
	ItemSchema *oas3.JSONSchema[oas3.Referenceable]
	// Encoding is a map allowing for more complex encoding scenarios.
	Encoding *sequencedmap.Map[string, *Encoding]
	// PrefixEncoding provides positional encoding information for multipart content (OpenAPI 3.2+).
	// This field SHALL only apply when the media type is multipart. This field MUST NOT be present if encoding is present.
	PrefixEncoding []*Encoding
	// ItemEncoding provides encoding information for array items in multipart content (OpenAPI 3.2+).
	// This field SHALL only apply when the media type is multipart. This field MUST NOT be present if encoding is present.
	ItemEncoding *Encoding
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

// GetPrefixEncoding returns the value of the PrefixEncoding field. Returns nil if not set.
func (m *MediaType) GetPrefixEncoding() []*Encoding {
	if m == nil {
		return nil
	}
	return m.PrefixEncoding
}

// GetItemEncoding returns the value of the ItemEncoding field. Returns nil if not set.
func (m *MediaType) GetItemEncoding() *Encoding {
	if m == nil {
		return nil
	}
	return m.ItemEncoding
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

	// Validate prefixEncoding field if present
	if core.PrefixEncoding.Present {
		for _, enc := range m.PrefixEncoding {
			if enc != nil {
				errs = append(errs, enc.Validate(ctx, opts...)...)
			}
		}
	}

	// Validate itemEncoding field if present
	if core.ItemEncoding.Present {
		errs = append(errs, m.ItemEncoding.Validate(ctx, opts...)...)
	}

	// Validate mutual exclusivity: encoding MUST NOT be present with prefixEncoding or itemEncoding
	if core.Encoding.Present && (core.PrefixEncoding.Present || core.ItemEncoding.Present) {
		errs = append(errs, validation.NewValueError(
			validation.NewValueValidationError("encoding field MUST NOT be present when prefixEncoding or itemEncoding is present"),
			core,
			core.Encoding,
		))
	}

	// Validate multipart-only constraint for encoding, prefixEncoding, and itemEncoding
	o := validation.NewOptions(opts...)
	mtCtx := validation.GetContextObject[MediaTypeContext](o)
	if mtCtx != nil && mtCtx.MediaType != "" {
		isMultipart := strings.HasPrefix(strings.ToLower(mtCtx.MediaType), "multipart/")
		isFormURLEncoded := strings.ToLower(mtCtx.MediaType) == "application/x-www-form-urlencoded"

		if core.PrefixEncoding.Present && !isMultipart {
			errs = append(errs, validation.NewValueError(
				validation.NewValueValidationError("prefixEncoding field SHALL only apply when the media type is multipart"),
				core,
				core.PrefixEncoding,
			))
		}

		if core.ItemEncoding.Present && !isMultipart {
			errs = append(errs, validation.NewValueError(
				validation.NewValueValidationError("itemEncoding field SHALL only apply when the media type is multipart"),
				core,
				core.ItemEncoding,
			))
		}

		if core.Encoding.Present && !isMultipart && !isFormURLEncoded {
			errs = append(errs, validation.NewValueError(
				validation.NewValueValidationError("encoding field SHALL only apply when the media type is multipart or application/x-www-form-urlencoded"),
				core,
				core.Encoding,
			))
		}
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
