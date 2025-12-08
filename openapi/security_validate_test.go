package openapi_test

import (
	"bytes"
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
			wantErrs: []string{"[2:1] securityScheme.type is missing"},
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
undefined_scheme: []
`,
			expectedErr: "securityRequirement scheme undefined_scheme is not defined in components.securitySchemes",
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
