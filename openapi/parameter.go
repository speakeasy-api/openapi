package openapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

// ParameterIn represents the location of a parameter that is passed in the request.
type ParameterIn string

var _ fmt.Stringer = (*ParameterIn)(nil)

func (p ParameterIn) String() string {
	return string(p)
}

const (
	// ParameterInQuery represents the location of a parameter that is passed in the query string.
	ParameterInQuery ParameterIn = "query"
	// ParameterInQueryString represents the location of a parameter that is passed as the entire query string.
	ParameterInQueryString ParameterIn = "querystring"
	// ParameterInHeader represents the location of a parameter that is passed in the header.
	ParameterInHeader ParameterIn = "header"
	// ParameterInPath represents the location of a parameter that is passed in the path.
	ParameterInPath ParameterIn = "path"
	// ParameterInCookie represents the location of a parameter that is passed in the cookie.
	ParameterInCookie ParameterIn = "cookie"
)

// Parameter represents a single parameter to be included in a request.
type Parameter struct {
	marshaller.Model[core.Parameter]

	// Name is the case sensitive name of the parameter.
	Name string
	// In is the location of the parameter. One of "query", "querystring", "header", "path" or "cookie".
	In ParameterIn
	// Description is a brief description of the parameter. May contain CommonMark syntax.
	Description *string
	// Required determines whether this parameter is mandatory. If the parameter location is "path", this property is REQUIRED and its value MUST be true.
	Required *bool
	// Deprecated describes whether this parameter is deprecated.
	Deprecated *bool
	// AllowEmptyValue determines if empty values are allowed for query parameters.
	AllowEmptyValue *bool
	// Style determines the serialization style of the parameter.
	Style *SerializationStyle
	// Explode determines for array and object values whether separate parameters should be generated for each item in the array or object.
	Explode *bool
	// AllowReserved determines if the value of this parameter can contain reserved characters as defined by RFC3986.
	AllowReserved *bool
	// Schema is the schema defining the type used for the parameter. Mutually exclusive with Content.
	Schema *oas3.JSONSchema[oas3.Referenceable]
	// Content represents the content type and schema of a parameter. Mutually exclusive with Schema.
	Content *sequencedmap.Map[string, *MediaType]
	// Example is an example of the parameter's value. Mutually exclusive with Examples.
	Example values.Value
	// Examples is a map of examples of the parameter's value. Mutually exclusive with Example.
	Examples *sequencedmap.Map[string, *ReferencedExample]
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

// GetIn returns the value of the In field. Returns empty ParameterIn if not set.
func (p *Parameter) GetIn() ParameterIn {
	if p == nil {
		return ""
	}
	return p.In
}

// GetSchema returns the value of the Schema field. Returns nil if not set.
func (p *Parameter) GetSchema() *oas3.JSONSchema[oas3.Referenceable] {
	if p == nil {
		return nil
	}
	return p.Schema
}

// GetRequired returns the value of the Required field. False by default if not set.
func (p *Parameter) GetRequired() bool {
	if p == nil || p.Required == nil {
		return false
	}
	return *p.Required
}

// GetDeprecated returns the value of the Deprecated field. False by default if not set.
func (p *Parameter) GetDeprecated() bool {
	if p == nil || p.Deprecated == nil {
		return false
	}
	return *p.Deprecated
}

// GetAllowEmptyValue returns the value of the AllowEmptyValue field. False by default if not set.
func (p *Parameter) GetAllowEmptyValue() bool {
	if p == nil || p.AllowEmptyValue == nil {
		return false
	}
	return *p.AllowEmptyValue
}

// GetStyle returns the value of the Style field. Defaults determined by the In field.
//
// Defaults:
//   - ParameterInQuery: SerializationStyleForm
//   - ParameterInHeader: SerializationStyleSimple
//   - ParameterInPath: SerializationStyleSimple
//   - ParameterInCookie: SerializationStyleForm
//   - ParameterInQueryString: Incompatible with style field
func (p *Parameter) GetStyle() SerializationStyle {
	if p == nil || p.Style == nil {
		switch p.In {
		case ParameterInQuery:
			return SerializationStyleForm
		case ParameterInHeader:
			return SerializationStyleSimple
		case ParameterInPath:
			return SerializationStyleSimple
		case ParameterInCookie:
			return SerializationStyleForm
		case ParameterInQueryString:
			return "" // No style allowed for querystring parameters
		default:
			return "" // Unknown type
		}
	}
	return *p.Style
}

// GetExplode returns the value of the Explode field. When style is "form" default is true otherwise false.
func (p *Parameter) GetExplode() bool {
	if p == nil || p.Explode == nil {
		return p.GetStyle() == SerializationStyleForm
	}
	return *p.Explode
}

// GetContent returns the value of the Content field. Returns nil if not set.
func (p *Parameter) GetContent() *sequencedmap.Map[string, *MediaType] {
	if p == nil {
		return nil
	}
	return p.Content
}

// GetExample returns the value of the Example field. Returns nil if not set.
func (p *Parameter) GetExample() values.Value {
	if p == nil {
		return nil
	}
	return p.Example
}

// GetExamples returns the value of the Examples field. Returns nil if not set.
func (p *Parameter) GetExamples() *sequencedmap.Map[string, *ReferencedExample] {
	if p == nil {
		return nil
	}
	return p.Examples
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (p *Parameter) GetExtensions() *extensions.Extensions {
	if p == nil || p.Extensions == nil {
		return extensions.New()
	}
	return p.Extensions
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (p *Parameter) GetDescription() string {
	if p == nil || p.Description == nil {
		return ""
	}
	return *p.Description
}

// GetAllowReserved returns the value of the AllowReserved field. False by default if not set.
func (p *Parameter) GetAllowReserved() bool {
	if p == nil || p.AllowReserved == nil {
		return false
	}
	return *p.AllowReserved
}

// Validate will validate the Parameter object against the OpenAPI Specification.
func (p *Parameter) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := p.GetCore()
	errs := []error{}

	if core.Name.Present && p.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`parameter.name` is required"), core, core.Name))
	}

	if core.In.Present && p.In == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`parameter.in` is required"), core, core.In))
	} else {
		switch p.In {
		case ParameterInQuery, ParameterInQueryString, ParameterInHeader, ParameterInPath, ParameterInCookie:
		default:
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.in must be one of [`%s`]", strings.Join([]string{string(ParameterInQuery), string(ParameterInQueryString), string(ParameterInHeader), string(ParameterInPath), string(ParameterInCookie)}, ", ")), core, core.In))
		}
	}

	if p.In == ParameterInPath && (!core.Required.Present || !*p.Required) {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`parameter.in=path` requires `required=true`"), core, core.Required))
	}

	if core.AllowEmptyValue.Present && p.In != ParameterInQuery {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("`parameter.allowEmptyValue` is only valid for `in=query`"), core, core.AllowEmptyValue))
	}

	if core.Style.Present {
		switch p.In {
		case ParameterInQueryString:
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("parameter field style is not allowed for in=querystring"), core, core.Style))

		case ParameterInPath:
			allowedStyles := []string{string(SerializationStyleSimple), string(SerializationStyleLabel), string(SerializationStyleMatrix)}
			if !slices.Contains(allowedStyles, string(*p.Style)) {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.style must be one of [`%s`] for in=path", strings.Join(allowedStyles, ", ")), core, core.Style))
			}
		case ParameterInQuery:
			allowedStyles := []string{string(SerializationStyleForm), string(SerializationStyleSpaceDelimited), string(SerializationStylePipeDelimited), string(SerializationStyleDeepObject)}
			if !slices.Contains(allowedStyles, string(*p.Style)) {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.style must be one of [`%s`] for in=query", strings.Join(allowedStyles, ", ")), core, core.Style))
			}
		case ParameterInHeader:
			allowedStyles := []string{string(SerializationStyleSimple)}
			if !slices.Contains(allowedStyles, string(*p.Style)) {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.style must be one of [`%s`] for in=header", strings.Join(allowedStyles, ", ")), core, core.Style))
			}
		case ParameterInCookie:
			allowedStyles := []string{string(SerializationStyleForm)}
			if !slices.Contains(allowedStyles, string(*p.Style)) {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("parameter.style must be one of [`%s`] for in=cookie", strings.Join(allowedStyles, ", ")), core, core.Style))
			}
		}
	}

	if core.Schema.Present {
		switch p.In {
		case ParameterInQueryString:
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("`parameter.schema` is not allowed for `in=querystring`"), core, core.Schema))
		default:
			errs = append(errs, p.Schema.Validate(ctx, opts...)...)
		}
	}

	if !core.Content.Present || p.Content == nil {
		// Querystring parameters must use content instead of schema
		if p.In == ParameterInQueryString {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`parameter.content` is required for `in=querystring`"), core, core.Content))
		}
	} else if p.Content.Len() != 1 {
		// If present, content must have exactly one entry
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, errors.New("`parameter.content` must have exactly one entry"), core, core.Content))
	}

	for mediaType, obj := range p.Content.All() {
		// Pass media type context for validation
		contentOpts := append([]validation.Option{}, opts...)
		contentOpts = append(contentOpts, validation.WithContextObject(&MediaTypeContext{MediaType: mediaType}))
		errs = append(errs, obj.Validate(ctx, contentOpts...)...)
	}

	for _, obj := range p.Examples.All() {
		errs = append(errs, obj.Validate(ctx, opts...)...)
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
