package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAS3NoNullableRule_VersionFiltering verifies that the linter engine
// properly filters rules based on their Versions() method
func TestOAS3NoNullableRule_VersionFiltering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		expectErrors bool
		description  string
	}{
		{
			name: "OpenAPI 3.1.0 - rule should run and detect violation",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableName:
      type: string
      nullable: true
paths: {}
`,
			expectErrors: true,
			description:  "OpenAPI 3.1.0 should trigger the rule",
		},
		{
			name: "OpenAPI 3.0.0 - rule should not run",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableName:
      type: string
      nullable: true
paths: {}
`,
			expectErrors: false,
			description:  "OpenAPI 3.0.0 should not trigger the rule (version filtering)",
		},
		{
			name: "OpenAPI 3.0.3 - rule should not run",
			yaml: `
openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableName:
      type: string
      nullable: true
paths: {}
`,
			expectErrors: false,
			description:  "OpenAPI 3.0.3 should not trigger the rule (version filtering)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			// Build index
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			// Create linter with default config
			config := &linter.Config{
				Extends: []string{"all"},
			}
			l := openapiLinter.NewLinter(config)

			// Lint the document
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)
			output, err := l.Lint(ctx, docInfo, nil, nil)
			require.NoError(t, err, "lint should succeed")

			// Filter results to only oas3-no-nullable rule
			var ruleResults []error
			for _, result := range output.Results {
				// Check if this is a validation error from our rule
				if strings.Contains(result.Error(), "nullable") &&
					strings.Contains(result.Error(), "3.1") {
					ruleResults = append(ruleResults, result)
				}
			}

			if tt.expectErrors {
				assert.NotEmpty(t, ruleResults, tt.description)
			} else {
				assert.Empty(t, ruleResults, tt.description)
			}
		})
	}
}
