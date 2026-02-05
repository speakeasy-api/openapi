package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoRefSiblingsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "reference only schema",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      $ref: '#/components/schemas/Person'
    Person:
      type: object
      properties:
        name:
          type: string
`,
		},
		{
			name: "schema without reference",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
`,
		},
		{
			name: "nested reference-only schemas",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        address:
          $ref: '#/components/schemas/Address'
    Address:
      type: object
      properties:
        street:
          type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.NoRefSiblingsRule{}
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

func TestNoRefSiblingsRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "ref with description",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      $ref: '#/components/schemas/Person'
      description: A user object
    Person:
      type: object
      properties:
        name:
          type: string
`,
			expectedError: "[9:7] warning style-no-ref-siblings schema contains $ref with sibling properties, which is not allowed in OAS 3.0.x",
		},
		{
			name: "ref with type",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      $ref: '#/components/schemas/Person'
      type: object
    Person:
      type: object
`,
			expectedError: "[9:7] warning style-no-ref-siblings schema contains $ref with sibling properties, which is not allowed in OAS 3.0.x",
		},
		{
			name: "ref with example",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      $ref: '#/components/schemas/Person'
      example:
        name: John
    Person:
      type: object
`,
			expectedError: "[9:7] warning style-no-ref-siblings schema contains $ref with sibling properties, which is not allowed in OAS 3.0.x",
		},
		{
			name: "ref with title",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      $ref: '#/components/schemas/Person'
      title: User Schema
    Person:
      type: object
`,
			expectedError: "[9:7] warning style-no-ref-siblings schema contains $ref with sibling properties, which is not allowed in OAS 3.0.x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.NoRefSiblingsRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestNoRefSiblingsRule_OAS31Allowed(t *testing.T) {
	t.Parallel()

	yaml := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      $ref: '#/components/schemas/Person'
      description: A user object
    Person:
      type: object
      properties:
        name:
          type: string
`

	ctx := t.Context()

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yaml))
	require.NoError(t, err)

	// Create linter with the rule registered
	lntr, err := openapiLinter.NewLinter(&linter.Config{})
	require.NoError(t, err)

	// Build index for the rule
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	// Lint through the framework which will apply version filtering
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Filter to only no-ref-siblings errors
	var noRefSiblingsErrors []error
	for _, result := range output.Results {
		// Check if this is an error from the no-ref-siblings rule
		if strings.Contains(result.Error(), "style-no-ref-siblings") {
			noRefSiblingsErrors = append(noRefSiblingsErrors, result)
		}
	}

	// Should be empty because rule only applies to OAS 3.0.x
	assert.Empty(t, noRefSiblingsErrors, "OAS 3.1 should allow $ref siblings")
}

func TestNoRefSiblingsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.NoRefSiblingsRule{}

	assert.Equal(t, "style-no-ref-siblings", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0.0", "3.0.1", "3.0.2", "3.0.3"}, rule.Versions())
}
