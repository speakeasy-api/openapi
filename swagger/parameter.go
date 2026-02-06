package swagger

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// ParameterIn represents the location of a parameter.
type ParameterIn string

const (
	// ParameterInQuery represents a query parameter.
	ParameterInQuery ParameterIn = "query"
	// ParameterInHeader represents a header parameter.
	ParameterInHeader ParameterIn = "header"
	// ParameterInPath represents a path parameter.
	ParameterInPath ParameterIn = "path"
	// ParameterInFormData represents a form data parameter.
	ParameterInFormData ParameterIn = "formData"
	// ParameterInBody represents a body parameter.
	ParameterInBody ParameterIn = "body"
)

// CollectionFormat represents how array parameters are serialized.
type CollectionFormat string

const (
	// CollectionFormatCSV represents comma-separated values.
	CollectionFormatCSV CollectionFormat = "csv"
	// CollectionFormatSSV represents space-separated values.
	CollectionFormatSSV CollectionFormat = "ssv"
	// CollectionFormatTSV represents tab-separated values.
	CollectionFormatTSV CollectionFormat = "tsv"
	// CollectionFormatPipes represents pipe-separated values.
	CollectionFormatPipes CollectionFormat = "pipes"
	// CollectionFormatMulti represents multiple parameter instances.
	CollectionFormatMulti CollectionFormat = "multi"
)

// Parameter describes a single operation parameter.
type Parameter struct {
	marshaller.Model[core.Parameter]

	// Name is the name of the parameter.
	Name string
	// In is the location of the parameter.
	In ParameterIn
	// Description is a brief description of the parameter.
	Description *string
	// Required determines whether this parameter is mandatory.
	Required *bool

	// For body parameters
	// Schema is the schema defining the type used for the body parameter.
	Schema *oas3.JSONSchema[oas3.Referenceable]

	// For non-body parameters
	// Type is the type of the parameter.
	Type *string
	// Format is the extending format for the type.
	Format *string
	// AllowEmptyValue sets the ability to pass empty-valued parameters (query or formData only).
	AllowEmptyValue *bool
	// Items describes the type of items in the array (if type is array).
	Items *Items
	// CollectionFormat determines the format of the array.
	CollectionFormat *CollectionFormat
	// Default declares the value the server will use if none is provided.
	Default values.Value
	// Maximum specifies the maximum value.
	Maximum *float64
	// ExclusiveMaximum specifies if maximum is exclusive.
	ExclusiveMaximum *bool
	// Minimum specifies the minimum value.
	Minimum *float64
	// ExclusiveMinimum specifies if minimum is exclusive.
	ExclusiveMinimum *bool
	// MaxLength specifies the maximum length.
	MaxLength *int64
	// MinLength specifies the minimum length.
	MinLength *int64
	// Pattern specifies a regex pattern the string must match.
	Pattern *string
	// MaxItems specifies the maximum number of items in an array.
	MaxItems *int64
	// MinItems specifies the minimum number of items in an array.
	MinItems *int64
	// UniqueItems specifies if all items must be unique.
	UniqueItems *bool
	// Enum specifies a list of allowed values.
	Enum []values.Value
	// MultipleOf specifies the value must be a multiple of this number.
	MultipleOf *float64

	// Extensions provides a list of extensions to the Parameter object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Parameter] = (*Parameter)(nil)

// GetName returns the value of the Name field. Returns empty string if not set.
func (p *Parameter) GetName() string {
	if p == nil {
		return ""
	}
	return p.Name
}

// GetIn returns the value of the In field.
func (p *Parameter) GetIn() ParameterIn {
	if p == nil {
		return ""
	}
	return p.In
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (p *Parameter) GetDescription() string {
	if p == nil || p.Description == nil {
		return ""
	}
	return *p.Description
}

// GetRequired returns the value of the Required field. False by default if not set.
func (p *Parameter) GetRequired() bool {
	if p == nil || p.Required == nil {
		return false
	}
	return *p.Required
}

// GetSchema returns the value of the Schema field. Returns nil if not set.
func (p *Parameter) GetSchema() *oas3.JSONSchema[oas3.Referenceable] {
	if p == nil {
		return nil
	}
	return p.Schema
}

// GetType returns the value of the Type field. Returns empty string if not set.
func (p *Parameter) GetType() string {
	if p == nil || p.Type == nil {
		return ""
	}
	return *p.Type
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (p *Parameter) GetExtensions() *extensions.Extensions {
	if p == nil || p.Extensions == nil {
		return extensions.New()
	}
	return p.Extensions
}

// Validate validates the Parameter object against the Swagger Specification.
func (p *Parameter) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := p.GetCore()
	errs := []error{}

	if c.Name.Present && p.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`parameter.name` is required"), c, c.Name))
	}

	if c.In.Present && p.In == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.in is required"), c, c.In))
	} else if c.In.Present {
		errs = append(errs, p.validateIn(c)...)
		errs = append(errs, p.validateParameterType(ctx, c, opts...)...)
	}

	// allowEmptyValue only valid for query or formData
	if c.AllowEmptyValue.Present && p.In != ParameterInQuery && p.In != ParameterInFormData {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("parameter.allowEmptyValue is only valid for in=query or in=formData"), c, c.AllowEmptyValue))
	}

	// Validate items if present
	if c.Items.Present && p.Items != nil {
		errs = append(errs, p.Items.Validate(ctx, opts...)...)
	}

	// Validate file type parameter consumes from operation context
	if p.Type != nil && *p.Type == "file" {
		validationOpts := validation.NewOptions(opts...)
		if operation := validation.GetContextObject[Operation](validationOpts); operation != nil {
			opCore := operation.GetCore()
			if !opCore.Consumes.Present || len(operation.Consumes) == 0 {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationRequiredField,
					errors.New("parameter with type=file requires operation to have consumes defined"),
					c, c.Type))
			} else {
				hasValidConsumes := false
				for _, mimeType := range operation.Consumes {
					if mimeType == "multipart/form-data" || mimeType == "application/x-www-form-urlencoded" {
						hasValidConsumes = true
						break
					}
				}
				if !hasValidConsumes {
					errs = append(errs, validation.NewValueError(
						validation.SeverityError,
						validation.RuleValidationAllowedValues,
						errors.New("parameter with type=file requires operation consumes to be 'multipart/form-data' or 'application/x-www-form-urlencoded'"),
						c, c.Type))
				}
			}
		}
	}

	p.Valid = len(errs) == 0 && c.GetValid()

	return errs
}

func (p *Parameter) validateIn(c *core.Parameter) []error {
	errs := []error{}

	validIns := []ParameterIn{ParameterInQuery, ParameterInHeader, ParameterInPath, ParameterInFormData, ParameterInBody}
	valid := false
	for _, in := range validIns {
		if p.In == in {
			valid = true
			break
		}
	}
	if !valid {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.in must be one of [`%s`]", strings.Join([]string{string(ParameterInQuery), string(ParameterInHeader), string(ParameterInPath), string(ParameterInFormData), string(ParameterInBody)}, ", ")), c, c.In))
	}

	return errs
}

func (p *Parameter) validateParameterType(ctx context.Context, c *core.Parameter, opts ...validation.Option) []error {
	errs := []error{}

	// Path parameters must be required
	if p.In == ParameterInPath && (!c.Required.Present || !p.GetRequired()) {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.in=path requires required=true"), c, c.Required))
	}

	// Body parameters require schema
	if p.In == ParameterInBody {
		if !c.Schema.Present {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.schema is required for in=body"), c, c.Schema))
			return errs
		}
		errs = append(errs, p.Schema.Validate(ctx, opts...)...)
		return errs
	}

	// Non-body parameters require type
	if !c.Type.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.type is required for non-body parameters"), c, c.Type))
		return errs
	}

	if c.Type.Present && (p.Type == nil || *p.Type == "") {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.type is required for non-body parameters"), c, c.Type))
		return errs
	}

	if p.Type != nil {
		validTypes := []string{"string", "number", "integer", "boolean", "array", "file"}
		valid := false
		for _, t := range validTypes {
			if *p.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.type must be one of [`%s`]", strings.Join(validTypes, ", ")), c, c.Type))
		}

		// File type only allowed for formData
		if *p.Type == "file" && p.In != ParameterInFormData {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("parameter.type=file requires in=formData"), c, c.Type))
		}

		// Array type requires items
		if *p.Type == "array" && !c.Items.Present {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("parameter.items is required when type=array"), c, c.Items))
		}

		// Validate collectionFormat=multi only for query or formData
		if p.CollectionFormat != nil && *p.CollectionFormat == CollectionFormatMulti {
			if p.In != ParameterInQuery && p.In != ParameterInFormData {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationAllowedValues,
					errors.New("collectionFormat='multi' is only valid for in=query or in=formData"),
					c, c.CollectionFormat))
			}
		}
	}

	return errs
}

// Items is a limited subset of JSON-Schema's items object for array parameters.
type Items struct {
	marshaller.Model[core.Items]

	// Type is the internal type of the array.
	Type string
	// Format is the extending format for the type.
	Format *string
	// Items describes the type of items in nested arrays.
	Items *Items
	// CollectionFormat determines the format of the array.
	CollectionFormat *CollectionFormat
	// Default declares the value the server will use if none is provided.
	Default values.Value
	// Maximum specifies the maximum value.
	Maximum *float64
	// ExclusiveMaximum specifies if maximum is exclusive.
	ExclusiveMaximum *bool
	// Minimum specifies the minimum value.
	Minimum *float64
	// ExclusiveMinimum specifies if minimum is exclusive.
	ExclusiveMinimum *bool
	// MaxLength specifies the maximum length.
	MaxLength *int64
	// MinLength specifies the minimum length.
	MinLength *int64
	// Pattern specifies a regex pattern the string must match.
	Pattern *string
	// MaxItems specifies the maximum number of items in an array.
	MaxItems *int64
	// MinItems specifies the minimum number of items in an array.
	MinItems *int64
	// UniqueItems specifies if all items must be unique.
	UniqueItems *bool
	// Enum specifies a list of allowed values.
	Enum []values.Value
	// MultipleOf specifies the value must be a multiple of this number.
	MultipleOf *float64

	// Extensions provides a list of extensions to the Items object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Items] = (*Items)(nil)

// GetType returns the value of the Type field. Returns empty string if not set.
func (i *Items) GetType() string {
	if i == nil {
		return ""
	}
	return i.Type
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (i *Items) GetExtensions() *extensions.Extensions {
	if i == nil || i.Extensions == nil {
		return extensions.New()
	}
	return i.Extensions
}

// Validate validates the Items object against the Swagger Specification.
func (i *Items) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := i.GetCore()
	errs := []error{}

	if c.Type.Present && i.Type == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`items.type` is required"), c, c.Type))
	} else if c.Type.Present {
		validTypes := []string{"string", "number", "integer", "boolean", "array"}
		valid := false
		for _, t := range validTypes {
			if i.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("items.type must be one of [`%s`]", strings.Join(validTypes, ", ")), c, c.Type))
		}

		// Array type requires items
		if i.Type == "array" && !c.Items.Present {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("items.items is required when type=array"), c, c.Items))
		}
	}

	// Validate nested items if present
	if c.Items.Present && i.Items != nil {
		errs = append(errs, i.Items.Validate(ctx, opts...)...)
	}

	i.Valid = len(errs) == 0 && c.GetValid()

	return errs
}
