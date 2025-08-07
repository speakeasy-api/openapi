package oas3

import (
	"context"
	"net/url"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// XML represents the metadata of a schema describing a XML element.
type XML struct {
	marshaller.Model[core.XML]

	// Name replaces the name of the element/attribute used for the described schema property.
	Name *string
	// Namespace defines a URI of the namespace definition. Value MUST be in the form of an absolute URI.
	Namespace *string
	// Prefix to be used for the name.
	Prefix *string
	// Attribute determines whether the property definition creates an attribute.
	Attribute *bool
	// Wrapped determines whether the property definition is wrapped.
	Wrapped *bool
	// Extensions provides a list of extensions to the XML object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.XML] = (*XML)(nil)

// GetName returns the value of the Name field. Returns empty string if not set.
func (x *XML) GetName() string {
	if x == nil {
		return ""
	}
	return *x.Name
}

// GetNamespace returns the value of the Namespace field. Returns empty string if not set.
func (x *XML) GetNamespace() string {
	if x == nil {
		return ""
	}
	return *x.Namespace
}

// GetPrefix returns the value of the Prefix field. Returns empty string if not set.
func (x *XML) GetPrefix() string {
	if x == nil {
		return ""
	}
	return *x.Prefix
}

// GetAttribute returns the value of the Attribute field. Returns empty string if not set.
func (x *XML) GetAttribute() bool {
	if x == nil {
		return false
	}
	return *x.Attribute
}

// GetWrapped returns the value of the Wrapped field. Returns empty string if not set.
func (x *XML) GetWrapped() bool {
	if x == nil {
		return false
	}
	return *x.Wrapped
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (x *XML) GetExtensions() *extensions.Extensions {
	if x == nil || x.Extensions == nil {
		return extensions.New()
	}
	return x.Extensions
}

// IsEqual compares two XML instances for equality.
func (x *XML) IsEqual(other *XML) bool {
	if x == nil && other == nil {
		return true
	}
	if x == nil || other == nil {
		return false
	}

	// Compare all pointer fields using reflect.DeepEqual
	if !reflect.DeepEqual(x.Name, other.Name) {
		return false
	}
	if !reflect.DeepEqual(x.Namespace, other.Namespace) {
		return false
	}
	if !reflect.DeepEqual(x.Prefix, other.Prefix) {
		return false
	}
	if !reflect.DeepEqual(x.Attribute, other.Attribute) {
		return false
	}
	if !reflect.DeepEqual(x.Wrapped, other.Wrapped) {
		return false
	}

	// Compare Extensions
	if x.Extensions == nil && other.Extensions == nil {
		return true
	}
	if x.Extensions == nil || other.Extensions == nil {
		return false
	}
	return x.Extensions.IsEqual(other.Extensions)
}

// Validate will validate the XML object according to the OpenAPI Specification.
func (x *XML) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := x.GetCore()
	errs := []error{}

	if x.Namespace != nil {
		u, err := url.Parse(*x.Namespace)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("namespace is not a valid uri: %s", err), core, core.Namespace))
		} else if !u.IsAbs() {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("namespace must be an absolute uri: %s", *x.Namespace), core, core.Namespace))
		}
	}

	x.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
