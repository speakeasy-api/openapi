package oas3

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// ExternalDocumentation allows referencing external documentation for the associated object.
type ExternalDocumentation struct {
	marshaller.Model[core.ExternalDocumentation]

	// Description is a description of the target documentation. May contain CommonMark syntax.
	Description *string
	// URL is the URL for the target documentation.
	URL string
	// Extensions provides a list of extensions to the ExternalDocumentation object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.ExternalDocumentation] = (*ExternalDocumentation)(nil)

// GetDescription returns the value of the Description field. Returns an empty string if not set.
func (e *ExternalDocumentation) GetDescription() string {
	if e == nil || e.Description == nil {
		return ""
	}
	return *e.Description
}

// GetURL returns the value of the URL field. Returns an empty string if not set.
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

// IsEqual compares two ExternalDocumentation instances for equality.
func (e *ExternalDocumentation) IsEqual(other *ExternalDocumentation) bool {
	if e == nil && other == nil {
		return true
	}
	if e == nil || other == nil {
		return false
	}

	// Compare Description
	if !equalPtrs(e.Description, other.Description) {
		return false
	}

	// Compare URL
	if e.URL != other.URL {
		return false
	}

	// Compare Extensions
	if e.Extensions == nil && other.Extensions == nil {
		return true
	}
	if e.Extensions == nil || other.Extensions == nil {
		return false
	}
	return e.Extensions.IsEqual(other.Extensions)
}

// Validate will validate the ExternalDocumentation object according to the OpenAPI Specification.
func (e *ExternalDocumentation) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := e.GetCore()
	errs := []error{}

	if core.URL.Present {
		if core.URL.Value == "" {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`externalDocumentation.url` is required"), core, core.URL))
		} else {
			if _, err := url.Parse(core.URL.Value); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityWarning, validation.RuleValidationInvalidFormat, fmt.Errorf("externalDocumentation.url is not a valid uri: %w", err), core, core.URL))
			}
		}
	}

	e.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
