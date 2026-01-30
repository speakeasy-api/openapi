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

func TestTypedEnumRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "string enum with string values",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
`,
		},
		{
			name: "integer enum with integer values",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Priority:
      type: integer
      enum:
        - 1
        - 2
        - 3
`,
		},
		{
			name: "number enum with numeric values",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Rating:
      type: number
      enum:
        - 1.5
        - 2.0
        - 4.5
`,
		},
		{
			name: "boolean enum",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Flag:
      type: boolean
      enum:
        - true
        - false
`,
		},
		{
			name: "number type with integer value is valid",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Count:
      type: number
      enum:
        - 1
        - 2
`,
		},
		{
			name: "null type with null value",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Nullable:
      type: "null"
      enum:
        - null
`,
		},
		{
			name: "enum without type specified",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Mixed:
      enum:
        - value1
        - 123
`,
		},
		{
			name: "schema without enum",
			yaml: `
openapi: 3.1.0
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
`,
		},
		{
			name: "nullable integer enum with null value",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableIntEnum:
      type: integer
      nullable: true
      enum:
        - 1
        - 2
        - 3
        - null
`,
		},
		{
			name: "nullable string enum with null value",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NullableStringEnum:
      type: string
      nullable: true
      enum:
        - First
        - Second
        - Third
        - null
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.TypedEnumRule{}
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

func TestTypedEnumRule_TypeMismatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "string type with integer value",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - 123
`,
			expectedError: "[12:11] warning semantic-typed-enum enum value at index 1 does not match schema type [string]",
		},
		{
			name: "integer type with string value",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Priority:
      type: integer
      enum:
        - 1
        - high
`,
			expectedError: "[12:11] warning semantic-typed-enum enum value at index 1 does not match schema type [integer]",
		},
		{
			name: "boolean type with string value",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Flag:
      type: boolean
      enum:
        - true
        - yes
`,
			expectedError: "[12:11] warning semantic-typed-enum enum value at index 1 does not match schema type [boolean]",
		},
		{
			name: "openapi 3.0 null in enum without nullable true",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - null
`,
			expectedError: "[13:11] warning semantic-typed-enum enum contains null at index 2 but schema does not have 'nullable: true'. Add 'nullable: true' to allow null values",
		},
		{
			name: "openapi 3.1 null in enum without null in type",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Priority:
      type: integer
      enum:
        - 1
        - 2
        - null
`,
			expectedError: `[13:11] warning semantic-typed-enum enum contains null at index 2 but schema type does not include null. Change 'type: [integer]' to 'type: ["integer", "null"]' to allow null values`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.TypedEnumRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs, "should have lint errors")
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestTypedEnumRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.TypedEnumRule{}

	assert.Equal(t, "semantic-typed-enum", rule.ID())
	assert.Equal(t, rules.CategorySemantic, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
