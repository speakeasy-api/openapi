package swagger

import (
	"context"
	"net/url"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// ExternalDocumentation allows referencing an external resource for extended documentation.
type ExternalDocumentation struct {
	marshaller.Model[core.ExternalDocumentation]

	// Description is a short description of the target documentation. GFM syntax can be used for rich text representation.
	Description *string
	// URL is the URL for the target documentation. MUST be in the format of a URL.
	URL string
	// Extensions provides a list of extensions to the ExternalDocumentation object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.ExternalDocumentation] = (*ExternalDocumentation)(nil)

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (e *ExternalDocumentation) GetDescription() string {
	if e == nil || e.Description == nil {
		return ""
	}
	return *e.Description
}

// GetURL returns the value of the URL field. Returns empty string if not set.
func (e *ExternalDocumentation) GetURL() string {
	if e == nil {
		return ""
	}
	return e.URL
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (e *ExternalDocumentation) GetExtensions() *extensions.Extensions {
	if e == nil || e.Extensions == nil {
		return extensions.New()
	}
	return e.Extensions
}

// Validate validates the ExternalDocumentation object against the Swagger Specification.
func (e *ExternalDocumentation) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := e.GetCore()
	errs := []error{}

	if c.URL.Present && e.URL == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("externalDocs.url is required"), c, c.URL))
	}

	if c.URL.Present {
		if _, err := url.Parse(e.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("externalDocs.url is not a valid uri: %s", err), c, c.URL))
		}
	}

	e.Valid = len(errs) == 0 && c.GetValid()

	return errs
}
