package openapi

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// Header represents a single header parameter.
type Header struct {
	marshaller.Model[core.Header]

	// Description is a brief description of the header. May contain CommonMark syntax.
	Description *string
	// Required determines whether this header is mandatory.
	Required *bool
	// Deprecated describes whether this header is deprecated.
	Deprecated *bool
	// Style determines the serialization style of the header.
	Style *SerializationStyle
	// Explode determines for array and object values whether separate headers should be generated for each item in the array or object.
	Explode *bool
	// Schema is the schema defining the type used for the header. Mutually exclusive with Content.
	Schema *oas3.JSONSchema[oas3.Referenceable]
	// Content represents the content type and schema of a header. Mutually exclusive with Schema.
	Content *sequencedmap.Map[string, *MediaType]
	// Example is an example of the header's value. Mutually exclusive with Examples.
	Example values.Value
	// Examples is a map of examples of the header's value. Mutually exclusive with Example.
	Examples *sequencedmap.Map[string, *ReferencedExample]
	// Extensions provides a list of extensions to the Header object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Header] = (*Header)(nil)

// GetSchema returns the value of the Schema field. Returns nil if not set.
func (h *Header) GetSchema() *oas3.JSONSchema[oas3.Referenceable] {
	if h == nil {
		return nil
	}
	return h.Schema
}

// GetRequired returns the value of the Required field. False by default if not set.
func (h *Header) GetRequired() bool {
	if h == nil || h.Required == nil {
		return false
	}
	return *h.Required
}

// GetDeprecated returns the value of the Deprecated field. False by default if not set.
func (h *Header) GetDeprecated() bool {
	if h == nil || h.Deprecated == nil {
		return false
	}
	return *h.Deprecated
}

// GetStyle returns the value of the Style field. SerializationStyleSimple by default if not set.
func (h *Header) GetStyle() SerializationStyle {
	if h == nil || h.Style == nil {
		return SerializationStyleSimple
	}
	return *h.Style
}

// GetExplode returns the value of the Explode field. False by default if not set.
func (h *Header) GetExplode() bool {
	if h == nil || h.Explode == nil {
		return false
	}
	return *h.Explode
}

// GetContent returns the value of the Content field. Returns nil if not set.
func (h *Header) GetContent() *sequencedmap.Map[string, *MediaType] {
	if h == nil {
		return nil
	}
	return h.Content
}

// GetExample returns the value of the Example field. Returns nil if not set.
func (h *Header) GetExample() values.Value {
	if h == nil {
		return nil
	}
	return h.Example
}

// GetExamples returns the value of the Examples field. Returns nil if not set.
func (h *Header) GetExamples() *sequencedmap.Map[string, *ReferencedExample] {
	if h == nil {
		return nil
	}
	return h.Examples
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (h *Header) GetExtensions() *extensions.Extensions {
	if h == nil || h.Extensions == nil {
		return extensions.New()
	}
	return h.Extensions
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (h *Header) GetDescription() string {
	if h == nil || h.Description == nil {
		return ""
	}
	return *h.Description
}

// Validate will validate the Header object against the OpenAPI Specification.
func (h *Header) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := h.GetCore()
	errs := []error{}

	if core.Style.Present {
		allowedStyles := []string{string(SerializationStyleSimple)}
		if !slices.Contains(allowedStyles, string(*h.Style)) {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("header.style must be one of [%s]", strings.Join(allowedStyles, ", ")), core, core.Style))
		}
	}

	if core.Schema.Present {
		errs = append(errs, h.Schema.Validate(ctx, opts...)...)
	}

	for mediaType, obj := range h.Content.All() {
		// Pass media type context for validation
		contentOpts := append([]validation.Option{}, opts...)
		contentOpts = append(contentOpts, validation.WithContextObject(&MediaTypeContext{MediaType: mediaType}))
		errs = append(errs, obj.Validate(ctx, contentOpts...)...)
	}

	for _, obj := range h.Examples.All() {
		errs = append(errs, obj.Validate(ctx, opts...)...)
	}

	h.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
