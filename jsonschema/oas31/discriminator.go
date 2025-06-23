package oas31

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Discriminator is used to aid in serialization, deserialization, and validation of the oneOf, anyOf and allOf schemas.
type Discriminator struct {
	marshaller.Model[core.Discriminator]

	// PropertyName is the name of the property in the payload that will hold the discriminator value.
	PropertyName string
	// Mapping is an object to hold mappings between payload values and schema names or references.
	Mapping *sequencedmap.Map[string, string]
	// Extensions provides a list of extensions to the Discriminator object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Discriminator] = (*Discriminator)(nil)

// GetPropertyName returns the value of the PropertyName field. Returns empty string if not set.
func (d *Discriminator) GetPropertyName() string {
	if d == nil {
		return ""
	}
	return d.PropertyName
}

// GetMapping returns the value of the Mapping field. Returns nil if not set.
func (d *Discriminator) GetMapping() *sequencedmap.Map[string, string] {
	if d == nil {
		return nil
	}
	return d.Mapping
}

// GetExtensions returns the value of the Extensions field. Returns nil if not set.
func (d *Discriminator) GetExtensions() *extensions.Extensions {
	if d == nil {
		return nil
	}
	return d.Extensions
}

// Validate will validate the Discriminator object according to the OpenAPI Specification.
func (d *Discriminator) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := d.GetCore()
	errs := []error{}

	if core.PropertyName.Present {
		if core.PropertyName.Value == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("propertyName is required"), core, core.PropertyName))
		}
	}

	d.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
