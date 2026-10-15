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

func TestAllOfTypeConflictRule_Conflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		schemas string
		line    int
	}{
		{
			name: "object and string branches in OpenAPI 3.0", version: "3.0.3", line: 9,
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
`,
		},
		{
			name: "object and string branches in OpenAPI 3.1", version: "3.1.0", line: 9,
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
`,
		},
		{
			name: "object and string branches in OpenAPI 3.2", version: "3.2.0", line: 9,
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
`,
		},
		{
			name: "parent and immediate branch", version: "3.1.0", line: 10,
			schemas: `    Example:
      type: object
      allOf:
        - type: string
`,
		},
		{
			name: "disjoint type unions", version: "3.1.0", line: 9,
			schemas: `    Example:
      allOf:
        - type: [object, array]
        - type: [string, boolean]
`,
		},
		{
			name: "pairwise overlap but empty combined intersection", version: "3.1.0", line: 9,
			schemas: `    Example:
      allOf:
        - type: [object, string]
        - type: [object, boolean]
        - type: [string, boolean]
`,
		},
		{
			name: "nullable only on one branch does not permit null", version: "3.0.3", line: 9,
			schemas: `    Example:
      allOf:
        - type: object
          nullable: true
        - type: string
`,
		},
		{
			name: "parent nullable without type cannot override branches", version: "3.0.3", line: 10,
			schemas: `    Example:
      nullable: true
      allOf:
        - type: object
        - type: string
`,
		},
		{
			name: "nullable is not a type modifier in OpenAPI 3.1", version: "3.1.0", line: 9,
			schemas: `    Example:
      allOf:
        - type: object
          nullable: true
        - type: string
          nullable: true
`,
		},
		{
			name: "absent branch type does not hide proven conflict", version: "3.1.0", line: 9,
			schemas: `    Example:
      allOf:
        - type: object
        - description: No explicit type
        - type: string
`,
		},
		{
			name: "nested allOf is checked at its own index entry", version: "3.1.0", line: 11,
			schemas: `    Example:
      allOf:
        - description: Nested composition
          allOf:
            - type: object
            - type: string
`,
		},
		{
			name: "pure resolved local component reference", version: "3.1.0", line: 9,
			schemas: `    Example:
      allOf:
        - $ref: '#/components/schemas/Object'
        - type: string
    Object:
      type: object
`,
		},
		{
			name: "referenced composition is reported once at its target", version: "3.1.0", line: 14,
			schemas: `    Holder:
      type: object
      properties:
        example:
          $ref: '#/components/schemas/Example'
    Example:
      allOf:
        - type: object
        - type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader("openapi: "+tt.version+"\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\ncomponents:\n  schemas:\n"+tt.schemas))
			require.NoError(t, err, "document should unmarshal for semantic linting")
			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

			errs := (&rules.AllOfTypeConflictRule{}).Run(ctx, docInfo, nil)

			require.Len(t, errs, 1, "should report exactly one proven type conflict")
			var finding *validation.Error
			require.ErrorAs(t, errs[0], &finding, "finding should carry source location and severity")
			assert.Equal(t, rules.RuleSemanticAllOfTypeConflict, finding.Rule, "finding should identify the semantic rule")
			assert.Equal(t, validation.SeverityWarning, finding.Severity, "type conflicts should default to warnings")
			require.NotNil(t, finding.Node, "finding should point to the allOf keyword")
			assert.Equal(t, "allOf", finding.Node.Value, "finding should point to the conflicting composition")
			assert.Equal(t, tt.line, finding.Node.Line, "nested conflicts should point to their own composition")
			assert.Equal(t, "`allOf` has incompatible explicit type constraints on this schema and its immediate branches; no value can satisfy their intersection", finding.UnderlyingError.Error(), "message should describe unsatisfiable types, not structural invalidity")
		})
	}
}

func TestAllOfTypeConflictRule_CompatibleOrUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		schemas string
	}{
		{
			name: "number and integer overlap", version: "3.1.0",
			schemas: `    Example:
      type: number
      allOf:
        - type: integer
        - type: number
`,
		},
		{
			name: "same types overlap", version: "3.0.3",
			schemas: `    Example:
      type: object
      allOf:
        - type: object
        - type: object
`,
		},
		{
			name: "unions overlap", version: "3.1.0",
			schemas: `    Example:
      type: [object, string]
      allOf:
        - type: [string, boolean]
`,
		},
		{
			name: "number and integer in the same union are valid", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: [number, integer]
        - type: integer
`,
		},
		{
			name: "nullable branches share null in OpenAPI 3.0", version: "3.0.3",
			schemas: `    Example:
      allOf:
        - type: object
          nullable: true
        - type: string
          nullable: true
`,
		},
		{
			name: "nullable parent and branch share null", version: "3.0.3",
			schemas: `    Example:
      type: object
      nullable: true
      allOf:
        - type: string
          nullable: true
`,
		},
		{
			name: "null unions share null in OpenAPI 3.1", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: [object, 'null']
        - type: [string, 'null']
`,
		},
		{
			name: "null unions share null in OpenAPI 3.2", version: "3.2.0",
			schemas: `    Example:
      allOf:
        - type: [object, 'null']
        - type: [string, 'null']
`,
		},
		{
			name: "absent explicit types", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - description: No type
        - properties:
            name:
              type: string
`,
		},
		{
			name: "one typed branch cannot conflict alone", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - description: No type
`,
		},
		{
			name: "anyOf alternatives are not intersected", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: string
        - anyOf:
            - type: object
            - type: string
`,
		},
		{
			name: "oneOf alternatives are not intersected", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: string
        - oneOf:
            - type: object
            - type: string
`,
		},
		{
			name: "do not infer nested allOf types into the parent", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: string
        - allOf:
            - type: object
`,
		},
		{
			name: "unknown type skips the composition", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
        - type: unknown
`,
		},
		{
			name: "unknown member of union cannot be discarded", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: [string, unknown]
`,
		},
		{
			name: "empty type union", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: []
`,
		},
		{
			name: "duplicate type union members", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: [string, string]
`,
		},
		{
			name: "invalid null type in OpenAPI 3.0", version: "3.0.3",
			schemas: `    Example:
      allOf:
        - type: object
        - type: 'null'
`,
		},
		{
			name: "invalid type array in OpenAPI 3.0", version: "3.0.3",
			schemas: `    Example:
      allOf:
        - type: object
        - type: [string]
`,
		},
		{
			name: "invalid nullable value in OpenAPI 3.0", version: "3.0.3",
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
          nullable: 'true'
`,
		},
		{
			name: "boolean branch skips rather than inferring its constraints", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
        - false
`,
		},
		{
			name: "reference with siblings is not interpreted", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - $ref: '#/components/schemas/Object'
          type: string
        - type: boolean
    Object:
      type: object
`,
		},
		{
			name: "reference chain is not followed", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - $ref: '#/components/schemas/Alias'
        - type: string
    Alias:
      $ref: '#/components/schemas/Object'
    Object:
      type: object
`,
		},
		{
			name: "unresolved reference cannot supply a type", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - $ref: '#/components/schemas/Missing'
        - type: string
`,
		},
		{
			name: "resolved nullable reference remains compatible", version: "3.0.3",
			schemas: `    Example:
      allOf:
        - $ref: '#/components/schemas/Object'
        - type: string
          nullable: true
    Object:
      type: object
      nullable: true
`,
		},
		{
			name: "non-scalar type value skips the composition", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: {kind: string}
`,
		},
		{
			name: "non-string type union member skips the composition", version: "3.1.0",
			schemas: `    Example:
      allOf:
        - type: object
        - type: [string, 5]
`,
		},
		{
			name: "non-boolean nullable skips the composition in OpenAPI 3.0", version: "3.0.3",
			schemas: `    Example:
      allOf:
        - type: object
        - type: string
          nullable: [true]
`,
		},

		{
			name: "custom schema dialect is not interpreted", version: "3.1.0",
			schemas: `    Example:
      $schema: https://example.com/custom-dialect
      allOf:
        - type: object
        - type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader("openapi: "+tt.version+"\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\ncomponents:\n  schemas:\n"+tt.schemas))
			require.NoError(t, err, "document should unmarshal even when separate structural validation may find errors")
			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

			errs := (&rules.AllOfTypeConflictRule{}).Run(ctx, docInfo, nil)

			assert.Empty(t, errs, "compatible or uncertain constraints must not produce a type-conflict finding")
		})
	}
}

func TestAllOfTypeConflictRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := &rules.AllOfTypeConflictRule{}
	assert.Equal(t, "semantic-allof-type-conflict", rule.ID(), "rule should expose its stable ID")
	assert.Equal(t, rules.CategorySemantic, rule.Category(), "rule should be a semantic check")
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity(), "rule should warn by default")
	assert.Nil(t, rule.Versions(), "rule should be available for all supported OpenAPI versions")
	assert.Contains(t, rule.Description(), "not that the OpenAPI document is structurally invalid", "metadata should distinguish satisfiability from structural validity")
	assert.NotEmpty(t, rule.Summary(), "rule should have a summary")
	assert.NotEmpty(t, rule.HowToFix(), "rule should explain how to fix findings")
	assert.Contains(t, rule.Link(), "#semantic-allof-type-conflict", "rule should link to its documentation")
}

func TestAllOfTypeConflictRule_SkippedDocuments(t *testing.T) {
	t.Parallel()

	const conflict = `paths: {}
components:
  schemas:
    Example:
      allOf:
        - type: object
        - type: string
`
	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "unsupported OpenAPI version",
			yml:  "openapi: 3.9.0\ninfo:\n  title: Test\n  version: 1.0.0\n" + conflict,
		},
		{
			name: "document-level custom JSON Schema dialect",
			yml:  "openapi: 3.1.0\njsonSchemaDialect: https://example.com/custom-dialect\ninfo:\n  title: Test\n  version: 1.0.0\n" + conflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yml))
			require.NoError(t, err, "document should unmarshal")
			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

			errs := (&rules.AllOfTypeConflictRule{}).Run(ctx, docInfo, nil)

			assert.Empty(t, errs, "documents the rule cannot interpret should be skipped")
		})
	}
}

func TestAllOfTypeConflictRule_MissingDocumentOrIndex(t *testing.T) {
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
			assert.Empty(t, (&rules.AllOfTypeConflictRule{}).Run(t.Context(), tt.docInfo, nil), "missing input should be skipped safely")
		})
	}
}
