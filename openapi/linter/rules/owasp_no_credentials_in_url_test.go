package rules_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwaspNoCredentialsInURLRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "secretPath names a path to a secret, not a credential",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /files/{secretPath}:
    get:
      parameters:
        - name: secretPath
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "safe query parameter names",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: userId
          in: query
          schema:
            type: string
        - name: filter
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "credentials in header parameters are allowed",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: api-key
          in: header
          schema:
            type: string
        - name: password
          in: header
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "credentials in cookie parameters are allowed",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: token
          in: cookie
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
		},
		{
			name: "no parameters",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoCredentialsInURLRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOwaspNoCredentialsInURLRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedParam string
	}{
		{
			name: "token in query parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: token
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "token",
		},
		{
			name: "api-key in query parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: api-key
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "api-key",
		},
		{
			name: "password in path parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /auth/{password}:
    get:
      parameters:
        - name: password
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "password",
		},
		{
			name: "client_secret in query parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /oauth:
    get:
      parameters:
        - name: client_secret
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "client_secret",
		},
		{
			name: "clientSecret in path parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /oauth/{clientSecret}:
    get:
      parameters:
        - name: clientSecret
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "clientSecret",
		},
		{
			name: "access_token in query parameter",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api:
    get:
      parameters:
        - name: access_token
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "access_token",
		},
		{
			name: "multiple credential parameters",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api:
    get:
      parameters:
        - name: token
          in: query
          schema:
            type: string
        - name: api-key
          in: query
          schema:
            type: string
        - name: userId
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 2,
			expectedParam: "",
		},
		{
			name: "case insensitive match - TOKEN",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api:
    get:
      parameters:
        - name: TOKEN
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "TOKEN",
		},
		{
			name: "apikey without dash",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /api:
    get:
      parameters:
        - name: apikey
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`,
			expectedCount: 1,
			expectedParam: "apikey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspNoCredentialsInURLRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, tt.expectedCount)
			for _, err := range errs {
				assert.Contains(t, err.Error(), "credentials")
				if tt.expectedParam != "" {
					assert.Contains(t, err.Error(), tt.expectedParam)
				}
			}
		})
	}
}

func TestOwaspNoCredentialsInURLRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspNoCredentialsInURLRule{}

	assert.Equal(t, "owasp-no-credentials-in-url", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}

func TestOwaspNoCredentialsInURLRule_FlagsCredentialNames(t *testing.T) {
	t.Parallel()

	names := []string{
		"secret", "token", "password", "passwd", "pwd", "TOKEN", "token2",
		"clientSecret", "client_secret", "client-secret", "ClientSecret", "clientsecret",
		"access_token", "accessToken", "accesstoken", "refresh_token", "id_token", "idToken",
		"authToken", "auth_token", "userPassword", "user_password",
		"api-key", "api_key", "apiKey", "apikey", "APIKey", "X-API-Key", "x-api-key",
		"secretKey", "secret_key", "apiSecret", "mySecret",
		"myapikey", "salesapikey", "mysecretkey", "MyAPIKey",
		"mytoken", "mysecret", "oldpassword",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			errs := lintQueryParameterName(t, name)

			require.Len(t, errs, 1, "should flag %q as a credential", name)
			assert.Contains(t, errs[0].Error(), "`"+name+"`", "should name the offending parameter")
		})
	}
}

func TestOwaspNoCredentialsInURLRule_AllowsDescriptiveNames(t *testing.T) {
	t.Parallel()

	names := []string{
		"secretPath", "secret_path", "secret-path", "secretpath", "mySecretPath",
		"secretName", "secretId", "secretRef", "secretVersion",
		"tokenId", "tokenType", "token_type", "tokenCount", "tokenExpiry", "tokenTTL",
		"passwordHint", "passwordResetId",
		"key", "keyName", "sortKey", "apiKeyName", "apiVersion", "monkey", "turkey",
		"userId", "filter",
		"no", "on", "null", // YAML reserved words stay plain strings in the fixture
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			errs := lintQueryParameterName(t, name)

			assert.Empty(t, errs, "should not flag %q, which only refers to a credential", name)
		})
	}
}

// lintQueryParameterName runs the rule over a document whose only parameter is a query
// parameter with the given name.
func lintQueryParameterName(t *testing.T, name string) []error {
	t.Helper()
	ctx := t.Context()

	// Double-quote the name so `:`, spaces and reserved words (no, on, null) stay plain
	// strings; Go's escapes are all valid YAML double-quoted escapes.
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /things:
    get:
      parameters:
        - name: `+strconv.Quote(name)+`
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
`))
	require.NoError(t, err)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	params := idx.GetAllParameters()
	require.Len(t, params, 1)
	require.Equal(t, name, params[0].Node.GetObject().GetName(), "fixture should carry the name unchanged")
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	rule := &rules.OwaspNoCredentialsInURLRule{}
	return rule.Run(ctx, docInfo, &linter.RuleConfig{})
}
