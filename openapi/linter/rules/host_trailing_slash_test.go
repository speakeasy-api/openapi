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

func TestOAS3HostTrailingSlashRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "no trailing slash",
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
			name: "url with path no trailing slash",
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
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3HostTrailingSlashRule{}
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

func TestOAS3HostTrailingSlashRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "trailing slash on domain",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com/
paths: {}
`,
			expectedError: "[7:10] warning style-oas3-host-trailing-slash server url \"https://api.example.com/\" should not have a trailing slash",
		},
		{
			name: "trailing slash on path",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com/v1/
paths: {}
`,
			expectedError: "[7:10] warning style-oas3-host-trailing-slash server url \"https://api.example.com/v1/\" should not have a trailing slash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OAS3HostTrailingSlashRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			require.Len(t, errs, 1, "should have one lint error")
			assert.Equal(t, tt.expectedError, errs[0].Error(), "error message should match exactly")
		})
	}
}

func TestOAS3HostTrailingSlashRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OAS3HostTrailingSlashRule{}

	assert.Equal(t, "style-oas3-host-trailing-slash", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.NotNil(t, rule.Versions())
}
