package swagger

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Version is the Swagger specification version supported by this package.
const Version = "2.0"

// Swagger is the root document object for the API specification.
type Swagger struct {
	marshaller.Model[core.Swagger]

	// Swagger is the version of the Swagger specification that this document uses.
	Swagger string
	// Info provides metadata about the API.
	Info Info
	// Host is the host (name or ip) serving the API.
	Host *string
	// BasePath is the base path on which the API is served.
	BasePath *string
	// Schemes is the transfer protocol of the API.
	Schemes []string
	// Consumes is a list of MIME types the APIs can consume.
	Consumes []string
	// Produces is a list of MIME types the APIs can produce.
	Produces []string
	// Paths is the available paths and operations for the API.
	Paths *Paths
	// Definitions is an object to hold data types produced and consumed by operations.
	Definitions *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Concrete]]
	// Parameters is an object to hold parameters that can be used across operations.
	Parameters *sequencedmap.Map[string, *Parameter]
	// Responses is an object to hold responses that can be used across operations.
	Responses *sequencedmap.Map[string, *Response]
	// SecurityDefinitions are security scheme definitions that can be used across the specification.
	SecurityDefinitions *sequencedmap.Map[string, *SecurityScheme]
	// Security is a declaration of which security schemes are applied for the API as a whole.
	Security []*SecurityRequirement
	// Tags is a list of tags used by the specification with additional metadata.
	Tags []*Tag
	// ExternalDocs is additional external documentation.
	ExternalDocs *ExternalDocumentation
	// Extensions provides a list of extensions to the Swagger object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Swagger] = (*Swagger)(nil)

// GetSwagger returns the value of the Swagger field. Returns empty string if not set.
func (s *Swagger) GetSwagger() string {
	if s == nil {
		return ""
	}
	return s.Swagger
}

// GetInfo returns the value of the Info field.
func (s *Swagger) GetInfo() *Info {
	if s == nil {
		return nil
	}
	return &s.Info
}

// GetHost returns the value of the Host field. Returns empty string if not set.
func (s *Swagger) GetHost() string {
	if s == nil || s.Host == nil {
		return ""
	}
	return *s.Host
}

// GetBasePath returns the value of the BasePath field. Returns empty string if not set.
func (s *Swagger) GetBasePath() string {
	if s == nil || s.BasePath == nil {
		return ""
	}
	return *s.BasePath
}

// GetSchemes returns the value of the Schemes field. Returns nil if not set.
func (s *Swagger) GetSchemes() []string {
	if s == nil {
		return nil
	}
	return s.Schemes
}

// GetConsumes returns the value of the Consumes field. Returns nil if not set.
func (s *Swagger) GetConsumes() []string {
	if s == nil {
		return nil
	}
	return s.Consumes
}

// GetProduces returns the value of the Produces field. Returns nil if not set.
func (s *Swagger) GetProduces() []string {
	if s == nil {
		return nil
	}
	return s.Produces
}

// GetPaths returns the value of the Paths field. Returns nil if not set.
func (s *Swagger) GetPaths() *Paths {
	if s == nil {
		return nil
	}
	return s.Paths
}

// GetDefinitions returns the value of the Definitions field. Returns nil if not set.
func (s *Swagger) GetDefinitions() *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Concrete]] {
	if s == nil {
		return nil
	}
	return s.Definitions
}

// GetParameters returns the value of the Parameters field. Returns nil if not set.
func (s *Swagger) GetParameters() *sequencedmap.Map[string, *Parameter] {
	if s == nil {
		return nil
	}
	return s.Parameters
}

// GetResponses returns the value of the Responses field. Returns nil if not set.
func (s *Swagger) GetResponses() *sequencedmap.Map[string, *Response] {
	if s == nil {
		return nil
	}
	return s.Responses
}

// GetSecurityDefinitions returns the value of the SecurityDefinitions field. Returns nil if not set.
func (s *Swagger) GetSecurityDefinitions() *sequencedmap.Map[string, *SecurityScheme] {
	if s == nil {
		return nil
	}
	return s.SecurityDefinitions
}

// GetSecurity returns the value of the Security field. Returns nil if not set.
func (s *Swagger) GetSecurity() []*SecurityRequirement {
	if s == nil {
		return nil
	}
	return s.Security
}

// GetTags returns the value of the Tags field. Returns nil if not set.
func (s *Swagger) GetTags() []*Tag {
	if s == nil {
		return nil
	}
	return s.Tags
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (s *Swagger) GetExternalDocs() *ExternalDocumentation {
	if s == nil {
		return nil
	}
	return s.ExternalDocs
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (s *Swagger) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// Validate validates the Swagger object against the Swagger Specification.
func (s *Swagger) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := s.GetCore()
	errs := []error{}

	if c.Swagger.Present && s.Swagger == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("swagger is required"), c, c.Swagger))
	} else if c.Swagger.Present && s.Swagger != "2.0" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, errors.New("swagger must be '2.0'"), c, c.Swagger))
	}

	if c.Info.Present {
		errs = append(errs, s.Info.Validate(ctx, opts...)...)
	}

	// Validate basePath starts with leading slash
	if c.BasePath.Present && s.BasePath != nil && *s.BasePath != "" {
		if !strings.HasPrefix(*s.BasePath, "/") {
			errs = append(errs, validation.NewValueError(
				validation.SeverityError,
				validation.RuleValidationInvalidSyntax,
				errors.New("basePath must start with a leading slash '/'"),
				c, c.BasePath))
		}
	}

	// Validate schemes if present
	if c.Schemes.Present {
		validSchemes := []string{"http", "https", "ws", "wss"}
		for _, scheme := range s.Schemes {
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
					fmt.Errorf("scheme must be one of [http, https, ws, wss], got '%s'", scheme),
					c, c.Schemes))
			}
		}
	}

	// Validate consumes MIME types
	if c.Consumes.Present {
		for _, mimeType := range s.Consumes {
			if _, _, err := mime.ParseMediaType(mimeType); err != nil {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationInvalidFormat,
					fmt.Errorf("consumes contains invalid MIME type '%s': %w", mimeType, err),
					c, c.Consumes))
			}
		}
	}

	// Validate produces MIME types
	if c.Produces.Present {
		for _, mimeType := range s.Produces {
			if _, _, err := mime.ParseMediaType(mimeType); err != nil {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationInvalidFormat,
					fmt.Errorf("produces contains invalid MIME type '%s': %w", mimeType, err),
					c, c.Produces))
			}
		}
	}

	// Pass Swagger as context for nested validation (operations, security requirements)
	if c.Paths.Present && s.Paths != nil {
		errs = append(errs, s.Paths.Validate(ctx, append(opts, validation.WithContextObject(s))...)...)
	}

	// Validate tag names are unique
	tagNames := make(map[string]bool)
	for _, tag := range s.Tags {
		if tag != nil && tag.Name != "" {
			if tagNames[tag.Name] {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationDuplicateKey,
					fmt.Errorf("tag name '%s' must be unique", tag.Name),
					c, c.Tags))
			}
			tagNames[tag.Name] = true
		}
		errs = append(errs, tag.Validate(ctx, opts...)...)
	}

	if c.ExternalDocs.Present && s.ExternalDocs != nil {
		errs = append(errs, s.ExternalDocs.Validate(ctx, opts...)...)
	}

	for _, param := range s.Parameters.All() {
		errs = append(errs, param.Validate(ctx, opts...)...)
	}

	for _, resp := range s.Responses.All() {
		errs = append(errs, resp.Validate(ctx, opts...)...)
	}

	for _, secScheme := range s.SecurityDefinitions.All() {
		errs = append(errs, secScheme.Validate(ctx, opts...)...)
	}

	// Pass Swagger as context for security requirement validation
	for _, secReq := range s.Security {
		errs = append(errs, secReq.Validate(ctx, append(opts, validation.WithContextObject(s))...)...)
	}

	// Validate operationId uniqueness across all operations
	errs = append(errs, s.validateOperationIDUniqueness(c)...)

	s.Valid = len(errs) == 0 && c.GetValid()

	return errs
}

// validateOperationIDUniqueness validates that all operationIds are unique across the document
func (s *Swagger) validateOperationIDUniqueness(c *core.Swagger) []error {
	errs := []error{}
	operationIDs := make(map[string]bool)

	if s.Paths == nil {
		return errs
	}

	for _, pathItem := range s.Paths.All() {
		if pathItem == nil {
			continue
		}

		for _, operation := range pathItem.All() {
			if operation == nil || operation.OperationID == nil || *operation.OperationID == "" {
				continue
			}

			opID := *operation.OperationID
			if operationIDs[opID] {
				errs = append(errs, validation.NewValueError(
					validation.SeverityError,
					validation.RuleValidationDuplicateKey,
					fmt.Errorf("operationId '%s' must be unique among all operations", opID),
					c, c.Paths))
			}
			operationIDs[opID] = true
		}
	}

	return errs
}
