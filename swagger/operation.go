package swagger

import (
	"context"
	"errors"
	"fmt"
	"mime"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Operation describes a single API operation on a path.
type Operation struct {
	marshaller.Model[core.Operation]

	// Tags is a list of tags for API documentation control.
	Tags []string
	// Summary is a short summary of what the operation does.
	Summary *string
	// Description is a verbose explanation of the operation behavior.
	Description *string
	// ExternalDocs is additional external documentation for this operation.
	ExternalDocs *ExternalDocumentation
	// OperationID is a unique string used to identify the operation.
	OperationID *string
	// Consumes is a list of MIME types the operation can consume.
	Consumes []string
	// Produces is a list of MIME types the operation can produce.
	Produces []string
	// Parameters is a list of parameters that are applicable for this operation.
	Parameters []*ReferencedParameter
	// Responses is the list of possible responses as they are returned from executing this operation.
	Responses *Responses
	// Schemes is the transfer protocol for the operation.
	Schemes []string
	// Deprecated declares this operation to be deprecated.
	Deprecated *bool
	// Security is a declaration of which security schemes are applied for this operation.
	Security []*SecurityRequirement
	// Extensions provides a list of extensions to the Operation object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Operation] = (*Operation)(nil)

// GetTags returns the value of the Tags field. Returns nil if not set.
func (o *Operation) GetTags() []string {
	if o == nil {
		return nil
	}
	return o.Tags
}

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (o *Operation) GetSummary() string {
	if o == nil || o.Summary == nil {
		return ""
	}
	return *o.Summary
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (o *Operation) GetDescription() string {
	if o == nil || o.Description == nil {
		return ""
	}
	return *o.Description
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (o *Operation) GetExternalDocs() *ExternalDocumentation {
	if o == nil {
		return nil
	}
	return o.ExternalDocs
}

// GetOperationID returns the value of the OperationID field. Returns empty string if not set.
func (o *Operation) GetOperationID() string {
	if o == nil || o.OperationID == nil {
		return ""
	}
	return *o.OperationID
}

// GetConsumes returns the value of the Consumes field. Returns nil if not set.
func (o *Operation) GetConsumes() []string {
	if o == nil {
		return nil
	}
	return o.Consumes
}

// GetProduces returns the value of the Produces field. Returns nil if not set.
func (o *Operation) GetProduces() []string {
	if o == nil {
		return nil
	}
	return o.Produces
}

// GetParameters returns the value of the Parameters field. Returns nil if not set.
func (o *Operation) GetParameters() []*ReferencedParameter {
	if o == nil {
		return nil
	}
	return o.Parameters
}

// GetResponses returns the value of the Responses field. Returns nil if not set.
func (o *Operation) GetResponses() *Responses {
	if o == nil {
		return nil
	}
	return o.Responses
}

// GetSchemes returns the value of the Schemes field. Returns nil if not set.
func (o *Operation) GetSchemes() []string {
	if o == nil {
		return nil
	}
	return o.Schemes
}

// GetDeprecated returns the value of the Deprecated field. False by default if not set.
func (o *Operation) GetDeprecated() bool {
	if o == nil || o.Deprecated == nil {
		return false
	}
	return *o.Deprecated
}

// GetSecurity returns the value of the Security field. Returns nil if not set.
func (o *Operation) GetSecurity() []*SecurityRequirement {
	if o == nil {
		return nil
	}
	return o.Security
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (o *Operation) GetExtensions() *extensions.Extensions {
	if o == nil || o.Extensions == nil {
		return extensions.New()
	}
	return o.Extensions
}

// Validate validates the Operation object against the Swagger Specification.
func (o *Operation) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := o.GetCore()
	errs := []error{}

	if !c.Responses.Present {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("operation.responses is required"), c, c.Responses))
	} else if o.Responses != nil {
		errs = append(errs, o.Responses.Validate(ctx, opts...)...)
	}

	// Validate schemes if present
	if c.Schemes.Present {
		validSchemes := []string{"http", "https", "ws", "wss"}
		for _, scheme := range o.Schemes {
			valid := false
			for _, vs := range validSchemes {
				if scheme == vs {
					valid = true
					break
				}
			}
			if !valid {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationAllowedValues,
					fmt.Errorf("operation.scheme must be one of [http, https, ws, wss], got '%s'", scheme),
					c, c.Schemes))
			}
		}
	}

	// Validate consumes MIME types
	if c.Consumes.Present {
		for _, mimeType := range o.Consumes {
			if _, _, err := mime.ParseMediaType(mimeType); err != nil {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationInvalidFormat,
					fmt.Errorf("operation.consumes contains invalid MIME type '%s': %w", mimeType, err),
					c, c.Consumes))
			}
		}
	}

	// Validate produces MIME types
	if c.Produces.Present {
		for _, mimeType := range o.Produces {
			if _, _, err := mime.ParseMediaType(mimeType); err != nil {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationInvalidFormat,
					fmt.Errorf("operation.produces contains invalid MIME type '%s': %w", mimeType, err),
					c, c.Produces))
			}
		}
	}

	if c.ExternalDocs.Present && o.ExternalDocs != nil {
		errs = append(errs, o.ExternalDocs.Validate(ctx, opts...)...)
	}

	// TODO allow validation of parameter uniqueness and body parameter count, this isn't done at the moment as we would need to resolve references
	// Pass operation as context for file type parameter validation
	for _, param := range o.Parameters {
		errs = append(errs, param.Validate(ctx, append(opts, validation.WithContextObject(o))...)...)
	}

	// Pass operation's parent Swagger as context for security requirement validation
	// Note: Swagger context should be provided by caller
	for _, secReq := range o.Security {
		errs = append(errs, secReq.Validate(ctx, opts...)...)
	}

	o.Valid = len(errs) == 0 && c.GetValid()

	return errs
}
