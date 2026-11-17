package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/validation"
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
