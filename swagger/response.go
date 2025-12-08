package swagger

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// Responses is a container for the expected responses of an operation.
type Responses struct {
	marshaller.Model[core.Responses]
	*sequencedmap.Map[string, *ReferencedResponse]

	// Default is the documentation of responses other than the ones declared for specific HTTP response codes.
	Default *ReferencedResponse
	// Extensions provides a list of extensions to the Responses object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Responses] = (*Responses)(nil)

// NewResponses creates a new Responses object with an initialized map.
func NewResponses() *Responses {
	return &Responses{
		Map: sequencedmap.New[string, *ReferencedResponse](),
	}
}

// GetDefault returns the value of the Default field. Returns nil if not set.
func (r *Responses) GetDefault() *ReferencedResponse {
	if r == nil {
		return nil
	}
	return r.Default
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (r *Responses) GetExtensions() *extensions.Extensions {
	if r == nil || r.Extensions == nil {
		return extensions.New()
	}
	return r.Extensions
}

// Validate validates the Responses object against the Swagger Specification.
func (r *Responses) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := r.GetCore()
	errs := []error{}

	// Responses object must contain at least one response code
	hasResponse := (c.Default.Present && r.Default != nil) || (r.Map != nil && r.Len() > 0)
	if !hasResponse {
		errs = append(errs, validation.NewValueError(
			validation.NewMissingValueError("responses must contain at least one response code or default"),
			c, c.Default))
	}

	if c.Default.Present && r.Default != nil {
		errs = append(errs, r.Default.Validate(ctx, opts...)...)
	}

	for _, response := range r.All() {
		errs = append(errs, response.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && c.GetValid()

	return errs
}

// Response describes a single response from an API operation.
type Response struct {
	marshaller.Model[core.Response]

	// Description is a short description of the response.
	Description string
	// Schema is a definition of the response structure.
	Schema *oas3.JSONSchema[oas3.Referenceable]
	// Headers is a list of headers that are sent with the response.
	Headers *sequencedmap.Map[string, *Header]
	// Examples is an example of the response message.
	Examples *sequencedmap.Map[string, values.Value]
	// Extensions provides a list of extensions to the Response object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Response] = (*Response)(nil)

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (r *Response) GetDescription() string {
	if r == nil {
		return ""
	}
	return r.Description
}

// GetSchema returns the value of the Schema field. Returns nil if not set.
func (r *Response) GetSchema() *oas3.JSONSchema[oas3.Referenceable] {
	if r == nil {
		return nil
	}
	return r.Schema
}

// GetHeaders returns the value of the Headers field. Returns nil if not set.
func (r *Response) GetHeaders() *sequencedmap.Map[string, *Header] {
	if r == nil {
		return nil
	}
	return r.Headers
}

// GetExamples returns the value of the Examples field. Returns nil if not set.
func (r *Response) GetExamples() *sequencedmap.Map[string, values.Value] {
	if r == nil {
		return nil
	}
	return r.Examples
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (r *Response) GetExtensions() *extensions.Extensions {
	if r == nil || r.Extensions == nil {
		return extensions.New()
	}
	return r.Extensions
}

// Validate validates the Response object against the Swagger Specification.
func (r *Response) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := r.GetCore()
	errs := []error{}

	if c.Description.Present && r.Description == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("response.description is required"), c, c.Description))
	}

	for _, header := range r.Headers.All() {
		errs = append(errs, header.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && c.GetValid()

	return errs
}

// Header describes a single header in a response.
type Header struct {
	marshaller.Model[core.Header]

	// Description is a short description of the header.
	Description *string
	// Type is the type of the object.
	Type string
	// Format is the extending format for the type.
	Format *string
	// Items describes the type of items in the array (if type is array).
	Items *Items
	// CollectionFormat determines the format of the array.
	CollectionFormat *CollectionFormat
	// Default declares the value the server will use if none is provided.
	Default values.Value
	// Maximum specifies the maximum value.
	Maximum *float64
	// ExclusiveMaximum specifies if maximum is exclusive.
	ExclusiveMaximum *bool
	// Minimum specifies the minimum value.
	Minimum *float64
	// ExclusiveMinimum specifies if minimum is exclusive.
	ExclusiveMinimum *bool
	// MaxLength specifies the maximum length.
	MaxLength *int64
	// MinLength specifies the minimum length.
	MinLength *int64
	// Pattern specifies a regex pattern the string must match.
	Pattern *string
	// MaxItems specifies the maximum number of items in an array.
	MaxItems *int64
	// MinItems specifies the minimum number of items in an array.
	MinItems *int64
	// UniqueItems specifies if all items must be unique.
	UniqueItems *bool
	// Enum specifies a list of allowed values.
	Enum []values.Value
	// MultipleOf specifies the value must be a multiple of this number.
	MultipleOf *float64

	// Extensions provides a list of extensions to the Header object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Header] = (*Header)(nil)

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (h *Header) GetDescription() string {
	if h == nil || h.Description == nil {
		return ""
	}
	return *h.Description
}

// GetType returns the value of the Type field. Returns empty string if not set.
func (h *Header) GetType() string {
	if h == nil {
		return ""
	}
	return h.Type
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (h *Header) GetExtensions() *extensions.Extensions {
	if h == nil || h.Extensions == nil {
		return extensions.New()
	}
	return h.Extensions
}

// Validate validates the Header object against the Swagger Specification.
func (h *Header) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := h.GetCore()
	errs := []error{}

	if c.Type.Present && h.Type == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("header.type is required"), c, c.Type))
	} else if c.Type.Present {
		validTypes := []string{"string", "number", "integer", "boolean", "array"}
		valid := false
		for _, t := range validTypes {
			if h.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("header.type must be one of [string, number, integer, boolean, array]"), c, c.Type))
		}

		// Array type requires items
		if h.Type == "array" && !c.Items.Present {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("header.items is required when type=array"), c, c.Items))
		}
	}

	// Validate items if present
	if c.Items.Present && h.Items != nil {
		errs = append(errs, h.Items.Validate(ctx, opts...)...)
	}

	h.Valid = len(errs) == 0 && c.GetValid()

	return errs
}
