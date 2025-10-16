package swagger

import (
	"context"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// SecuritySchemeType represents the type of security scheme.
type SecuritySchemeType string

const (
	// SecuritySchemeTypeBasic represents basic authentication.
	SecuritySchemeTypeBasic SecuritySchemeType = "basic"
	// SecuritySchemeTypeAPIKey represents API key authentication.
	SecuritySchemeTypeAPIKey SecuritySchemeType = "apiKey"
	// SecuritySchemeTypeOAuth2 represents OAuth2 authentication.
	SecuritySchemeTypeOAuth2 SecuritySchemeType = "oauth2"
)

// SecuritySchemeIn represents the location of the API key.
type SecuritySchemeIn string

const (
	// SecuritySchemeInQuery represents an API key in the query string.
	SecuritySchemeInQuery SecuritySchemeIn = "query"
	// SecuritySchemeInHeader represents an API key in the header.
	SecuritySchemeInHeader SecuritySchemeIn = "header"
)

// OAuth2Flow represents the flow type for OAuth2.
type OAuth2Flow string

const (
	// OAuth2FlowImplicit represents the implicit flow.
	OAuth2FlowImplicit OAuth2Flow = "implicit"
	// OAuth2FlowPassword represents the password flow.
	OAuth2FlowPassword OAuth2Flow = "password"
	// OAuth2FlowApplication represents the application flow.
	OAuth2FlowApplication OAuth2Flow = "application"
	// OAuth2FlowAccessCode represents the access code flow.
	OAuth2FlowAccessCode OAuth2Flow = "accessCode"
)

// SecurityScheme defines a security scheme that can be used by the operations.
type SecurityScheme struct {
	marshaller.Model[core.SecurityScheme]

	// Type is the type of the security scheme. Valid values are "basic", "apiKey" or "oauth2".
	Type SecuritySchemeType
	// Description is a short description for security scheme.
	Description *string
	// Name is the name of the header or query parameter to be used (apiKey only).
	Name *string
	// In is the location of the API key. Valid values are "query" or "header" (apiKey only).
	In *SecuritySchemeIn
	// Flow is the flow used by the OAuth2 security scheme. Valid values are "implicit", "password", "application" or "accessCode" (oauth2 only).
	Flow *OAuth2Flow
	// AuthorizationURL is the authorization URL to be used for this flow (oauth2 "implicit" and "accessCode" only).
	AuthorizationURL *string
	// TokenURL is the token URL to be used for this flow (oauth2 "password", "application" and "accessCode" only).
	TokenURL *string
	// Scopes lists the available scopes for the OAuth2 security scheme (oauth2 only).
	Scopes *sequencedmap.Map[string, string]
	// Extensions provides a list of extensions to the SecurityScheme object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.SecurityScheme] = (*SecurityScheme)(nil)

// GetType returns the value of the Type field.
func (s *SecurityScheme) GetType() SecuritySchemeType {
	if s == nil {
		return ""
	}
	return s.Type
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (s *SecurityScheme) GetDescription() string {
	if s == nil || s.Description == nil {
		return ""
	}
	return *s.Description
}

// GetName returns the value of the Name field. Returns empty string if not set.
func (s *SecurityScheme) GetName() string {
	if s == nil || s.Name == nil {
		return ""
	}
	return *s.Name
}

// GetIn returns the value of the In field. Returns empty string if not set.
func (s *SecurityScheme) GetIn() SecuritySchemeIn {
	if s == nil || s.In == nil {
		return ""
	}
	return *s.In
}

// GetFlow returns the value of the Flow field. Returns empty string if not set.
func (s *SecurityScheme) GetFlow() OAuth2Flow {
	if s == nil || s.Flow == nil {
		return ""
	}
	return *s.Flow
}

// GetAuthorizationURL returns the value of the AuthorizationURL field. Returns empty string if not set.
func (s *SecurityScheme) GetAuthorizationURL() string {
	if s == nil || s.AuthorizationURL == nil {
		return ""
	}
	return *s.AuthorizationURL
}

// GetTokenURL returns the value of the TokenURL field. Returns empty string if not set.
func (s *SecurityScheme) GetTokenURL() string {
	if s == nil || s.TokenURL == nil {
		return ""
	}
	return *s.TokenURL
}

// GetScopes returns the value of the Scopes field. Returns nil if not set.
func (s *SecurityScheme) GetScopes() *sequencedmap.Map[string, string] {
	if s == nil {
		return nil
	}
	return s.Scopes
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (s *SecurityScheme) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// Validate validates the SecurityScheme object against the Swagger Specification.
func (s *SecurityScheme) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := s.GetCore()
	errs := []error{}

	if c.Type.Present && s.Type == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.type is required"), c, c.Type))
	} else {
		validTypes := []SecuritySchemeType{SecuritySchemeTypeBasic, SecuritySchemeTypeAPIKey, SecuritySchemeTypeOAuth2}
		valid := false
		for _, t := range validTypes {
			if s.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme.type must be one of [%s]", strings.Join([]string{string(SecuritySchemeTypeBasic), string(SecuritySchemeTypeAPIKey), string(SecuritySchemeTypeOAuth2)}, ", ")), c, c.Type))
		}
	}

	// Validate apiKey specific fields
	if s.Type == SecuritySchemeTypeAPIKey {
		if !c.Name.Present || s.Name == nil || *s.Name == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.name is required for type=apiKey"), c, c.Name))
		}
		if !c.In.Present || s.In == nil {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.in is required for type=apiKey"), c, c.In))
		} else if *s.In != SecuritySchemeInQuery && *s.In != SecuritySchemeInHeader {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme.in must be one of [%s] for type=apiKey", strings.Join([]string{string(SecuritySchemeInQuery), string(SecuritySchemeInHeader)}, ", ")), c, c.In))
		}
	}

	// Validate oauth2 specific fields
	if s.Type == SecuritySchemeTypeOAuth2 {
		if !c.Flow.Present || s.Flow == nil {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.flow is required for type=oauth2"), c, c.Flow))
		} else {
			validFlows := []OAuth2Flow{OAuth2FlowImplicit, OAuth2FlowPassword, OAuth2FlowApplication, OAuth2FlowAccessCode}
			valid := false
			for _, f := range validFlows {
				if *s.Flow == f {
					valid = true
					break
				}
			}
			if !valid {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme.flow must be one of [%s] for type=oauth2", strings.Join([]string{string(OAuth2FlowImplicit), string(OAuth2FlowPassword), string(OAuth2FlowApplication), string(OAuth2FlowAccessCode)}, ", ")), c, c.Flow))
			}

			if s.Flow != nil {
				// authorizationUrl required for implicit and accessCode flows
				if (*s.Flow == OAuth2FlowImplicit || *s.Flow == OAuth2FlowAccessCode) && (!c.AuthorizationURL.Present || s.AuthorizationURL == nil || *s.AuthorizationURL == "") {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.authorizationUrl is required for flow=%s", *s.Flow), c, c.AuthorizationURL))
				}

				// tokenUrl required for password, application and accessCode flows
				if (*s.Flow == OAuth2FlowPassword || *s.Flow == OAuth2FlowApplication || *s.Flow == OAuth2FlowAccessCode) && (!c.TokenURL.Present || s.TokenURL == nil || *s.TokenURL == "") {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.tokenUrl is required for flow=%s", *s.Flow), c, c.TokenURL))
				}
			}
		}

		if !c.Scopes.Present {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme.scopes is required for type=oauth2"), c, c.Scopes))
		}
	}

	// Validate URLs
	if c.AuthorizationURL.Present && s.AuthorizationURL != nil && *s.AuthorizationURL != "" {
		if _, err := url.Parse(*s.AuthorizationURL); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme.authorizationUrl is not a valid uri: %s", err), c, c.AuthorizationURL))
		}
	}

	if c.TokenURL.Present && s.TokenURL != nil && *s.TokenURL != "" {
		if _, err := url.Parse(*s.TokenURL); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme.tokenUrl is not a valid uri: %s", err), c, c.TokenURL))
		}
	}

	s.Valid = len(errs) == 0 && c.GetValid()

	return errs
}

// SecurityRequirement lists the required security schemes to execute an operation.
// The keys are the names of security schemes and the values are lists of scope names.
// For non-oauth2 security schemes, the array MUST be empty.
type SecurityRequirement struct {
	marshaller.Model[core.SecurityRequirement]
	*sequencedmap.Map[string, []string]
}

var _ interfaces.Model[core.SecurityRequirement] = (*SecurityRequirement)(nil)

// NewSecurityRequirement creates a new SecurityRequirement with an initialized map.
func NewSecurityRequirement() *SecurityRequirement {
	return &SecurityRequirement{
		Map: sequencedmap.New[string, []string](),
	}
}

// Validate validates the SecurityRequirement object against the Swagger Specification.
func (s *SecurityRequirement) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := s.GetCore()
	errs := []error{}

	// Get Swagger context to access security definitions
	validationOpts := validation.NewOptions(opts...)
	swagger := validation.GetContextObject[Swagger](validationOpts)

	if swagger == nil || s.Map == nil {
		s.Valid = c.GetValid()
		return errs
	}

	// Validate each security requirement
	for name := range s.Keys() {
		scopes, _ := s.Get(name)

		// Check that the security scheme name exists in securityDefinitions
		secScheme, exists := swagger.SecurityDefinitions.Get(name)
		if !exists {
			errs = append(errs, validation.NewValidationError(
				validation.NewValueValidationError("security requirement '%s' does not match any security scheme in securityDefinitions", name),
				c.RootNode))
			continue
		}

		// For non-oauth2 security schemes, the array MUST be empty
		if secScheme.Type != SecuritySchemeTypeOAuth2 {
			if len(scopes) > 0 {
				errs = append(errs, validation.NewValidationError(
					validation.NewValueValidationError("security requirement '%s' must have empty scopes array for non-oauth2 security scheme (type=%s)", name, secScheme.Type),
					c.RootNode))
			}
		}
	}

	s.Valid = len(errs) == 0 && c.GetValid()
	return errs
}
