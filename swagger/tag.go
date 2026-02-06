package swagger

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Tag allows adding metadata to a single tag that is used by operations.
type Tag struct {
	marshaller.Model[core.Tag]

	// Name is the name of the tag.
	Name string
	// Description is a short description for the tag. GFM syntax can be used for rich text representation.
	Description *string
	// ExternalDocs is additional external documentation for this tag.
	ExternalDocs *ExternalDocumentation
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
func (t *Tag) GetExternalDocs() *ExternalDocumentation {
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

// Validate validates the Tag object against the Swagger Specification.
func (t *Tag) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := t.GetCore()
	errs := []error{}

	if c.Name.Present && t.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`tag.name` is required"), c, c.Name))
	}

	if c.ExternalDocs.Present {
		errs = append(errs, t.ExternalDocs.Validate(ctx, opts...)...)
	}

	t.Valid = len(errs) == 0 && c.GetValid()

	return errs
}
