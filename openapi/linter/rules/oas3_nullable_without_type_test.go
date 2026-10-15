package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAS3NullableWithoutTypeRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		version       string
		schemas       string
		expectedCount int
	}{
		{
			name:    "nullable true without type",
			version: "3.0.3",
			schemas: `components:
  schemas:
    Nullable:
      nullable: true
`,
			expectedCount: 1,
		},
		{
			name:    "false absent and explicit type do not report",
			version: "3.0.3",
			schemas: `components:
  schemas:
    False:
      nullable: false
    Absent: {}
    Typed:
      type: string
      nullable: true
`,
			expectedCount: 0,
		},
		{
			name:    "composition type is not a local type",
			version: "3.0.3",
			schemas: `components:
  schemas:
    Composed:
      nullable: true
      allOf:
        - type: string
`,
			expectedCount: 1,
		},
		{
			name:    "nested and referenced schemas",
			version: "3.0.3",
			schemas: `components:
  schemas:
    Parent:
      type: object
      properties:
        nested:
          nullable: true
    Referenced:
      nullable: true
    UsesReference:
      $ref: '#/components/schemas/Referenced'
`,
			expectedCount: 2,
		},
		{
			name:    "boolean schemas are skipped",
			version: "3.0.3",
			schemas: `components:
  schemas:
    Open:
      type: object
      additionalProperties: true
`,
			expectedCount: 0,
		},
		{
			name:    "OpenAPI 3.1 is outside this rule version",
			version: "3.1.0",
			schemas: `components:
  schemas:
    Nullable:
      nullable: true
`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			yaml := "openapi: " + tt.version + "\ninfo:\n  title: Test\n  version: 1.0.0\n" + tt.schemas + "paths: {}\n"
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yaml))
			require.NoError(t, err)

			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")
			rule := &rules.OAS3NullableWithoutTypeRule{}
			errs := rule.Run(ctx, docInfo, nil)

			require.Len(t, errs, tt.expectedCount)
			for _, err := range errs {
				assert.Contains(t, err.Error(), "no effect")
				assert.Contains(t, err.Error(), "not a structurally invalid schema")
				var validationErr *validation.Error
				require.ErrorAs(t, err, &validationErr)
				assert.Equal(t, validation.SeverityWarning, validationErr.Severity)
				assert.Equal(t, "nullable", validationErr.Node.Value)
			}
		})
	}
}

func TestOAS3NullableWithoutTypeRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OAS3NullableWithoutTypeRule{}
	assert.Equal(t, "oas3-nullable-without-type", rule.ID())
	assert.Equal(t, rules.CategorySchemas, rule.Category())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Equal(t, []string{"3.0"}, rule.Versions())
	assert.NotEmpty(t, rule.Description(), "rule should have a description")
	assert.NotEmpty(t, rule.Summary(), "rule should have a summary")
	assert.NotEmpty(t, rule.HowToFix(), "rule should explain how to fix findings")
	assert.Contains(t, rule.Link(), "#oas3-nullable-without-type", "rule should link to its documentation")
}

func TestOAS3NullableWithoutTypeRule_MissingDocumentOrIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		docInfo *linter.DocumentInfo[*openapi.OpenAPI]
	}{
		{name: "nil document info"},
		{name: "nil document", docInfo: &linter.DocumentInfo[*openapi.OpenAPI]{}},
		{name: "nil index", docInfo: &linter.DocumentInfo[*openapi.OpenAPI]{Document: &openapi.OpenAPI{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, (&rules.OAS3NullableWithoutTypeRule{}).Run(t.Context(), tt.docInfo, nil), "missing input should be skipped safely")
		})
	}
}
