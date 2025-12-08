package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/require"
)

func TestSecurityScheme_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_basic_auth",
			yml: `type: basic
description: Basic authentication`,
		},
		{
			name: "valid_apiKey_header",
			yml: `type: apiKey
name: X-API-Key
in: header
description: API key authentication`,
		},
		{
			name: "valid_apiKey_query",
			yml: `type: apiKey
name: api_key
in: query`,
		},
		{
			name: "valid_oauth2_implicit",
			yml: `type: oauth2
flow: implicit
authorizationUrl: https://example.com/oauth/authorize
scopes:
  read: Read access
  write: Write access`,
		},
		{
			name: "valid_oauth2_password",
			yml: `type: oauth2
flow: password
tokenUrl: https://example.com/oauth/token
scopes:
  admin: Admin access`,
		},
		{
			name: "valid_oauth2_application",
			yml: `type: oauth2
flow: application
tokenUrl: https://example.com/oauth/token
scopes:
  api: API access`,
		},
		{
			name: "valid_oauth2_accessCode",
			yml: `type: oauth2
flow: accessCode
authorizationUrl: https://example.com/oauth/authorize
tokenUrl: https://example.com/oauth/token
scopes:
  read: Read access
  write: Write access`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityScheme swagger.SecurityScheme

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
			name:     "missing_type",
			yml:      `description: Some security scheme`,
			wantErrs: []string{"securityScheme.type is missing"},
		},
		{
			name: "invalid_type",
			yml: `type: invalid
description: Test`,
			wantErrs: []string{"securityScheme.type must be one of"},
		},
		{
			name: "apiKey_missing_name",
			yml: `type: apiKey
in: header`,
			wantErrs: []string{"securityScheme.name is required for type=apiKey"},
		},
		{
			name: "apiKey_missing_in",
			yml: `type: apiKey
name: X-API-Key`,
			wantErrs: []string{"securityScheme.in is required for type=apiKey"},
		},
		{
			name: "apiKey_invalid_in",
			yml: `type: apiKey
name: X-API-Key
in: invalid`,
			wantErrs: []string{"securityScheme.in must be one of"},
		},
		{
			name: "oauth2_missing_flow",
			yml: `type: oauth2
scopes:
  read: Read access`,
			wantErrs: []string{"securityScheme.flow is required for type=oauth2"},
		},
		{
			name: "oauth2_invalid_flow",
			yml: `type: oauth2
flow: invalid
scopes:
  read: Read access`,
			wantErrs: []string{"securityScheme.flow must be one of"},
		},
		{
			name: "oauth2_implicit_missing_authorizationUrl",
			yml: `type: oauth2
flow: implicit
scopes:
  read: Read access`,
			wantErrs: []string{"securityScheme.authorizationUrl is required for flow=implicit"},
		},
		{
			name: "oauth2_password_missing_tokenUrl",
			yml: `type: oauth2
flow: password
scopes:
  read: Read access`,
			wantErrs: []string{"securityScheme.tokenUrl is required for flow=password"},
		},
		{
			name: "oauth2_accessCode_missing_authorizationUrl",
			yml: `type: oauth2
flow: accessCode
tokenUrl: https://example.com/token
scopes:
  read: Read access`,
			wantErrs: []string{"securityScheme.authorizationUrl is required for flow=accessCode"},
		},
		{
			name: "oauth2_accessCode_missing_tokenUrl",
			yml: `type: oauth2
flow: accessCode
authorizationUrl: https://example.com/authorize
scopes:
  read: Read access`,
			wantErrs: []string{"securityScheme.tokenUrl is required for flow=accessCode"},
		},
		{
			name: "oauth2_missing_scopes",
			yml: `type: oauth2
flow: password
tokenUrl: https://example.com/token`,
			wantErrs: []string{"securityScheme.scopes is required for type=oauth2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var securityScheme swagger.SecurityScheme

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &securityScheme)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := securityScheme.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

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
