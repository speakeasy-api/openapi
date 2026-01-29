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

func TestOAS3APIServersRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "servers with valid URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
`,
		},
		{
			name: "multiple servers",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
  - url: https://staging.example.com
`,
		},
		{
			name: "server with template variables",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://{environment}.example.com
    variables:
      environment:
        default: api
`,
		},
		{
			name: "server with path only",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: /api/v1
`,
		},
		{
			name: "server with host and path",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3APIServersRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs, "should have no lint errors")
		})
	}
}

func TestOAS3APIServersRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "no servers defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: OK
`,
			expectedError: "no servers defined for the specification",
		},
		{
			name: "servers array is empty",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers: []
`,
			expectedError: "no servers defined for the specification",
		},
		{
			name: "server missing URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - description: Missing URL
`,
			expectedError: "server definition is missing a URL",
		},
		{
			name: "server with invalid URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: "://invalid-url"
`,
			expectedError: "server URL \"://invalid-url\" cannot be parsed",
		},
		{
			name: "server with empty URL",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: ""
`,
			expectedError: "server definition is missing a URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3APIServersRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			require.NotEmpty(t, errs, "should have lint errors")
			assert.Contains(t, errs[0].Error(), tt.expectedError)
		})
	}
}

func TestOAS3APIServersRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		shouldError   bool
		expectedError string
	}{
		{
			name: "server with scheme-only URL (no host or path)",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: "http://"
`,
			shouldError:   true,
			expectedError: "is not valid: no hostname or path provided",
		},
		{
			name: "server with template variables in path",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com/{version}
    variables:
      version:
        default: v1
`,
			shouldError: false,
		},
		{
			name: "server with multiple template variables",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://{environment}.example.com/{version}
    variables:
      environment:
        default: api
      version:
        default: v1
`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3APIServersRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			if tt.shouldError {
				require.NotEmpty(t, errs, "should have lint errors")
				assert.Contains(t, errs[0].Error(), tt.expectedError)
			} else {
				assert.Empty(t, errs, "should have no lint errors")
			}
		})
	}
}

func TestOAS3APIServersRule_NilInputs(t *testing.T) {
	t.Parallel()

	rule := &rules.OAS3APIServersRule{}
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

func TestOAS3APIServersRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OAS3APIServersRule{}

	assert.Equal(t, "style-oas3-api-servers", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1.0", "3.1.1", "3.2.0"}, rule.Versions())
}
