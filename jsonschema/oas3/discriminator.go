package oas3

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
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

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (d *Discriminator) GetExtensions() *extensions.Extensions {
	if d == nil || d.Extensions == nil {
		return extensions.New()
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

// IsEqual compares two Discriminator instances for equality.
func (d *Discriminator) IsEqual(other *Discriminator) bool {
	if d == nil && other == nil {
		return true
	}
	if d == nil || other == nil {
		return false
	}

	// Compare PropertyName
	if d.PropertyName != other.PropertyName {
		return false
	}

	// Compare Mapping using sequencedmap's IsEqual method
	if d.Mapping == nil && other.Mapping == nil {
		// Both nil, continue
	} else if d.Mapping == nil || other.Mapping == nil {
		return false
	} else if !d.Mapping.IsEqual(other.Mapping) {
		return false
	}

	// Compare Extensions
	if d.Extensions == nil && other.Extensions == nil {
		return true
	}
	if d.Extensions == nil || other.Extensions == nil {
		return false
	}
	return d.Extensions.IsEqual(other.Extensions)
}
