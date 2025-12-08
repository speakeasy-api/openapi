package openapi

import (
	"context"
	"fmt"
	"mime"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Encoding represents a single encoding definition applied to a single schema property.
type Encoding struct {
	marshaller.Model[core.Encoding]

	// ContentType is a string that describes the media type of the encoding. Can be a specific media type (e.g. application/json), a wildcard media type (e.g. image/*) or a comma-separated list of the two types.
	ContentType *string
	// Headers represents additional headers that can be added to the request.
	Headers *sequencedmap.Map[string, *ReferencedHeader]
	// Style describes how the property is serialized based on its type.
	Style *SerializationStyle
	// Explode determines for array or object properties whether separate parameters should be generated for each item in the array or object.
	Explode *bool
	// AllowReserved determines if the value of this parameter can contain reserved characters as defined by RFC3986.
	AllowReserved *bool

	// Extensions provides a list of extensions to the Encoding object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Encoding] = (*Encoding)(nil)

// GetContentType will return the value of the content type or the default based on the schema of the associated property.
// schema can either be the schema of the property or nil, if nil the provided content type will be used or the default "application/octet-stream" will be used.
func (e *Encoding) GetContentType(schema *oas3.JSONSchema[oas3.Concrete]) string {
	if e == nil || e.ContentType == nil {
		if schema == nil || schema.IsRight() {
			return "application/octet-stream"
		}

		types := schema.GetLeft().GetType()
		if len(types) == 1 {
			switch types[0] {
			case oas3.SchemaTypeObject:
				return "application/json"
			case oas3.SchemaTypeArray:
				if schema.GetLeft().Items.IsResolved() {
					return e.GetContentType(schema.Left.Items.GetResolvedSchema())
				}
			default:
				break
			}
		}

		return "application/octet-stream"
	}

	return *e.ContentType
}

// GetContentTypeValue returns the raw value of the ContentType field. Returns empty string if not set.
func (e *Encoding) GetContentTypeValue() string {
	if e == nil || e.ContentType == nil {
		return ""
	}
	return *e.ContentType
}

// GetStyle will return the value of the style or the default SerializationStyleForm.
func (e *Encoding) GetStyle() SerializationStyle {
	if e == nil || e.Style == nil {
		return SerializationStyleForm
	}

	return *e.Style
}

// GetExplode will return the value of the explode or the default based on the style.
func (e *Encoding) GetExplode() bool {
	if e == nil || e.Explode == nil {
		return e.GetStyle() == SerializationStyleForm
	}
	return *e.Explode
}

// GetAllowReserved will return the value of the allowReserved or the default false.
func (e *Encoding) GetAllowReserved() bool {
	if e == nil || e.AllowReserved == nil {
		return false
	}
	return *e.AllowReserved
}

// GetHeaders returns the value of the Headers field. Returns nil if not set.
func (e *Encoding) GetHeaders() *sequencedmap.Map[string, *ReferencedHeader] {
	if e == nil {
		return nil
	}
	return e.Headers
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (e *Encoding) GetExtensions() *extensions.Extensions {
	if e == nil || e.Extensions == nil {
		return extensions.New()
	}
	return e.Extensions
}

// Validate will validate the Encoding object against the OpenAPI Specification.
func (e *Encoding) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := e.GetCore()
	errs := []error{}

	if core.ContentType.Present {
		mediaTypes := []string{*e.ContentType}
		if strings.Contains(*e.ContentType, ",") {
			mediaTypes = strings.Split(*e.ContentType, ",")
		}

		for _, mediaType := range mediaTypes {
			_, _, err := mime.ParseMediaType(mediaType)
			if err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError(fmt.Sprintf("encoding.contentType %s is not a valid media type: %s", mediaType, err)), core, core.ContentType))
			}
		}
	}

	for _, header := range e.Headers.All() {
		errs = append(errs, header.Validate(ctx, opts...)...)
	}

	if core.Style.Present {
		allowedStyles := []string{string(SerializationStyleForm), string(SerializationStyleSpaceDelimited), string(SerializationStylePipeDelimited), string(SerializationStyleDeepObject)}
		if !slices.Contains(allowedStyles, string(*e.Style)) {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError(fmt.Sprintf("encoding.style must be one of [%s]", strings.Join(allowedStyles, ", "))), core, core.Style))
		}
	}

	e.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
