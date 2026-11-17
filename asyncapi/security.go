package asyncapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// SecuritySchemeType is the type of a security scheme.
type SecuritySchemeType string

const (
	SecuritySchemeTypeUserPassword         SecuritySchemeType = "userPassword"
	SecuritySchemeTypeAPIKey               SecuritySchemeType = "apiKey"
	SecuritySchemeTypeX509                 SecuritySchemeType = "X509"
	SecuritySchemeTypeSymmetricEncryption  SecuritySchemeType = "symmetricEncryption"
	SecuritySchemeTypeAsymmetricEncryption SecuritySchemeType = "asymmetricEncryption"
	SecuritySchemeTypeHTTPAPIKey           SecuritySchemeType = "httpApiKey"
	SecuritySchemeTypeHTTP                 SecuritySchemeType = "http"
	SecuritySchemeTypeOAuth2               SecuritySchemeType = "oauth2"
	SecuritySchemeTypeOpenIDConnect        SecuritySchemeType = "openIdConnect"
	SecuritySchemeTypePlain                SecuritySchemeType = "plain"
	SecuritySchemeTypeScramSha256          SecuritySchemeType = "scramSha256"
	SecuritySchemeTypeScramSha512          SecuritySchemeType = "scramSha512"
	SecuritySchemeTypeGSSAPI               SecuritySchemeType = "gssapi"
)

var securitySchemeTypes = []SecuritySchemeType{
	SecuritySchemeTypeUserPassword, SecuritySchemeTypeAPIKey, SecuritySchemeTypeX509, SecuritySchemeTypeSymmetricEncryption,
	SecuritySchemeTypeAsymmetricEncryption, SecuritySchemeTypeHTTPAPIKey, SecuritySchemeTypeHTTP, SecuritySchemeTypeOAuth2,
	SecuritySchemeTypeOpenIDConnect, SecuritySchemeTypePlain, SecuritySchemeTypeScramSha256, SecuritySchemeTypeScramSha512,
	SecuritySchemeTypeGSSAPI,
}

// SecurityScheme defines a security scheme that can be used by servers and operations.
type SecurityScheme struct {
	marshaller.Model[core.SecurityScheme]

	// Type is the type of the security scheme. Required.
	Type SecuritySchemeType
	// Description is a description of the security scheme. May contain CommonMark syntax.
	Description *string
	// Name is the name of the header, query or cookie parameter to be used. Required for `httpApiKey`.
	Name *string
	// In is the location of the API key. Required for `apiKey` (`user` or `password`) and `httpApiKey` (`query`, `header` or `cookie`).
	In *string
	// Scheme is the name of the HTTP Authorization scheme to be used. Required for `http`.
	Scheme *string
	// BearerFormat is a hint to the client to identify how the bearer token is formatted.
	BearerFormat *string
	// Flows contains configuration information for the supported flows. Required for `oauth2`.
	Flows *OAuthFlows
	// OpenIDConnectURL is the OpenID Connect URL to discover OAuth2 configuration values. Required for `openIdConnect`.
	OpenIDConnectURL *string
	// Scopes are the scopes needed to access the resource for `oauth2` and `openIdConnect`. Empty means no scopes are needed.
	Scopes []string
	// Extensions provides a list of extensions to the Security Scheme Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.SecurityScheme] = (*SecurityScheme)(nil)

// GetType returns the type, or an empty type if not set.
func (s *SecurityScheme) GetType() SecuritySchemeType {
	if s == nil {
		return ""
	}
	return s.Type
}

// GetDescription returns the description, or an empty string if not set.
func (s *SecurityScheme) GetDescription() string {
	if s == nil || s.Description == nil {
		return ""
	}
	return *s.Description
}

// GetName returns the name, or an empty string if not set.
func (s *SecurityScheme) GetName() string {
	if s == nil || s.Name == nil {
		return ""
	}
	return *s.Name
}

// GetIn returns the location of the API key, or an empty string if not set.
func (s *SecurityScheme) GetIn() string {
	if s == nil || s.In == nil {
		return ""
	}
	return *s.In
}

// GetScheme returns the HTTP Authorization scheme, or an empty string if not set.
func (s *SecurityScheme) GetScheme() string {
	if s == nil || s.Scheme == nil {
		return ""
	}
	return *s.Scheme
}

// GetBearerFormat returns the bearer format, or an empty string if not set.
func (s *SecurityScheme) GetBearerFormat() string {
	if s == nil || s.BearerFormat == nil {
		return ""
	}
	return *s.BearerFormat
}

// GetFlows returns the OAuth flows, or nil if not set.
func (s *SecurityScheme) GetFlows() *OAuthFlows {
	if s == nil {
		return nil
	}
	return s.Flows
}

// GetOpenIDConnectURL returns the OpenID Connect URL, or an empty string if not set.
func (s *SecurityScheme) GetOpenIDConnectURL() string {
	if s == nil || s.OpenIDConnectURL == nil {
		return ""
	}
	return *s.OpenIDConnectURL
}

// GetScopes returns the scopes, or nil if not set.
func (s *SecurityScheme) GetScopes() []string {
	if s == nil {
		return nil
	}
	return s.Scopes
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (s *SecurityScheme) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// Validate will validate the Security Scheme Object against the AsyncAPI Specification.
func (s *SecurityScheme) Validate(ctx context.Context, opts ...validation.Option) []error {
	if s == nil {
		return nil
	}

	c := s.GetCore()
	errs := []error{}

	required := func(missing bool, field string, node validation.ValueNodeGetter) {
		if missing {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("`securityScheme.%s` is required for type `%s`", field, s.Type), c, node))
		}
	}
	allowed := func(value *string, field string, node validation.ValueNodeGetter, values ...string) {
		if value == nil {
			required(true, field, node)
			return
		}
		for _, v := range values {
			if *value == v {
				return
			}
		}
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("securityScheme.%s must be one of [`%s`] for type `%s`, got `%s`", field, strings.Join(values, "`, `"), s.Type, *value), c, node))
	}

	switch s.Type {
	case "":
		errs = append(errs, requiredString(string(s.Type), c.Type.Present, c, c.Type, "securityScheme.type")...)
	case SecuritySchemeTypeAPIKey:
		allowed(s.In, "in", c.In, "user", "password")
	case SecuritySchemeTypeHTTPAPIKey:
		required(s.Name == nil || *s.Name == "", "name", c.Name)
		allowed(s.In, "in", c.In, "query", "header", "cookie")
	case SecuritySchemeTypeHTTP:
		required(s.Scheme == nil || *s.Scheme == "", "scheme", c.Scheme)
	case SecuritySchemeTypeOAuth2:
		required(s.Flows == nil, "flows", c.Flows)
	case SecuritySchemeTypeOpenIDConnect:
		required(s.OpenIDConnectURL == nil || *s.OpenIDConnectURL == "", "openIdConnectUrl", c.OpenIDConnectURL)
	case SecuritySchemeTypeUserPassword, SecuritySchemeTypeX509, SecuritySchemeTypeSymmetricEncryption, SecuritySchemeTypeAsymmetricEncryption,
		SecuritySchemeTypePlain, SecuritySchemeTypeScramSha256, SecuritySchemeTypeScramSha512, SecuritySchemeTypeGSSAPI:
	default:
		names := make([]string, 0, len(securitySchemeTypes))
		for _, t := range securitySchemeTypes {
			names = append(names, string(t))
		}
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("securityScheme.type must be one of [`%s`], got `%s`", strings.Join(names, "`, `"), s.Type), c, c.Type))
	}

	if s.OpenIDConnectURL != nil && *s.OpenIDConnectURL != "" {
		if err := validateAbsoluteURL(*s.OpenIDConnectURL, "securityScheme.openIdConnectUrl"); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, err, c, c.OpenIDConnectURL))
		}
	}
	if s.Flows != nil {
		errs = append(errs, s.Flows.Validate(ctx, opts...)...)
	}

	errs = append(errs, unknownPropertyErrors(c, "securityScheme")...)

	s.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// OAuthFlows allows configuration of the supported OAuth flows.
type OAuthFlows struct {
	marshaller.Model[core.OAuthFlows]

	// Implicit is the configuration for the OAuth Implicit flow.
	Implicit *OAuthFlow
	// Password is the configuration for the OAuth Resource Owner Protected Credentials flow.
	Password *OAuthFlow
	// ClientCredentials is the configuration for the OAuth Client Credentials flow.
	ClientCredentials *OAuthFlow
	// AuthorizationCode is the configuration for the OAuth Authorization Code flow.
	AuthorizationCode *OAuthFlow
	// Extensions provides a list of extensions to the OAuth Flows Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OAuthFlows] = (*OAuthFlows)(nil)

// GetImplicit returns the implicit flow, or nil if not set.
func (f *OAuthFlows) GetImplicit() *OAuthFlow {
	if f == nil {
		return nil
	}
	return f.Implicit
}

// GetPassword returns the password flow, or nil if not set.
func (f *OAuthFlows) GetPassword() *OAuthFlow {
	if f == nil {
		return nil
	}
	return f.Password
}

// GetClientCredentials returns the client credentials flow, or nil if not set.
func (f *OAuthFlows) GetClientCredentials() *OAuthFlow {
	if f == nil {
		return nil
	}
	return f.ClientCredentials
}

// GetAuthorizationCode returns the authorization code flow, or nil if not set.
func (f *OAuthFlows) GetAuthorizationCode() *OAuthFlow {
	if f == nil {
		return nil
	}
	return f.AuthorizationCode
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (f *OAuthFlows) GetExtensions() *extensions.Extensions {
	if f == nil || f.Extensions == nil {
		return extensions.New()
	}
	return f.Extensions
}

// Validate will validate the OAuth Flows Object against the AsyncAPI Specification.
func (f *OAuthFlows) Validate(ctx context.Context, opts ...validation.Option) []error {
	if f == nil {
		return nil
	}

	c := f.GetCore()
	errs := []error{}

	errs = append(errs, f.Implicit.validateFlow(ctx, "implicit", true, false, opts...)...)
	errs = append(errs, f.Password.validateFlow(ctx, "password", false, true, opts...)...)
	errs = append(errs, f.ClientCredentials.validateFlow(ctx, "clientCredentials", false, true, opts...)...)
	errs = append(errs, f.AuthorizationCode.validateFlow(ctx, "authorizationCode", true, true, opts...)...)
	errs = append(errs, unknownPropertyErrors(c, "oauthFlows")...)

	f.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// OAuthFlow is the configuration details for a supported OAuth flow.
type OAuthFlow struct {
	marshaller.Model[core.OAuthFlow]

	// AuthorizationURL is the authorization URL for this flow. Required for `implicit` and `authorizationCode`.
	AuthorizationURL *string
	// TokenURL is the token URL for this flow. Required for `password`, `clientCredentials` and `authorizationCode`.
	TokenURL *string
	// RefreshURL is the URL used for obtaining refresh tokens.
	RefreshURL *string
	// AvailableScopes are the available scopes for the OAuth2 security scheme, mapping scope names to descriptions. Required.
	AvailableScopes *sequencedmap.Map[string, string]
	// Extensions provides a list of extensions to the OAuth Flow Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OAuthFlow] = (*OAuthFlow)(nil)

// GetAuthorizationURL returns the authorization URL, or an empty string if not set.
func (f *OAuthFlow) GetAuthorizationURL() string {
	if f == nil || f.AuthorizationURL == nil {
		return ""
	}
	return *f.AuthorizationURL
}

// GetTokenURL returns the token URL, or an empty string if not set.
func (f *OAuthFlow) GetTokenURL() string {
	if f == nil || f.TokenURL == nil {
		return ""
	}
	return *f.TokenURL
}

// GetRefreshURL returns the refresh URL, or an empty string if not set.
func (f *OAuthFlow) GetRefreshURL() string {
	if f == nil || f.RefreshURL == nil {
		return ""
	}
	return *f.RefreshURL
}

// GetAvailableScopes returns the available scopes, or nil if not set.
func (f *OAuthFlow) GetAvailableScopes() *sequencedmap.Map[string, string] {
	if f == nil {
		return nil
	}
	return f.AvailableScopes
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (f *OAuthFlow) GetExtensions() *extensions.Extensions {
	if f == nil || f.Extensions == nil {
		return extensions.New()
	}
	return f.Extensions
}

// Validate will validate the OAuth Flow Object against the AsyncAPI Specification.
// The URLs a flow requires depend on its kind, which is checked when validating the parent OAuth Flows Object.
func (f *OAuthFlow) Validate(ctx context.Context, opts ...validation.Option) []error {
	return f.validateFlow(ctx, "", false, false, opts...)
}

func (f *OAuthFlow) validateFlow(_ context.Context, kind string, requireAuthorizationURL, requireTokenURL bool, _ ...validation.Option) []error {
	if f == nil {
		return nil
	}

	c := f.GetCore()
	errs := []error{}

	field := "oauthFlow"
	if kind != "" {
		field = "oauthFlows." + kind
	}

	checkURL := func(value *string, required bool, name string, node validation.ValueNodeGetter) {
		switch {
		case value == nil || *value == "":
			if required {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("`%s.%s` is required", field, name), c, node))
			}
		default:
			if err := validateAbsoluteURL(*value, field+"."+name); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, err, c, node))
			}
		}
	}

	checkURL(f.AuthorizationURL, requireAuthorizationURL, "authorizationUrl", c.AuthorizationURL)
	checkURL(f.TokenURL, requireTokenURL, "tokenUrl", c.TokenURL)
	checkURL(f.RefreshURL, false, "refreshUrl", c.RefreshURL)

	if f.AvailableScopes == nil && c.GetRootNode() == nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`"+field+".availableScopes` is required"), c, c.AvailableScopes))
	}

	errs = append(errs, unknownPropertyErrors(c, field)...)

	f.Valid = noErrors(errs) && coreValid(c)

	return errs
}
