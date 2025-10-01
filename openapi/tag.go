package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Tag represent the metadata for a single tag relating to operations in the API.
type Tag struct {
	marshaller.Model[core.Tag]

	// The name of the tag.
	Name string
	// A description for the tag. May contain CommonMark syntax.
	Description *string
	// External documentation for this tag.
	ExternalDocs *oas3.ExternalDocumentation

	// Extensions provides a list of extensions to the Tag object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Tag] = (*Tag)(nil)

// GetName returns the value of the Name field. Returns empty string if not set.
func (t *Tag) GetName() string {
	if t == nil {
		return ""
	}
	return t.Name
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (t *Tag) GetDescription() string {
	if t == nil || t.Description == nil {
		return ""
	}
	return *t.Description
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (t *Tag) GetExternalDocs() *oas3.ExternalDocumentation {
	if t == nil {
		return nil
	}
	return t.ExternalDocs
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (t *Tag) GetExtensions() *extensions.Extensions {
	if t == nil || t.Extensions == nil {
		return extensions.New()
	}
	return t.Extensions
}

// Validate will validate the Tag object against the OpenAPI Specification.
func (t *Tag) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := t.GetCore()
	errs := []error{}

	if core.Name.Present && t.Name == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("tag.name is required"), core, core.Name))
	}

	if t.ExternalDocs != nil {
		errs = append(errs, t.ExternalDocs.Validate(ctx, opts...)...)
	}

	t.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
