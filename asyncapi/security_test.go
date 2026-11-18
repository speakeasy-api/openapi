package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
)

func securityDoc(scheme string) string {
	return minimalDoc("components:\n  securitySchemes:\n    s:\n" + scheme)
}

func TestSecurityScheme_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "unknown type",
			doc:      securityDoc("      type: basic\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "missing type",
			doc:      securityDoc("      description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "apiKey without in",
			doc:      securityDoc("      type: apiKey\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "apiKey with an HTTP location",
			doc:      securityDoc("      type: apiKey\n      in: header\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "httpApiKey without name",
			doc:      securityDoc("      type: httpApiKey\n      in: query\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "httpApiKey with a user location",
			doc:      securityDoc("      type: httpApiKey\n      name: key\n      in: user\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "http without scheme",
			doc:      securityDoc("      type: http\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "oauth2 without flows",
			doc:      securityDoc("      type: oauth2\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "openIdConnect with a relative url",
			doc:      securityDoc("      type: openIdConnect\n      openIdConnectUrl: /.well-known/openid-configuration\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "implicit flow without authorization url",
			doc:      securityDoc("      type: oauth2\n      flows:\n        implicit:\n          availableScopes: {}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "authorization code flow without token url",
			doc:      securityDoc("      type: oauth2\n      flows:\n        authorizationCode:\n          authorizationUrl: https://auth.example.com\n          availableScopes: {}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "flow with a relative token url",
			doc:      securityDoc("      type: oauth2\n      flows:\n        clientCredentials:\n          tokenUrl: /token\n          availableScopes: {}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "flow without available scopes",
			doc:      securityDoc("      type: oauth2\n      flows:\n        password:\n          tokenUrl: https://auth.example.com/token\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
	})
}

func TestSecurityScheme_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []validationCase{}
	for _, scheme := range []string{
		"      type: userPassword\n",
		"      type: X509\n",
		"      type: symmetricEncryption\n",
		"      type: asymmetricEncryption\n",
		"      type: plain\n",
		"      type: scramSha256\n",
		"      type: scramSha512\n",
		"      type: gssapi\n",
		"      type: apiKey\n      in: password\n",
		"      type: httpApiKey\n      name: key\n      in: cookie\n",
		"      type: http\n      scheme: bearer\n",
		"      type: openIdConnect\n      openIdConnectUrl: https://auth.example.com/.well-known/openid-configuration\n      scopes: []\n",
	} {
		tests = append(tests, validationCase{name: scheme, doc: securityDoc(scheme), expected: []string{}})
	}

	runValidationCases(t, tests)
}

func TestSecurityScheme_Validate_Nil_Success(t *testing.T) {
	t.Parallel()

	runNilValidateCases(t, []nilValidateCase{
		{name: "security scheme", model: (*asyncapi.SecurityScheme)(nil)},
		{name: "oauth flows", model: (*asyncapi.OAuthFlows)(nil)},
		{name: "oauth flow", model: (*asyncapi.OAuthFlow)(nil)},
	})
}

func TestOAuthFlow_Validate_BuiltInCode_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    validatable
		expected []string
	}{
		{
			name:     "standalone flow requires available scopes",
			model:    &asyncapi.OAuthFlow{},
			expected: []string{"[-1:-1] error validation-required-field `oauthFlow.availableScopes` is required"},
		},
		{
			name:  "implicit flow requires authorization url and available scopes",
			model: &asyncapi.OAuthFlows{Implicit: &asyncapi.OAuthFlow{}},
			expected: []string{
				"[-1:-1] error validation-required-field `oauthFlows.implicit.authorizationUrl` is required",
				"[-1:-1] error validation-required-field `oauthFlows.implicit.availableScopes` is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, errorStrings(tt.model.Validate(t.Context())), "validation errors should match")
		})
	}
}

func TestOAuthFlow_Validate_BuiltInCode_Success(t *testing.T) {
	t.Parallel()

	flow := &asyncapi.OAuthFlow{
		TokenURL:        pointer.From("https://auth.example.com/token"),
		AvailableScopes: sequencedmap.New(sequencedmap.NewElem("orders:write", "Place orders")),
	}
	assert.Empty(t, flow.Validate(t.Context()), "complete flow should have no validation errors")
	assert.True(t, flow.Valid, "complete flow should be valid")
}

func TestSecurityScheme_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.SecurityScheme{
		Type:             asyncapi.SecuritySchemeTypeOAuth2,
		Description:      pointer.From("OAuth access."),
		Name:             pointer.From("api_key"),
		In:               pointer.From("header"),
		Scheme:           pointer.From("bearer"),
		BearerFormat:     pointer.From("JWT"),
		Flows:            &asyncapi.OAuthFlows{},
		OpenIDConnectURL: pointer.From("https://auth.example.com/.well-known/openid-configuration"),
		Scopes:           []string{"orders:write"},
		Extensions:       exts,
	}

	type getters struct {
		schemeType       asyncapi.SecuritySchemeType
		description      string
		name             string
		in               string
		scheme           string
		bearerFormat     string
		flows            *asyncapi.OAuthFlows
		openIDConnectURL string
		scopes           []string
		extensions       *extensions.Extensions
	}

	tests := []struct {
		name     string
		scheme   *asyncapi.SecurityScheme
		expected getters
	}{
		{
			name:     "nil scheme returns defaults",
			scheme:   nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			scheme:   &asyncapi.SecurityScheme{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:   "populated fields are returned",
			scheme: populated,
			expected: getters{
				schemeType:       asyncapi.SecuritySchemeTypeOAuth2,
				description:      "OAuth access.",
				name:             "api_key",
				in:               "header",
				scheme:           "bearer",
				bearerFormat:     "JWT",
				flows:            populated.Flows,
				openIDConnectURL: "https://auth.example.com/.well-known/openid-configuration",
				scopes:           []string{"orders:write"},
				extensions:       exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				schemeType:       tt.scheme.GetType(),
				description:      tt.scheme.GetDescription(),
				name:             tt.scheme.GetName(),
				in:               tt.scheme.GetIn(),
				scheme:           tt.scheme.GetScheme(),
				bearerFormat:     tt.scheme.GetBearerFormat(),
				flows:            tt.scheme.GetFlows(),
				openIDConnectURL: tt.scheme.GetOpenIDConnectURL(),
				scopes:           tt.scheme.GetScopes(),
				extensions:       tt.scheme.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestOAuthFlows_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.OAuthFlows{
		Implicit:          &asyncapi.OAuthFlow{AuthorizationURL: pointer.From("https://auth.example.com/implicit")},
		Password:          &asyncapi.OAuthFlow{TokenURL: pointer.From("https://auth.example.com/password")},
		ClientCredentials: &asyncapi.OAuthFlow{TokenURL: pointer.From("https://auth.example.com/client")},
		AuthorizationCode: &asyncapi.OAuthFlow{TokenURL: pointer.From("https://auth.example.com/code")},
		Extensions:        exts,
	}

	type getters struct {
		implicit          *asyncapi.OAuthFlow
		password          *asyncapi.OAuthFlow
		clientCredentials *asyncapi.OAuthFlow
		authorizationCode *asyncapi.OAuthFlow
		extensions        *extensions.Extensions
	}

	tests := []struct {
		name     string
		flows    *asyncapi.OAuthFlows
		expected getters
	}{
		{
			name:     "nil flows return defaults",
			flows:    nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			flows:    &asyncapi.OAuthFlows{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:  "populated fields are returned",
			flows: populated,
			expected: getters{
				implicit:          populated.Implicit,
				password:          populated.Password,
				clientCredentials: populated.ClientCredentials,
				authorizationCode: populated.AuthorizationCode,
				extensions:        exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				implicit:          tt.flows.GetImplicit(),
				password:          tt.flows.GetPassword(),
				clientCredentials: tt.flows.GetClientCredentials(),
				authorizationCode: tt.flows.GetAuthorizationCode(),
				extensions:        tt.flows.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestOAuthFlow_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	scopes := sequencedmap.New(sequencedmap.NewElem("orders:write", "Place orders"))

	type getters struct {
		authorizationURL string
		tokenURL         string
		refreshURL       string
		availableScopes  *sequencedmap.Map[string, string]
		extensions       *extensions.Extensions
	}

	tests := []struct {
		name     string
		flow     *asyncapi.OAuthFlow
		expected getters
	}{
		{
			name:     "nil flow returns defaults",
			flow:     nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			flow:     &asyncapi.OAuthFlow{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			flow: &asyncapi.OAuthFlow{
				AuthorizationURL: pointer.From("https://auth.example.com/authorize"),
				TokenURL:         pointer.From("https://auth.example.com/token"),
				RefreshURL:       pointer.From("https://auth.example.com/refresh"),
				AvailableScopes:  scopes,
				Extensions:       exts,
			},
			expected: getters{
				authorizationURL: "https://auth.example.com/authorize",
				tokenURL:         "https://auth.example.com/token",
				refreshURL:       "https://auth.example.com/refresh",
				availableScopes:  scopes,
				extensions:       exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				authorizationURL: tt.flow.GetAuthorizationURL(),
				tokenURL:         tt.flow.GetTokenURL(),
				refreshURL:       tt.flow.GetRefreshURL(),
				availableScopes:  tt.flow.GetAvailableScopes(),
				extensions:       tt.flow.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}
