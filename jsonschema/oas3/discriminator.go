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
	// This field is REQUIRED in all OpenAPI versions.
	// In OpenAPI 3.2+, the property that this references MAY be optional in the schema,
	// but when it is optional, DefaultMapping MUST be provided.
	PropertyName string
	// Mapping is an object to hold mappings between payload values and schema names or references.
	Mapping *sequencedmap.Map[string, string]
	// DefaultMapping is the schema name or URI reference to a schema that is expected to validate
	// the structure of the model when the discriminating property is not present in the payload
	// or contains a value for which there is no explicit or implicit mapping.
	// This field is part of OpenAPI 3.2+ and is required when the property referenced by
	// PropertyName is optional in the schema.
	DefaultMapping *string
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

// GetDefaultMapping returns the value of the DefaultMapping field. Returns empty string if not set.
func (d *Discriminator) GetDefaultMapping() string {
	if d == nil || d.DefaultMapping == nil {
		return ""
	}
	return *d.DefaultMapping
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

	// propertyName is REQUIRED in all OpenAPI versions
	if core.PropertyName.Present {
		if core.PropertyName.Value == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("discriminator.propertyName is required"), core, core.PropertyName))
		}
	} else {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("discriminator.propertyName is required"), core, core.PropertyName))
	}

	// defaultMapping validation - must not be empty if present
	if core.DefaultMapping.Present && (core.DefaultMapping.Value == nil || *core.DefaultMapping.Value == "") {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("discriminator.defaultMapping cannot be empty"), core, core.DefaultMapping))
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

	// Compare DefaultMapping (both pointers)
	switch {
	case d.DefaultMapping == nil && other.DefaultMapping == nil:
		// Both nil, continue
	case d.DefaultMapping == nil || other.DefaultMapping == nil:
		return false
	case *d.DefaultMapping != *other.DefaultMapping:
		return false
	}

	// Compare Mapping using sequencedmap's IsEqual method
	switch {
	case d.Mapping == nil && other.Mapping == nil:
		// Both nil, continue
	case d.Mapping == nil || other.Mapping == nil:
		return false
	case !d.Mapping.IsEqual(other.Mapping):
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
