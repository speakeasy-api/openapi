package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

type RequestBody struct {
	marshaller.Model[core.RequestBody]

	// Description is a description of the request body. May contain CommonMark syntax.
	Description *string
	// Content is a map of content types to the schema that describes them that the operation accepts.
	Content *sequencedmap.Map[string, *MediaType]
	// Required determines whether this request body is mandatory.
	Required *bool

	// Extensions provides a list of extensions to the RequestBody object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.RequestBody] = (*RequestBody)(nil)

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (r *RequestBody) GetDescription() string {
	if r == nil || r.Description == nil {
		return ""
	}
	return *r.Description
}

// GetContent returns the value of the Content field. Returns nil if not set.
func (r *RequestBody) GetContent() *sequencedmap.Map[string, *MediaType] {
	if r == nil {
		return nil
	}
	return r.Content
}

// GetRequired returns the value of the Required field. False by default if not set.
func (r *RequestBody) GetRequired() bool {
	if r == nil || r.Required == nil {
		return false
	}
	return *r.Required
}

// Validate will validate the RequestBody object against the OpenAPI Specification.
func (r *RequestBody) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := r.GetCore()
	errs := []error{}

	if core.Content.Present && r.Content.Len() == 0 {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("requestBody field content is required"), core, core.Content))
	}

	for _, content := range r.Content.All() {
		errs = append(errs, content.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
