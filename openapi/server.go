package openapi

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

var variablePattern = regexp.MustCompile(`\{([^}]+)\}`)

// Server represents a server available to provide the functionality described in the API.
type Server struct {
	marshaller.Model[core.Server]

	// A URL to a server capable of providing the functionality described in the API.
	// The URL supports Server Variables and may be absolute or relative to where the OpenAPI document is located.
	URL string
	// A description of the server. May contain CommonMark syntax.
	Description *string
	// A map of variables available to be templated into the URL.
	Variables *sequencedmap.Map[string, *ServerVariable]

	// Extensions provides a list of extensions to the Server object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Server] = (*Server)(nil)

// GetURL returns the value of the URL field. Returns empty string if not set.
func (s *Server) GetURL() string {
	if s == nil {
		return ""
	}
	return s.URL
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (s *Server) GetDescription() string {
	if s == nil || s.Description == nil {
		return ""
	}
	return *s.Description
}

// GetVariables returns the value of the Variables field. Returns nil if not set.
func (s *Server) GetVariables() *sequencedmap.Map[string, *ServerVariable] {
	if s == nil {
		return nil
	}
	return s.Variables
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (s *Server) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// Validate will validate the Server object against the OpenAPI Specification.
func (s *Server) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := s.GetCore()
	errs := []error{}

	if core.URL.Present {
		if s.URL == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("server field url is required"), core, core.URL))
		} else if !strings.Contains(s.URL, "{") {
			if _, err := url.Parse(s.URL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("server field url is not a valid uri: %s", err), core, core.URL))
			}
		} else {
			if resolvedURL, err := resolveServerVariables(s.URL, s.Variables); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("server field url is not a valid uri: %s", err), core, core.URL))
			} else if _, err := url.Parse(resolvedURL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("server field url is not a valid uri: %s", err), core, core.URL))
			}
		}
	}

	for _, variable := range s.Variables.All() {
		errs = append(errs, variable.Validate(ctx, opts...)...)
	}

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// ServerVariable represents a variable available to be templated in the associated Server's URL.
type ServerVariable struct {
	marshaller.Model[core.ServerVariable]

	// The default value to use for substitution. If none is provided by the end-user.
	Default string
	// A restricted set of allowed values if provided.
	Enum []string
	// A description of the variable.
	Description *string

	// Extensions provides a list of extensions to the ServerVariable object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.ServerVariable] = (*ServerVariable)(nil)

// GetDefault returns the value of the Default field. Returns empty string if not set.
func (v *ServerVariable) GetDefault() string {
	if v == nil {
		return ""
	}
	return v.Default
}

// GetEnum returns the value of the Enum field. Returns nil if not set.
func (v *ServerVariable) GetEnum() []string {
	if v == nil {
		return nil
	}
	return v.Enum
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (v *ServerVariable) GetDescription() string {
	if v == nil || v.Description == nil {
		return ""
	}
	return *v.Description
}

// Validate will validate the ServerVariable object against the OpenAPI Specification.
func (v *ServerVariable) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := v.GetCore()
	errs := []error{}

	if core.Default.Present && v.Default == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("serverVariable field default is required"), core, core.Default))
	}

	if core.Enum.Present {
		if !slices.Contains(v.Enum, v.Default) {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("serverVariable field default must be one of [%s]", strings.Join(v.Enum, ", ")), core, core.Enum))
		}
	}

	v.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func resolveServerVariables(serverURL string, variables *sequencedmap.Map[string, *ServerVariable]) (string, error) {
	if variables.Len() == 0 {
		return "", fmt.Errorf("serverURL contains variables but no variables are defined")
	}

	resolvedURL := serverURL

	matches := variablePattern.FindAllStringSubmatch(serverURL, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		variableName := match[1]
		placeholder := match[0]

		variable, exists := variables.Get(variableName)
		if !exists {
			return "", fmt.Errorf("server variable '%s' is not defined", variableName)
		}

		if variable.Default == "" {
			return "", fmt.Errorf("server variable '%s' has no default value", variableName)
		}

		resolvedURL = strings.ReplaceAll(resolvedURL, placeholder, variable.Default)
	}

	return resolvedURL, nil
}
