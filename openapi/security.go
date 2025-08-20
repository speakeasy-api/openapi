package openapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

type SecuritySchemaType string

var _ fmt.Stringer = (*SecuritySchemaType)(nil)

func (s SecuritySchemaType) String() string {
	return string(s)
}

const (
	SecuritySchemeTypeAPIKey        SecuritySchemaType = "apiKey"
	SecuritySchemeTypeHTTP          SecuritySchemaType = "http"
	SecuritySchemeTypeMutualTLS     SecuritySchemaType = "mutualTLS"
	SecuritySchemeTypeOAuth2        SecuritySchemaType = "oauth2"
	SecuritySchemeTypeOpenIDConnect SecuritySchemaType = "openIdConnect"
)

type SecuritySchemeIn string

var _ fmt.Stringer = (*SecuritySchemeIn)(nil)

func (s SecuritySchemeIn) String() string {
	return string(s)
}

const (
	SecuritySchemeInHeader SecuritySchemeIn = "header"
	SecuritySchemeInQuery  SecuritySchemeIn = "query"
	SecuritySchemeInCookie SecuritySchemeIn = "cookie"
)

type SecurityScheme struct {
	marshaller.Model[core.SecurityScheme]

	// Type represents the type of the security scheme.
	Type SecuritySchemaType
	// Description is a description of the security scheme.
	Description *string
	// Name is the name of the header, query or cookie parameter to be used.
	Name *string
	// In is the location of the API key.
	In *SecuritySchemeIn
	// Scheme is the name of the HTTP Authorization scheme to be used in the Authorization header.
	Scheme *string
	// BearerFormat is the name of the HTTP Authorization scheme to be used in the Authorization header.
	BearerFormat *string
	// Flows is a map of the different flows supported by the OAuth2 security scheme.
	Flows *OAuthFlows
	// OpenIdConnectUrl is a URL to discover OAuth2 configuration values.
	OpenIdConnectUrl *string
	// Extensions provides a list of extensions to the SecurityScheme object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.SecurityScheme] = (*SecurityScheme)(nil)

// GetType returns the value of the Type field. Returns empty SecuritySchemaType if not set.
func (s *SecurityScheme) GetType() SecuritySchemaType {
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

// GetIn returns the value of the In field. Returns empty SecuritySchemeIn if not set.
func (s *SecurityScheme) GetIn() SecuritySchemeIn {
	if s == nil || s.In == nil {
		return ""
	}
	return *s.In
}

// GetScheme returns the value of the Scheme field. Returns empty string if not set.
func (s *SecurityScheme) GetScheme() string {
	if s == nil || s.Scheme == nil {
		return ""
	}
	return *s.Scheme
}

// GetBearerFormat returns the value of the BearerFormat field. Returns empty string if not set.
func (s *SecurityScheme) GetBearerFormat() string {
	if s == nil || s.BearerFormat == nil {
		return ""
	}
	return *s.BearerFormat
}

// GetFlows returns the value of the Flows field. Returns nil if not set.
func (s *SecurityScheme) GetFlows() *OAuthFlows {
	if s == nil {
		return nil
	}
	return s.Flows
}

// GetOpenIdConnectUrl returns the value of the OpenIdConnectUrl field. Returns empty string if not set.
func (s *SecurityScheme) GetOpenIdConnectUrl() string {
	if s == nil || s.OpenIdConnectUrl == nil {
		return ""
	}
	return *s.OpenIdConnectUrl
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (s *SecurityScheme) GetExtensions() *extensions.Extensions {
	if s == nil || s.Extensions == nil {
		return extensions.New()
	}
	return s.Extensions
}

// Validate will validate the SecurityScheme object against the OpenAPI Specification.
func (s *SecurityScheme) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := s.GetCore()
	errs := []error{}

	if core.Type.Present {
		if s.Type == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme field type is required"), core, core.Type))
		} else {
			switch s.Type {
			case SecuritySchemeTypeAPIKey:
				if !core.Name.Present || *s.Name == "" {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme field name is required for type=apiKey"), core, core.Name))
				}
				if !core.In.Present || *s.In == "" {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme field in is required for type=apiKey"), core, core.In))
				} else {
					switch *s.In {
					case SecuritySchemeInHeader:
					case SecuritySchemeInQuery:
					case SecuritySchemeInCookie:
					default:
						errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme field in must be one of [%s] for type=apiKey", strings.Join([]string{string(SecuritySchemeInHeader), string(SecuritySchemeInQuery), string(SecuritySchemeInCookie)}, ", ")), core, core.In))
					}
				}
			case SecuritySchemeTypeHTTP:
				if !core.Scheme.Present || *s.Scheme == "" {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme field scheme is required for type=http"), core, core.Scheme))
				}
			case SecuritySchemeTypeMutualTLS:
			case SecuritySchemeTypeOAuth2:
				if !core.Flows.Present || s.Flows == nil {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme field flows is required for type=oauth2"), core, core.Flows))
				} else {
					errs = append(errs, s.Flows.Validate(ctx, opts...)...)
				}
			case SecuritySchemeTypeOpenIDConnect:
				if !core.OpenIdConnectUrl.Present || *s.OpenIdConnectUrl == "" {
					errs = append(errs, validation.NewValueError(validation.NewMissingValueError("securityScheme field openIdConnectUrl is required for type=openIdConnect"), core, core.OpenIdConnectUrl))
				}
			default:
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("securityScheme field type must be one of [%s]", strings.Join([]string{string(SecuritySchemeTypeAPIKey), string(SecuritySchemeTypeHTTP), string(SecuritySchemeTypeMutualTLS), string(SecuritySchemeTypeOAuth2), string(SecuritySchemeTypeOpenIDConnect)}, ", ")), core, core.Type))
			}
		}
	}

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// SecurityRequirement represents a security requirement for an API or operation.
// Each name in the map represents a security scheme that can be used to secure the API or operation.
// If the security scheme is of type "oauth2" or "openIdConnect", then the value is a list of scope names required by the operation.
// SecurityRequirement embeds sequencedmap.Map[string, []string] so all map operations are supported.
type SecurityRequirement struct {
	marshaller.Model[core.SecurityRequirement]
	sequencedmap.Map[string, []string]
}

var _ interfaces.Model[core.SecurityRequirement] = (*SecurityRequirement)(nil)

// NewSecurityRequirement creates a new SecurityRequirement object with the embedded map initialized.
func NewSecurityRequirement(elems ...*sequencedmap.Element[string, []string]) *SecurityRequirement {
	return &SecurityRequirement{
		Map: *sequencedmap.New(elems...),
	}
}

func (s *SecurityRequirement) Populate(source any) error {
	var coreReq *core.SecurityRequirement
	switch v := source.(type) {
	case *core.SecurityRequirement:
		coreReq = v
	case core.SecurityRequirement:
		coreReq = &v
	default:
		return fmt.Errorf("expected *core.SecurityRequirement or core.SecurityRequirement, got %T", source)
	}

	if !s.IsInitialized() {
		s.Map = *sequencedmap.New[string, []string]()
	}

	// Convert from core map to regular map
	if coreReq.IsInitialized() {
		for key, elem := range coreReq.All() {
			// elem.Value is marshaller.Node[[]string], need to get the actual value
			if elem.Present && elem.Value != nil {
				strSlice := make([]string, len(elem.Value))
				for i, v := range elem.Value {
					strSlice[i] = v.Value
				}
				s.Set(key, strSlice)
			}
		}
	}

	s.SetCore(coreReq)

	return nil
}

// Validate validates the SecurityRequirement object according to the OpenAPI specification.
func (s *SecurityRequirement) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := s.GetCore()
	errs := []error{}

	o := validation.NewOptions(opts...)

	openapi := validation.GetContextObject[OpenAPI](o)
	if openapi == nil {
		panic("OpenAPI is required")
	}

	for securityScheme := range s.Keys() {
		if openapi.Components == nil || !openapi.Components.SecuritySchemes.Has(securityScheme) {
			errs = append(errs, validation.NewMapKeyError(validation.NewValueValidationError("securityRequirement scheme %s is not defined in components.securitySchemes", securityScheme), core, core, securityScheme))
		}
	}

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// OAuthFlows represents the configuration of the supported OAuth flows.
type OAuthFlows struct {
	marshaller.Model[core.OAuthFlows]

	// Implicit represents configuration fields for the OAuth2 Implicit flow.
	Implicit *OAuthFlow
	// Password represents configuration fields for the OAuth2 Resource Owner Password flow.
	Password *OAuthFlow
	// ClientCredentials represents configuration fields for the OAuth2 Client Credentials flow.
	ClientCredentials *OAuthFlow
	// AuthorizationCode represents configuration fields for the OAuth2 Authorization Code flow.
	AuthorizationCode *OAuthFlow

	// Extensions provides a list of extensions to the OAuthFlows object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OAuthFlows] = (*OAuthFlows)(nil)

type OAuthFlowType string

const (
	OAuthFlowTypeImplicit          OAuthFlowType = "implicit"
	OAuthFlowTypePassword          OAuthFlowType = "password"
	OAuthFlowTypeClientCredentials OAuthFlowType = "clientCredentials"
	OAuthFlowTypeAuthorizationCode OAuthFlowType = "authorizationCode"
)

// GetImplicit returns the value of the Implicit field. Returns nil if not set.
func (o *OAuthFlows) GetImplicit() *OAuthFlow {
	if o == nil {
		return nil
	}
	return o.Implicit
}

// GetPassword returns the value of the Password field. Returns nil if not set.
func (o *OAuthFlows) GetPassword() *OAuthFlow {
	if o == nil {
		return nil
	}
	return o.Password
}

// GetClientCredentials returns the value of the ClientCredentials field. Returns nil if not set.
func (o *OAuthFlows) GetClientCredentials() *OAuthFlow {
	if o == nil {
		return nil
	}
	return o.ClientCredentials
}

// GetAuthorizationCode returns the value of the AuthorizationCode field. Returns nil if not set.
func (o *OAuthFlows) GetAuthorizationCode() *OAuthFlow {
	if o == nil {
		return nil
	}
	return o.AuthorizationCode
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (o *OAuthFlows) GetExtensions() *extensions.Extensions {
	if o == nil || o.Extensions == nil {
		return extensions.New()
	}
	return o.Extensions
}

// Validate will validate the OAuthFlows object against the OpenAPI Specification.
func (o *OAuthFlows) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := o.GetCore()
	errs := []error{}

	if o.Implicit != nil {
		errs = append(errs, o.Implicit.Validate(ctx, append(opts, validation.WithContextObject(pointer.From(OAuthFlowTypeImplicit)))...)...)
	}
	if o.Password != nil {
		errs = append(errs, o.Password.Validate(ctx, append(opts, validation.WithContextObject(pointer.From(OAuthFlowTypePassword)))...)...)
	}
	if o.ClientCredentials != nil {
		errs = append(errs, o.ClientCredentials.Validate(ctx, append(opts, validation.WithContextObject(pointer.From(OAuthFlowTypeClientCredentials)))...)...)
	}
	if o.AuthorizationCode != nil {
		errs = append(errs, o.AuthorizationCode.Validate(ctx, append(opts, validation.WithContextObject(pointer.From(OAuthFlowTypeAuthorizationCode)))...)...)
	}

	o.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// OAuthFlow represents the configuration details for a supported OAuth flow.
type OAuthFlow struct {
	marshaller.Model[core.OAuthFlow]

	// AuthorizationUrl is a URL to be used for obtaining authorization.
	AuthorizationURL *string
	// TokenUrl is a URL to be used for obtaining access tokens.
	TokenURL *string
	// RefreshUrl is a URL to be used for refreshing access tokens.
	RefreshURL *string
	// Scopes is a map between the name of the scope and a short description of the scope.
	Scopes *sequencedmap.Map[string, string]
	// Extensions provides a list of extensions to the OAuthFlow object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OAuthFlow] = (*OAuthFlow)(nil)

// GetAuthorizationURL returns the value of the AuthorizationURL field. Returns empty string if not set.
func (o *OAuthFlow) GetAuthorizationURL() string {
	if o == nil || o.AuthorizationURL == nil {
		return ""
	}
	return *o.AuthorizationURL
}

// GetTokenURL returns the value of the TokenURL field. Returns empty string if not set.
func (o *OAuthFlow) GetTokenURL() string {
	if o == nil || o.TokenURL == nil {
		return ""
	}
	return *o.TokenURL
}

// GetRefreshURL returns the value of the RefreshURL field. Returns empty string if not set.
func (o *OAuthFlow) GetRefreshURL() string {
	if o == nil || o.RefreshURL == nil {
		return ""
	}
	return *o.RefreshURL
}

// GetScopes returns the value of the Scopes field. Returns nil if not set.
func (o *OAuthFlow) GetScopes() *sequencedmap.Map[string, string] {
	if o == nil {
		return nil
	}
	return o.Scopes
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (o *OAuthFlow) GetExtensions() *extensions.Extensions {
	if o == nil || o.Extensions == nil {
		return extensions.New()
	}
	return o.Extensions
}

// Validate will validate the OAuthFlow object against the OpenAPI Specification.
func (o *OAuthFlow) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := o.GetCore()
	errs := []error{}

	op := validation.NewOptions(opts...)

	oAuthFlowType := validation.GetContextObject[OAuthFlowType](op)
	if oAuthFlowType == nil {
		panic("OAuthFlowType is required")
	}

	switch *oAuthFlowType {
	case OAuthFlowTypeImplicit:
		if !core.AuthorizationURL.Present || *o.AuthorizationURL == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("oAuthFlow field authorizationUrl is required for type=implicit"), core, core.AuthorizationURL))
		} else {
			if _, err := url.Parse(*o.AuthorizationURL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("oAuthFlow field authorizationUrl is not a valid uri: %s", err), core, core.AuthorizationURL))
			}
		}
	case OAuthFlowTypePassword:
		if !core.TokenURL.Present || *o.TokenURL == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("oAuthFlow field tokenUrl is required for type=password"), core, core.TokenURL))
		} else {
			if _, err := url.Parse(*o.TokenURL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("oAuthFlow field tokenUrl is not a valid uri: %s", err), core, core.TokenURL))
			}
		}
	case OAuthFlowTypeClientCredentials:
		if !core.TokenURL.Present || *o.TokenURL == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("oAuthFlow field tokenUrl is required for type=clientCredentials"), core, core.TokenURL))
		} else {
			if _, err := url.Parse(*o.TokenURL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("oAuthFlow field tokenUrl is not a valid uri: %s", err), core, core.TokenURL))
			}
		}
	case OAuthFlowTypeAuthorizationCode:
		if !core.AuthorizationURL.Present || *o.AuthorizationURL == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("oAuthFlow field authorizationUrl is required for type=authorizationCode"), core, core.AuthorizationURL))
		} else {
			if _, err := url.Parse(*o.AuthorizationURL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("oAuthFlow field authorizationUrl is not a valid uri: %s", err), core, core.AuthorizationURL))
			}
		}
		if !core.TokenURL.Present || *o.TokenURL == "" {
			errs = append(errs, validation.NewValueError(validation.NewMissingValueError("oAuthFlow field tokenUrl is required for type=authorizationCode"), core, core.TokenURL))
		} else {
			if _, err := url.Parse(*o.TokenURL); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("oAuthFlow field tokenUrl is not a valid uri: %s", err), core, core.TokenURL))
			}
		}
	}

	if core.RefreshURL.Present {
		if _, err := url.Parse(*o.RefreshURL); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("oAuthFlow field refreshUrl is not a valid uri: %s", err), core, core.RefreshURL))
		}
	}

	if !core.Scopes.Present {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("oAuthFlow field scopes is required (empty map is allowed)"), core, core.Scopes))
	}

	o.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
