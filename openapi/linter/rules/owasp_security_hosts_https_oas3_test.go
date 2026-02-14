package rules_test

import (
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

func TestOwaspSecurityHostsHttpsOAS3Rule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "https server url",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com
paths: {}
`,
		},
		{
			name: "https with path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths: {}
`,
		},
		{
			name: "multiple https servers",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com
    description: Production
  - url: https://staging.example.com
    description: Staging
paths: {}
`,
		},
		{
			name: "no servers defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspSecurityHostsHttpsOAS3Rule{}
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

func TestOwaspSecurityHostsHttpsOAS3Rule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectedText  string
	}{
		{
			name: "http server url",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: http://api.example.com
paths: {}
`,
			expectedCount: 1,
			expectedText:  "http://api.example.com",
		},
		{
			name: "ftp server url",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: ftp://api.example.com
paths: {}
`,
			expectedCount: 1,
			expectedText:  "ftp://api.example.com",
		},
		{
			name: "mixed https and http servers",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com
    description: Production
  - url: http://staging.example.com
    description: Staging (insecure)
paths: {}
`,
			expectedCount: 1,
			expectedText:  "http://staging.example.com",
		},
		{
			name: "multiple non-https servers",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: http://api.example.com
  - url: ws://websocket.example.com
paths: {}
`,
			expectedCount: 2,
			expectedText:  "",
		},
		{
			name: "relative url not starting with https",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: /api/v1
paths: {}
`,
			expectedCount: 1,
			expectedText:  "/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspSecurityHostsHttpsOAS3Rule{}
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
				assert.Contains(t, err.Error(), "HTTPS")
				if tt.expectedText != "" {
					assert.Contains(t, err.Error(), tt.expectedText)
				}
			}
		})
	}
}

func TestOwaspSecurityHostsHttpsOAS3Rule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "server with empty url",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: ""
    description: Empty URL
paths: {}
`,
		},
		{
			name: "server with variables in https url",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://{environment}.example.com
    variables:
      environment:
        default: api
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OwaspSecurityHostsHttpsOAS3Rule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			// Should not panic
			errs := rule.Run(ctx, docInfo, config)

			// Empty URL should be skipped, variables in https URL should be valid
			assert.Empty(t, errs)
		})
	}
}

func TestOwaspSecurityHostsHttpsOAS3Rule_NilInputs(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspSecurityHostsHttpsOAS3Rule{}
	config := &linter.RuleConfig{}
	ctx := t.Context()

	// Test with nil docInfo
	errs := rule.Run(ctx, nil, config)
	assert.Empty(t, errs)

	// Test with nil document
	var nilDoc *openapi.OpenAPI
	errs = rule.Run(ctx, linter.NewDocumentInfoWithIndex(nilDoc, "test.yaml", nil), config)
	assert.Empty(t, errs)
}

func TestOwaspSecurityHostsHttpsOAS3Rule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OwaspSecurityHostsHttpsOAS3Rule{}

	assert.Equal(t, "owasp-security-hosts-https-oas3", rule.ID())
	assert.Equal(t, rules.CategorySecurity, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
