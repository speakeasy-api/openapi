package openapi

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

type Responses struct {
	marshaller.Model[core.Responses]
	sequencedmap.Map[string, *ReferencedResponse]

	// Default represents the remaining responses not declared in the map.
	Default *ReferencedResponse

	// Extensions provides a list of extensions to the Responses object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Responses] = (*Responses)(nil)

// NewResponses creates a new Responses instance with an initialized map.
func NewResponses() *Responses {
	return &Responses{
		Map: *sequencedmap.New[string, *ReferencedResponse](),
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

func (r *Responses) Populate(source any) error {
	s, ok := source.(*core.Responses)
	if !ok {
		// Handle case where source is passed by value instead of pointer
		if val, isValue := source.(core.Responses); isValue {
			s = &val
		} else {
			return fmt.Errorf("expected *core.Responses or core.Responses, got %T", source)
		}
	}

	if !r.IsInitialized() {
		r.Map = *sequencedmap.New[string, *ReferencedResponse]()
	}

	// Manually populate the map to handle type conversion from string to HTTPStatusCode
	if s.Map != nil {
		for key, value := range s.AllUntyped() {
			statusCode := key.(string)
			referencedResponse := &ReferencedResponse{}
			if err := marshaller.Populate(value, referencedResponse); err != nil {
				return err
			}
			r.Set(statusCode, referencedResponse)
		}
	}

	if s.Default.Present {
		r.Default = &ReferencedResponse{}
		if err := marshaller.Populate(s.Default.Value, r.Default); err != nil {
			return err
		}
	}

	if s.Extensions != nil {
		if r.Extensions == nil {
			r.Extensions = extensions.New()
		}
		if err := r.Extensions.Populate(s.Extensions); err != nil {
			return err
		}
	}

	r.SetCore(s)

	return nil
}

// Validate will validate the Responses object according to the OpenAPI specification.
func (r *Responses) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := r.GetCore()
	errs := []error{}

	if r.Default != nil {
		errs = append(errs, r.Default.Validate(ctx, opts...)...)
	}

	if r.Len() == 0 {
		errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("responses must have at least one response code"), core.RootNode))
	}

	for _, response := range r.All() {
		errs = append(errs, response.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// Response represents a single response from an API Operation.
type Response struct {
	marshaller.Model[core.Response]

	// Description is a description of the response. May contain CommonMark syntax.
	Description string
	// Headers is a map of headers that are sent with the response.
	Headers *sequencedmap.Map[string, *ReferencedHeader]
	// Content is a map of content types to the schema that describes them.
	Content *sequencedmap.Map[string, *MediaType]
	// Links is a map of operations links that can be followed from the response.
	Links *sequencedmap.Map[string, *ReferencedLink]

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

// GetHeaders returns the value of the Headers field. Returns nil if not set.
func (r *Response) GetHeaders() *sequencedmap.Map[string, *ReferencedHeader] {
	if r == nil {
		return nil
	}
	return r.Headers
}

// GetContent returns the value of the Content field. Returns nil if not set.
func (r *Response) GetContent() *sequencedmap.Map[string, *MediaType] {
	if r == nil {
		return nil
	}
	return r.Content
}

// GetLinks returns the value of the Links field. Returns nil if not set.
func (r *Response) GetLinks() *sequencedmap.Map[string, *ReferencedLink] {
	if r == nil {
		return nil
	}
	return r.Links
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (r *Response) GetExtensions() *extensions.Extensions {
	if r == nil || r.Extensions == nil {
		return extensions.New()
	}
	return r.Extensions
}

// Validate will validate the Response object according to the OpenAPI specification.
func (r *Response) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := r.GetCore()
	errs := []error{}

	if core.Description.Present && r.Description == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("response field description is required"), core, core.Description))
	}

	for _, header := range r.GetHeaders().All() {
		errs = append(errs, header.Validate(ctx, opts...)...)
	}

	for _, content := range r.GetContent().All() {
		errs = append(errs, content.Validate(ctx, opts...)...)
	}

	for _, link := range r.GetLinks().All() {
		errs = append(errs, link.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
