package openapi_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/require"
)

func TestSecurityScheme_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_api_key_header",
			yml: `
type: apiKey
name: X-API-Key
in: header
description: API key authentication
`,
		},
		{
			name: "valid_api_key_query",
			yml: `
type: apiKey
name: api_key
in: query
`,
		},
		{
			name: "valid_api_key_cookie",
			yml: `
type: apiKey
name: sessionId
in: cookie
`,
		},
		{
			name: "valid_http_basic",
			yml: `
type: http
scheme: basic
description: Basic authentication
`,
		},
		{
			name: "valid_http_bearer",
			yml: `
type: http
scheme: bearer
bearerFormat: JWT
`,
		},
		{
			name: "valid_mutual_tls",
			yml: `
type: mutualTLS
description: Mutual TLS authentication
`,
		},
		{
			name: "valid_oauth2",
			yml: `
type: oauth2
flows:
  authorizationCode:
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes:
      read: Read access
      write: Write access
`,
		},
		{
			name: "valid_openid_connect",
			yml: `
type: openIdConnect
openIdConnectUrl: https://example.com/.well-known/openid_configuration
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
type: http
scheme: bearer
x-custom: value
x-another: 123
`,
		},
		{
			name: "valid_oauth2_with_metadata_url",
			yml: `
type: oauth2
flows:
  authorizationCode:
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes:
      read: Read access
oauth2MetadataUrl: https://example.com/.well-known/oauth-authorization-server
`,
		},
		{
			name: "valid_deprecated_scheme",
			yml: `
type: http
scheme: bearer
deprecated: true
`,
		},
		{
			name: "valid_oauth2_device_authorization",
			yml: `
type: oauth2
flows:
  deviceAuthorization:
    deviceAuthorizationUrl: https://example.com/oauth/device_authorization
    tokenUrl: https://example.com/oauth/token
    scopes:
      read: Read access
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityScheme openapi.SecurityScheme

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityScheme)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := securityScheme.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSecurityScheme_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing_type",
			yml: `
description: Some security scheme
`,
			wantErrs: []string{"[2:1] error validation-required-field securityScheme.type is required"},
		},
		{
			name: "invalid_type",
			yml: `
type: invalid
`,
			wantErrs: []string{"type must be one of"},
		},
		{
			name: "api_key_missing_name",
			yml: `
type: apiKey
in: header
`,
			wantErrs: []string{"name is required for type=apiKey"},
		},
		{
			name: "api_key_missing_in",
			yml: `
type: apiKey
name: X-API-Key
`,
			wantErrs: []string{"in is required for type=apiKey"},
		},
		{
			name: "api_key_invalid_in",
			yml: `
type: apiKey
name: X-API-Key
in: invalid
`,
			wantErrs: []string{"in must be one of"},
		},
		{
			name: "http_missing_scheme",
			yml: `
type: http
`,
			wantErrs: []string{"scheme is required for type=http"},
		},
		{
			name: "oauth2_missing_flows",
			yml: `
type: oauth2
`,
			wantErrs: []string{"flows is required for type=oauth2"},
		},
		{
			name: "openid_missing_url",
			yml: `
type: openIdConnect
`,
			wantErrs: []string{"openIdConnectUrl is required for type=openIdConnect"},
		},
		{
			name: "oauth2_invalid_metadata_url",
			yml: `
type: oauth2
flows:
  authorizationCode:
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes:
      read: Read access
oauth2MetadataUrl: ://invalid-url
`,
			wantErrs: []string{"oauth2MetadataUrl is not a valid uri"},
		},
		{
			name: "oauth2_flow_invalid_authorization_url",
			yml: `
type: oauth2
flows:
  implicit:
    authorizationUrl: http:// blah.
    scopes:
      read: Read access
`,
			wantErrs: []string{"authorizationUrl is not a valid uri"},
		},
		{
			name: "openid_invalid_url",
			yml: `
type: openIdConnect
openIdConnectUrl: http:// blah.
`,
			wantErrs: []string{"openIdConnectUrl is not a valid uri"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityScheme openapi.SecurityScheme

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityScheme)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := securityScheme.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			// Check that all expected errors are present
			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}

func TestSecurityScheme_Validate_UnusedProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yml          string
		wantWarnings []string
	}{
		{
			name: "http_with_in_property",
			yml: `
type: http
scheme: bearer
in: header
`,
			wantWarnings: []string{"in is not used for type=http"},
		},
		{
			name: "http_with_name_property",
			yml: `
type: http
scheme: bearer
name: X-API-Key
`,
			wantWarnings: []string{"name is not used for type=http"},
		},
		{
			name: "http_with_flows_property",
			yml: `
type: http
scheme: bearer
flows:
  implicit:
    authorizationUrl: https://example.com/oauth/authorize
    scopes: {}
`,
			wantWarnings: []string{"flows is not used for type=http"},
		},
		{
			name: "apiKey_with_scheme_property",
			yml: `
type: apiKey
name: X-API-Key
in: header
scheme: bearer
`,
			wantWarnings: []string{"scheme is not used for type=apiKey"},
		},
		{
			name: "apiKey_with_bearerFormat_property",
			yml: `
type: apiKey
name: X-API-Key
in: header
bearerFormat: JWT
`,
			wantWarnings: []string{"bearerFormat is not used for type=apiKey"},
		},
		{
			name: "apiKey_with_flows_property",
			yml: `
type: apiKey
name: X-API-Key
in: header
flows:
  implicit:
    authorizationUrl: https://example.com/oauth/authorize
    scopes: {}
`,
			wantWarnings: []string{"flows is not used for type=apiKey"},
		},
		{
			name: "mutualTLS_with_scheme_property",
			yml: `
type: mutualTLS
scheme: bearer
`,
			wantWarnings: []string{"scheme is not used for type=mutualTLS"},
		},
		{
			name: "mutualTLS_with_name_and_in_properties",
			yml: `
type: mutualTLS
name: X-API-Key
in: header
`,
			wantWarnings: []string{
				"name is not used for type=mutualTLS",
				"in is not used for type=mutualTLS",
			},
		},
		{
			name: "oauth2_with_scheme_property",
			yml: `
type: oauth2
flows:
  authorizationCode:
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes: {}
scheme: bearer
`,
			wantWarnings: []string{"scheme is not used for type=oauth2"},
		},
		{
			name: "oauth2_with_name_and_in_properties",
			yml: `
type: oauth2
flows:
  authorizationCode:
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes: {}
name: X-API-Key
in: header
`,
			wantWarnings: []string{
				"name is not used for type=oauth2",
				"in is not used for type=oauth2",
			},
		},
		{
			name: "openIdConnect_with_scheme_property",
			yml: `
type: openIdConnect
openIdConnectUrl: https://example.com/.well-known/openid-configuration
scheme: bearer
`,
			wantWarnings: []string{"scheme is not used for type=openIdConnect"},
		},
		{
			name: "openIdConnect_with_flows_property",
			yml: `
type: openIdConnect
openIdConnectUrl: https://example.com/.well-known/openid-configuration
flows:
  implicit:
    authorizationUrl: https://example.com/oauth/authorize
    scopes: {}
`,
			wantWarnings: []string{"flows is not used for type=openIdConnect"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityScheme openapi.SecurityScheme

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityScheme)
			require.NoError(t, err)

			errs := securityScheme.Validate(t.Context())

			// Combine unmarshalling and validation errors
			validationErrs = append(validationErrs, errs...)

			// Extract warnings (severity = warning)
			var warnings []error
			for _, e := range validationErrs {
				var verr *validation.Error
				if errors.As(e, &verr) && verr.Severity == validation.SeverityWarning {
					warnings = append(warnings, e)
				}
			}

			require.NotEmpty(t, warnings, "Expected validation warnings")
			require.Len(t, warnings, len(tt.wantWarnings), "Expected %d warnings, got %d: %v", len(tt.wantWarnings), len(warnings), warnings)

			// Check that all expected warnings are present
			for _, wantWarning := range tt.wantWarnings {
				found := false
				for _, gotWarning := range warnings {
					if gotWarning != nil && strings.Contains(gotWarning.Error(), wantWarning) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected warning containing '%s' not found in: %v", wantWarning, warnings)
			}
		})
	}
}

func TestSecurityRequirement_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_api_key_requirement",
			yml: `
api_key: []
`,
		},
		{
			name: "valid_oauth2_requirement",
			yml: `
oauth2:
  - read
  - write
`,
		},
		{
			name: "valid_multiple_requirements",
			yml: `
api_key: []
oauth2:
  - read
`,
		},
		{
			name: "valid_empty_requirement",
			yml:  `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityRequirement openapi.SecurityRequirement

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityRequirement)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create a mock OpenAPI document with security schemes
			openAPIDoc := &openapi.OpenAPI{
				Components: &openapi.Components{
					SecuritySchemes: sequencedmap.New(
						sequencedmap.NewElem("api_key", &openapi.ReferencedSecurityScheme{
							Object: &openapi.SecurityScheme{Type: openapi.SecuritySchemeTypeAPIKey},
						}),
						sequencedmap.NewElem("oauth2", &openapi.ReferencedSecurityScheme{
							Object: &openapi.SecurityScheme{Type: openapi.SecuritySchemeTypeOAuth2},
						}),
					),
				},
			}

			errs := securityRequirement.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSecurityRequirement_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yml         string
		expectedErr string
	}{
		{
			name: "undefined_security_scheme",
			yml: `
"://invalid uri": []
`,
			expectedErr: "securityRequirement scheme ://invalid uri is not defined in components.securitySchemes and is not a valid URI reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityRequirement openapi.SecurityRequirement

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityRequirement)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create a mock OpenAPI document with empty security schemes
			openAPIDoc := &openapi.OpenAPI{
				Components: &openapi.Components{
					SecuritySchemes: sequencedmap.New[string, *openapi.ReferencedSecurityScheme](),
				},
			}

			errs := securityRequirement.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.NotEmpty(t, errs, "Expected validation errors")
			require.Contains(t, errs[0].Error(), tt.expectedErr)
		})
	}
}

func TestSecurityRequirement_Validate_URIReferences_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "uri_reference_absolute",
			yml: `
https://example.com/security/schemes/oauth2: []
`,
		},
		{
			name: "uri_reference_relative",
			yml: `
./security/oauth2: []
`,
		},
		{
			name: "uri_reference_with_fragment",
			yml: `
https://example.com/api#/components/securitySchemes/oauth2: []
`,
		},
		{
			name: "mixed_component_and_uri",
			yml: `
api_key: []
https://example.com/security/oauth2:
  - read
  - write
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityRequirement openapi.SecurityRequirement

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityRequirement)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create a mock OpenAPI document with one known component
			openAPIDoc := &openapi.OpenAPI{
				Components: &openapi.Components{
					SecuritySchemes: sequencedmap.New(
						sequencedmap.NewElem("api_key", &openapi.ReferencedSecurityScheme{
							Object: &openapi.SecurityScheme{Type: openapi.SecuritySchemeTypeAPIKey},
						}),
					),
				},
			}

			errs := securityRequirement.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
			require.Empty(t, errs, "Expected no validation errors for valid URI references")
		})
	}
}

func TestSecurityRequirement_Validate_ComponentNamePrecedence_Success(t *testing.T) {
	t.Parallel()

	// Test that component names take precedence over URI interpretation
	// per spec: "Property names that are identical to a component name under
	// the Components Object MUST be treated as a component name"
	yml := `
foo: []
`

	var securityRequirement openapi.SecurityRequirement

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &securityRequirement)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Create a mock OpenAPI document where "foo" is both a valid URI segment
	// and a component name - component name should take precedence
	openAPIDoc := &openapi.OpenAPI{
		Components: &openapi.Components{
			SecuritySchemes: sequencedmap.New(
				sequencedmap.NewElem("foo", &openapi.ReferencedSecurityScheme{
					Object: &openapi.SecurityScheme{Type: openapi.SecuritySchemeTypeHTTP},
				}),
			),
		},
	}

	errs := securityRequirement.Validate(t.Context(), validation.WithContextObject(openAPIDoc))
	require.Empty(t, errs, "Component name 'foo' should take precedence over URI interpretation")
}

func TestOAuthFlows_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_implicit_flow",
			yml: `
implicit:
  authorizationUrl: https://example.com/oauth/authorize
  scopes:
    read: Read access
    write: Write access
`,
		},
		{
			name: "valid_password_flow",
			yml: `
password:
  tokenUrl: https://example.com/oauth/token
  scopes:
    read: Read access
`,
		},
		{
			name: "valid_client_credentials_flow",
			yml: `
clientCredentials:
  tokenUrl: https://example.com/oauth/token
  scopes:
    admin: Admin access
`,
		},
		{
			name: "valid_authorization_code_flow",
			yml: `
authorizationCode:
  authorizationUrl: https://example.com/oauth/authorize
  tokenUrl: https://example.com/oauth/token
  refreshUrl: https://example.com/oauth/refresh
  scopes:
    read: Read access
    write: Write access
`,
		},
		{
			name: "valid_multiple_flows",
			yml: `
implicit:
  authorizationUrl: https://example.com/oauth/authorize
  scopes:
    read: Read access
authorizationCode:
  authorizationUrl: https://example.com/oauth/authorize
  tokenUrl: https://example.com/oauth/token
  scopes:
    read: Read access
    write: Write access
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
implicit:
  authorizationUrl: https://example.com/oauth/authorize
  scopes:
    read: Read access
x-custom: value
`,
		},
		{
			name: "valid_device_authorization_flow",
			yml: `
deviceAuthorization:
  deviceAuthorizationUrl: https://example.com/oauth/device_authorization
  tokenUrl: https://example.com/oauth/token
  scopes:
    read: Read access
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var oauthFlows openapi.OAuthFlows

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &oauthFlows)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := oauthFlows.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestOAuthFlow_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		flowType openapi.OAuthFlowType
	}{
		{
			name: "valid_implicit_flow",
			yml: `
authorizationUrl: https://example.com/oauth/authorize
scopes:
  read: Read access
  write: Write access
`,
			flowType: openapi.OAuthFlowTypeImplicit,
		},
		{
			name: "valid_password_flow",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
`,
			flowType: openapi.OAuthFlowTypePassword,
		},
		{
			name: "valid_client_credentials_flow",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  admin: Admin access
`,
			flowType: openapi.OAuthFlowTypeClientCredentials,
		},
		{
			name: "valid_authorization_code_flow",
			yml: `
authorizationUrl: https://example.com/oauth/authorize
tokenUrl: https://example.com/oauth/token
refreshUrl: https://example.com/oauth/refresh
scopes:
  read: Read access
  write: Write access
`,
			flowType: openapi.OAuthFlowTypeAuthorizationCode,
		},
		{
			name: "valid_empty_scopes",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes: {}
`,
			flowType: openapi.OAuthFlowTypePassword,
		},
		{
			name: "valid_with_extensions",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
x-custom: value
`,
			flowType: openapi.OAuthFlowTypePassword,
		},
		{
			name: "valid_device_authorization_flow",
			yml: `
deviceAuthorizationUrl: https://example.com/oauth/device_authorization
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
`,
			flowType: openapi.OAuthFlowTypeDeviceAuthorization,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var oauthFlow openapi.OAuthFlow

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &oauthFlow)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := oauthFlow.Validate(t.Context(), validation.WithContextObject(&tt.flowType))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestOAuthFlow_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yml         string
		flowType    openapi.OAuthFlowType
		expectedErr string
	}{
		{
			name: "implicit_missing_authorization_url",
			yml: `
scopes:
  read: Read access
`,
			flowType:    openapi.OAuthFlowTypeImplicit,
			expectedErr: "authorizationUrl is required for type=implicit",
		},
		{
			name: "password_missing_token_url",
			yml: `scopes:
  read: Read access`,
			flowType:    openapi.OAuthFlowTypePassword,
			expectedErr: "tokenUrl is required for type=password",
		},
		{
			name: "client_credentials_missing_token_url",
			yml: `
scopes:
  admin: Admin access
`,
			flowType:    openapi.OAuthFlowTypeClientCredentials,
			expectedErr: "tokenUrl is required for type=clientCredentials",
		},
		{
			name: "authorization_code_missing_authorization_url",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
`,
			flowType:    openapi.OAuthFlowTypeAuthorizationCode,
			expectedErr: "authorizationUrl is required for type=authorizationCode",
		},
		{
			name: "authorization_code_missing_token_url",
			yml: `
authorizationUrl: https://example.com/oauth/authorize
scopes:
  read: Read access
`,
			flowType:    openapi.OAuthFlowTypeAuthorizationCode,
			expectedErr: "tokenUrl is required for type=authorizationCode",
		},
		{
			name: "missing_scopes",
			yml: `
tokenUrl: https://example.com/oauth/token
`,
			flowType:    openapi.OAuthFlowTypePassword,
			expectedErr: "scopes is required (empty map is allowed)",
		},
		{
			name: "device_authorization_missing_device_authorization_url",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
`,
			flowType:    openapi.OAuthFlowTypeDeviceAuthorization,
			expectedErr: "deviceAuthorizationUrl is required for type=deviceAuthorization",
		},
		{
			name: "device_authorization_missing_token_url",
			yml: `
deviceAuthorizationUrl: https://example.com/oauth/device_authorization
scopes:
  read: Read access
`,
			flowType:    openapi.OAuthFlowTypeDeviceAuthorization,
			expectedErr: "tokenUrl is required for type=deviceAuthorization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var oauthFlow openapi.OAuthFlow

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &oauthFlow)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := oauthFlow.Validate(t.Context(), validation.WithContextObject(&tt.flowType))
			require.NotEmpty(t, errs, "Expected validation errors")
			require.Contains(t, errs[0].Error(), tt.expectedErr)
		})
	}
}

func TestSecurityScheme_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `
type: oauth2
description: OAuth2 authentication
flows:
  authorizationCode:
    authorizationUrl: https://example.com/auth
    tokenUrl: https://example.com/token
    scopes:
      read: Read access
openIdConnectUrl: https://example.com/.well-known/openid
oauth2MetadataUrl: https://example.com/.well-known/oauth
deprecated: true
x-custom: value
`
	var securityScheme openapi.SecurityScheme

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &securityScheme)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, openapi.SecuritySchemeTypeOAuth2, securityScheme.GetType(), "GetType should return correct value")
	require.Equal(t, "OAuth2 authentication", securityScheme.GetDescription(), "GetDescription should return correct value")
	require.NotNil(t, securityScheme.GetFlows(), "GetFlows should return non-nil")
	require.Equal(t, "https://example.com/.well-known/openid", securityScheme.GetOpenIdConnectUrl(), "GetOpenIdConnectUrl should return correct value")
	require.Equal(t, "https://example.com/.well-known/oauth", securityScheme.GetOAuth2MetadataUrl(), "GetOAuth2MetadataUrl should return correct value")
	require.True(t, securityScheme.GetDeprecated(), "GetDeprecated should return true")
	require.NotNil(t, securityScheme.GetExtensions(), "GetExtensions should return non-nil")
}

func TestSecurityScheme_Getters_HTTPScheme(t *testing.T) {
	t.Parallel()

	yml := `
type: http
scheme: bearer
bearerFormat: JWT
`
	var securityScheme openapi.SecurityScheme

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &securityScheme)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "bearer", securityScheme.GetScheme(), "GetScheme should return correct value")
	require.Equal(t, "JWT", securityScheme.GetBearerFormat(), "GetBearerFormat should return correct value")
}

func TestSecurityScheme_Getters_Nil(t *testing.T) {
	t.Parallel()

	var securityScheme *openapi.SecurityScheme

	require.Empty(t, securityScheme.GetType(), "GetType should return empty for nil")
	require.Empty(t, securityScheme.GetDescription(), "GetDescription should return empty string for nil")
	require.Empty(t, securityScheme.GetScheme(), "GetScheme should return empty string for nil")
	require.Empty(t, securityScheme.GetBearerFormat(), "GetBearerFormat should return empty string for nil")
	require.Nil(t, securityScheme.GetFlows(), "GetFlows should return nil for nil scheme")
	require.Empty(t, securityScheme.GetOpenIdConnectUrl(), "GetOpenIdConnectUrl should return empty string for nil")
	require.Empty(t, securityScheme.GetOAuth2MetadataUrl(), "GetOAuth2MetadataUrl should return empty string for nil")
	require.False(t, securityScheme.GetDeprecated(), "GetDeprecated should return false for nil")
	require.NotNil(t, securityScheme.GetExtensions(), "GetExtensions should return empty extensions for nil scheme")
}

func TestOAuthFlows_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `
implicit:
  authorizationUrl: https://example.com/auth
  scopes:
    read: Read access
x-custom: value
`
	var flows openapi.OAuthFlows

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &flows)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.NotNil(t, flows.GetExtensions(), "GetExtensions should return non-nil")
}

func TestOAuthFlows_Getters_Nil(t *testing.T) {
	t.Parallel()

	var flows *openapi.OAuthFlows

	require.NotNil(t, flows.GetExtensions(), "GetExtensions should return empty extensions for nil flows")
}
