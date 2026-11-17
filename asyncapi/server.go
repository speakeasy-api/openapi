package asyncapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Server describes a message broker, a server or any other kind of computer program capable of sending and/or
// receiving data.
type Server struct {
	marshaller.Model[core.Server]

	// Host is the server host name. It may include the port and Server Variables. Required.
	Host string
	// Protocol is the protocol this server supports for connection, e.g. `kafka` or `mqtt`. Required.
	Protocol string
	// ProtocolVersion is the version of the protocol used for connection.
	ProtocolVersion *string
	// Pathname is the path to a resource in the host. It may include Server Variables.
	Pathname *string
	// Description is a description of the server. May contain CommonMark syntax.
	Description *string
	// Title is a human-friendly title for the server.
	Title *string
	// Summary is a short summary of the server.
	Summary *string
	// Variables are the variables used for substitution in the host and pathname.
	Variables *sequencedmap.Map[string, *ReferencedServerVariable]
	// Security are the security schemes that can be used with this server. Only one needs to be satisfied.
	Security []*ReferencedSecurityScheme
	// Tags are used for logical grouping and categorization of servers.
	Tags []*Tag
	// ExternalDocs is additional external documentation for this server.
	ExternalDocs *ExternalDocumentation
	// Bindings are protocol-specific definitions for the server.
	Bindings *ReferencedBindings
	// Extensions provides a list of extensions to the Server Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Server] = (*Server)(nil)

// GetHost returns the host, or an empty string if not set.
func (s *Server) GetHost() string {
	if s == nil {
		return ""
	}
	return s.Host
}

// GetProtocol returns the protocol, or an empty string if not set.
func (s *Server) GetProtocol() string {
	if s == nil {
		return ""
	}
	return s.Protocol
}

// GetProtocolVersion returns the protocol version, or an empty string if not set.
func (s *Server) GetProtocolVersion() string {
	if s == nil || s.ProtocolVersion == nil {
		return ""
	}
	return *s.ProtocolVersion
}

// GetPathname returns the pathname, or an empty string if not set.
func (s *Server) GetPathname() string {
	if s == nil || s.Pathname == nil {
		return ""
	}
	return *s.Pathname
}

// GetDescription returns the description, or an empty string if not set.
func (s *Server) GetDescription() string {
	if s == nil || s.Description == nil {
		return ""
	}
	return *s.Description
}

// GetTitle returns the title, or an empty string if not set.
func (s *Server) GetTitle() string {
	if s == nil || s.Title == nil {
		return ""
	}
	return *s.Title
}

// GetSummary returns the summary, or an empty string if not set.
func (s *Server) GetSummary() string {
	if s == nil || s.Summary == nil {
		return ""
	}
	return *s.Summary
}

// GetVariables returns the server variables, or nil if not set.
func (s *Server) GetVariables() *sequencedmap.Map[string, *ReferencedServerVariable] {
	if s == nil {
		return nil
	}
	return s.Variables
}

// GetSecurity returns the security schemes, or nil if not set.
func (s *Server) GetSecurity() []*ReferencedSecurityScheme {
	if s == nil {
		return nil
	}
	return s.Security
}

// GetTags returns the tags, or nil if not set.
func (s *Server) GetTags() []*Tag {
	if s == nil {
		return nil
	}
	return s.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (s *Server) GetExternalDocs() *ExternalDocumentation {
	if s == nil {
		return nil
	}
	return s.ExternalDocs
}

// GetBindings returns the server bindings, or nil if not set.
func (s *Server) GetBindings() *ReferencedBindings {
	if s == nil {
		return nil
	}
	return s.Bindings
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (s *Server) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// Validate will validate the Server Object against the AsyncAPI Specification.
func (s *Server) Validate(ctx context.Context, opts ...validation.Option) []error {
	if s == nil {
		return nil
	}

	c := s.GetCore()
	errs := []error{}

	errs = append(errs, requiredString(s.Host, c.Host.Present, c, c.Host, "server.host")...)
	errs = append(errs, requiredString(s.Protocol, c.Protocol.Present, c, c.Protocol, "server.protocol")...)

	if strings.Contains(s.Host, "://") {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("server.host `%s` must not include a scheme, use server.protocol instead", s.Host), c, c.Host))
	}

	for _, name := range expressionVariables(s.Host) {
		if s.Variables == nil || !s.Variables.Has(name) {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("server.host variable `%s` is not defined in server.variables", name), c, c.Host))
		}
	}
	for _, name := range expressionVariables(s.GetPathname()) {
		if s.Variables == nil || !s.Variables.Has(name) {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("server.pathname variable `%s` is not defined in server.variables", name), c, c.Pathname))
		}
	}

	if s.Variables != nil {
		for _, variable := range s.Variables.All() {
			errs = append(errs, variable.Validate(ctx, opts...)...)
		}
	}
	for _, scheme := range s.Security {
		errs = append(errs, scheme.Validate(ctx, opts...)...)
	}
	errs = append(errs, validateTags(ctx, s.Tags, c, c.Tags, "server.tags", opts...)...)
	if s.ExternalDocs != nil {
		errs = append(errs, s.ExternalDocs.Validate(ctx, opts...)...)
	}
	if s.Bindings != nil {
		errs = append(errs, s.Bindings.Validate(ctx, opts...)...)
	}

	errs = append(errs, unknownPropertyErrors(c, "server")...)

	s.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// ServerVariable is a variable for server host and pathname substitution.
type ServerVariable struct {
	marshaller.Model[core.ServerVariable]

	// Enum is the set of values the variable may take.
	Enum []string
	// Default is the value to use for substitution when no other value is supplied.
	Default *string
	// Description is a description of the variable. May contain CommonMark syntax.
	Description *string
	// Examples are examples of the variable's value.
	Examples []string
	// Extensions provides a list of extensions to the Server Variable Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.ServerVariable] = (*ServerVariable)(nil)

// GetEnum returns the allowed values, or nil if not set.
func (v *ServerVariable) GetEnum() []string {
	if v == nil {
		return nil
	}
	return v.Enum
}

// GetDefault returns the default value, or an empty string if not set.
func (v *ServerVariable) GetDefault() string {
	if v == nil || v.Default == nil {
		return ""
	}
	return *v.Default
}

// GetDescription returns the description, or an empty string if not set.
func (v *ServerVariable) GetDescription() string {
	if v == nil || v.Description == nil {
		return ""
	}
	return *v.Description
}

// GetExamples returns the examples, or nil if not set.
func (v *ServerVariable) GetExamples() []string {
	if v == nil {
		return nil
	}
	return v.Examples
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (v *ServerVariable) GetExtensions() *extensions.Extensions {
	if v == nil || v.Extensions == nil {
		return extensions.New()
	}
	return v.Extensions
}

// Validate will validate the Server Variable Object against the AsyncAPI Specification.
func (v *ServerVariable) Validate(_ context.Context, _ ...validation.Option) []error {
	if v == nil {
		return nil
	}

	c := v.GetCore()
	errs := []error{}

	errs = append(errs, validateEnumDefault(v.Enum, v.Default, c, c.Default, "serverVariable")...)
	errs = append(errs, unknownPropertyErrors(c, "serverVariable")...)

	v.Valid = noErrors(errs) && coreValid(c)

	return errs
}
