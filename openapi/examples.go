package openapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

type Example struct {
	marshaller.Model[core.Example]

	// Summary is a short summary of the example.
	Summary *string
	// Description is a description of the example.
	Description *string
	// Value is the example value. Mutually exclusive with ExternalValue, DataValue, and SerializedValue.
	// Deprecated for non-JSON serialization targets: Use DataValue and/or SerializedValue instead.
	Value values.Value
	// ExternalValue is a URI to the location of the example value. May be relative to the location of the document. Mutually exclusive with Value and SerializedValue.
	ExternalValue *string
	// DataValue is an example of the data structure that MUST be valid according to the relevant Schema Object. If this field is present, Value MUST be absent.
	DataValue values.Value
	// SerializedValue is an example of the serialized form of the value, including encoding and escaping. If this field is present, Value and ExternalValue MUST be absent.
	SerializedValue *string
	// Extensions provides a list of extensions to the Example object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Example] = (*Example)(nil)

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (e *Example) GetSummary() string {
	if e == nil || e.Summary == nil {
		return ""
	}
	return *e.Summary
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (e *Example) GetDescription() string {
	if e == nil || e.Description == nil {
		return ""
	}
	return *e.Description
}

// GetValue returns the value of the Value field. Returns nil if not set.
func (e *Example) GetValue() values.Value {
	if e == nil {
		return nil
	}
	return e.Value
}

// GetExternalValue returns the value of the ExternalValue field. Returns empty string if not set.
func (e *Example) GetExternalValue() string {
	if e == nil || e.ExternalValue == nil {
		return ""
	}
	return *e.ExternalValue
}

// GetDataValue returns the value of the DataValue field. Returns nil if not set.
func (e *Example) GetDataValue() values.Value {
	if e == nil {
		return nil
	}
	return e.DataValue
}

// GetSerializedValue returns the value of the SerializedValue field. Returns empty string if not set.
func (e *Example) GetSerializedValue() string {
	if e == nil || e.SerializedValue == nil {
		return ""
	}
	return *e.SerializedValue
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (e *Example) GetExtensions() *extensions.Extensions {
	if e == nil || e.Extensions == nil {
		return extensions.New()
	}
	return e.Extensions
}

// ResolveExternalValue will resolve the external value returning the value referenced.
func (e *Example) ResolveExternalValue(ctx context.Context) (values.Value, error) {
	// TODO implement resolving the external value
	return nil, nil
}

// Validate will validate the Example object against the OpenAPI Specification.
func (e *Example) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := e.GetCore()
	errs := []error{}

	// Check mutual exclusivity: value and externalValue
	if core.Value.Present && core.ExternalValue.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("example.value and example.externalValue are mutually exclusive"), core, core.Value))
	}

	// Check mutual exclusivity: dataValue and value
	if core.DataValue.Present && core.Value.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("example.dataValue and example.value are mutually exclusive"), core, core.DataValue))
	}

	// Check mutual exclusivity: serializedValue and value
	if core.SerializedValue.Present && core.Value.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("example.serializedValue and example.value are mutually exclusive"), core, core.SerializedValue))
	}

	// Check mutual exclusivity: serializedValue and externalValue
	if core.SerializedValue.Present && core.ExternalValue.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationMutuallyExclusiveFields, errors.New("example.serializedValue and example.externalValue are mutually exclusive"), core, core.SerializedValue))
	}

	if core.ExternalValue.Present {
		if _, err := url.Parse(*e.ExternalValue); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("example.externalValue is not a valid uri: %w", err), core, core.ExternalValue))
		}
	}

	e.Valid = len(errs) == 0 && core.GetValid()
	return errs
}
