package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestSecurityScheme_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "api_key_header",
			yml: `
type: apiKey
name: X-API-Key
in: header
description: API key authentication
`,
		},
		{
			name: "api_key_query",
			yml: `
type: apiKey
name: api_key
in: query
`,
		},
		{
			name: "api_key_cookie",
			yml: `
type: apiKey
name: sessionId
in: cookie
`,
		},
		{
			name: "http_basic",
			yml: `
type: http
scheme: basic
description: Basic authentication
`,
		},
		{
			name: "http_bearer",
			yml: `
type: http
scheme: bearer
bearerFormat: JWT
`,
		},
		{
			name: "mutual_tls",
			yml: `
type: mutualTLS
description: Mutual TLS authentication
`,
		},
		{
			name: "oauth2",
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
			name: "openid_connect",
			yml: `
type: openIdConnect
openIdConnectUrl: https://example.com/.well-known/openid_configuration
`,
		},
		{
			name: "with_extensions",
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

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &securityScheme)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Basic assertions to ensure unmarshaling worked
			require.NotEmpty(t, securityScheme.GetType())
		})
	}
}

func TestSecurityRequirement_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "api_key_requirement",
			yml: `
api_key: []
`,
		},
		{
			name: "oauth2_requirement",
			yml: `
oauth2:
  - read
  - write
`,
		},
		{
			name: "multiple_requirements",
			yml: `
api_key: []
oauth2:
  - read
`,
		},
		{
			name: "empty_requirement",
			yml:  `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityRequirement openapi.SecurityRequirement

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &securityRequirement)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Basic assertion to ensure unmarshaling worked
			require.NotNil(t, securityRequirement.Map)
		})
	}
}

func TestOAuthFlows_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
implicit:
  authorizationUrl: https://example.com/oauth/authorize
  scopes:
    read: Read access
    write: Write access
password:
  tokenUrl: https://example.com/oauth/token
  scopes:
    read: Read access
clientCredentials:
  tokenUrl: https://example.com/oauth/token
  scopes:
    admin: Admin access
authorizationCode:
  authorizationUrl: https://example.com/oauth/authorize
  tokenUrl: https://example.com/oauth/token
  refreshUrl: https://example.com/oauth/refresh
  scopes:
    read: Read access
    write: Write access
x-custom: value
`

	var oauthFlows openapi.OAuthFlows

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &oauthFlows)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.NotNil(t, oauthFlows.GetImplicit())
	require.Equal(t, "https://example.com/oauth/authorize", oauthFlows.GetImplicit().GetAuthorizationURL())

	require.NotNil(t, oauthFlows.GetPassword())
	require.Equal(t, "https://example.com/oauth/token", oauthFlows.GetPassword().GetTokenURL())

	require.NotNil(t, oauthFlows.GetClientCredentials())
	require.Equal(t, "https://example.com/oauth/token", oauthFlows.GetClientCredentials().GetTokenURL())

	require.NotNil(t, oauthFlows.GetAuthorizationCode())
	require.Equal(t, "https://example.com/oauth/authorize", oauthFlows.GetAuthorizationCode().GetAuthorizationURL())
	require.Equal(t, "https://example.com/oauth/token", oauthFlows.GetAuthorizationCode().GetTokenURL())
	require.Equal(t, "https://example.com/oauth/refresh", oauthFlows.GetAuthorizationCode().GetRefreshURL())

	ext, ok := oauthFlows.GetExtensions().Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "value", ext.Value)
}

func TestOAuthFlow_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "implicit_flow",
			yml: `
authorizationUrl: https://example.com/oauth/authorize
scopes:
  read: Read access
  write: Write access
`,
		},
		{
			name: "password_flow",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
`,
		},
		{
			name: "client_credentials_flow",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  admin: Admin access
`,
		},
		{
			name: "authorization_code_flow",
			yml: `
authorizationUrl: https://example.com/oauth/authorize
tokenUrl: https://example.com/oauth/token
refreshUrl: https://example.com/oauth/refresh
scopes:
  read: Read access
  write: Write access
`,
		},
		{
			name: "empty_scopes",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes: {}
`,
		},
		{
			name: "with_extensions",
			yml: `
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
x-custom: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var oauthFlow openapi.OAuthFlow

			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &oauthFlow)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Basic assertion to ensure unmarshaling worked
			require.NotNil(t, oauthFlow.GetScopes())
		})
	}
}
