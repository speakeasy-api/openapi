package rules

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOASSchemaCheck_StringConstraints_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "valid minLength and maxLength",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minLength: 5
      maxLength: 10
paths: {}
`,
		},
		{
			name: "valid pattern",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      pattern: ^[a-z]+$
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, tt.yaml)
			assert.Empty(t, errs)
		})
	}
}

func TestOASSchemaCheck_StringConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "negative minLength",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minLength: -1
paths: {}
`,
			expected: 1,
		},
		{
			name: "maxLength less than minLength",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minLength: 10
      maxLength: 5
paths: {}
`,
			expected: 1,
		},
		{
			name: "invalid regex pattern",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      pattern: "[invalid("
paths: {}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, tt.yaml)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func TestOASSchemaCheck_NumberConstraints_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "valid minimum and maximum",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: number
      minimum: 0
      maximum: 100
paths: {}
`,
		},
		{
			name: "valid multipleOf",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: integer
      multipleOf: 5
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, tt.yaml)
			assert.Empty(t, errs)
		})
	}
}

func TestOASSchemaCheck_NumberConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "multipleOf zero",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: integer
      multipleOf: 0
paths: {}
`,
			expected: 1,
		},
		{
			name: "maximum less than minimum",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: number
      minimum: 100
      maximum: 0
paths: {}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, tt.yaml)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func TestOASSchemaCheck_TypeMismatch_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "string type with number constraints",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minimum: 0
      maximum: 100
paths: {}
`,
			expected: 2, // minimum and maximum
		},
		{
			name: "number type with string constraints",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: number
      minLength: 5
      pattern: ^[a-z]+$
paths: {}
`,
			expected: 2, // minLength and pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, tt.yaml)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func TestOASSchemaCheck_ObjectRequired_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "required without properties",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      required:
        - name
paths: {}
`,
			expected: 1,
		},
		{
			name: "required field not in properties",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      properties:
        age:
          type: integer
      required:
        - name
paths: {}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, tt.yaml)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func runOASSchemaCheckTest(t *testing.T, yaml string) []error {
	t.Helper()
	ctx := t.Context()

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yaml))
	require.NoError(t, err)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

	return (&OASSchemaCheckRule{}).Run(ctx, docInfo, &linter.RuleConfig{})
}

func TestOASSchemaCheck_BooleanAndNullConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		schema          string
		expectedMessage string
	}{
		{
			name: "boolean rejects string constraint",
			schema: `type: boolean
      minLength: 2`,
			expectedMessage: "`minLength` constraint is only applicable to string types, not `boolean`",
		},
		{
			name: "null rejects array constraint",
			schema: `type: "null"
      uniqueItems: true`,
			expectedMessage: "`uniqueItems` constraint is only applicable to array types, not `null`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      `+tt.schema+`
paths: {}
`)
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], tt.expectedMessage)
		})
	}
}

func TestOASSchemaCheck_NullUnionConstraints_Success(t *testing.T) {
	t.Parallel()

	errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: [array, "null"]
      minItems: 1
      uniqueItems: true
paths: {}
`)
	assert.Empty(t, errs)
}

func TestOASSchemaCheck_ConstType_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		typeName        string
		constValue      string
		expectedMessage string
	}{
		{
			name:            "string rejects integer constant",
			typeName:        "string",
			constValue:      "42",
			expectedMessage: "`const` value type does not match schema type [`string`]",
		},
		{
			name:            "integer rejects fractional constant",
			typeName:        "integer",
			constValue:      "42.5",
			expectedMessage: "`const` value type does not match schema type [`integer`]",
		},
		{
			name:            "boolean rejects string constant",
			typeName:        "boolean",
			constValue:      `"true"`,
			expectedMessage: "`const` value type does not match schema type [`boolean`]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: `+tt.typeName+`
      const: `+tt.constValue+`
paths: {}
`)
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], tt.expectedMessage)
		})
	}
}

func TestOASSchemaCheck_ConstTypes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{name: "integer accepts whole float", schema: `type: integer
      const: 42.0`},
		{name: "number accepts integer", schema: `type: number
      const: 42`},
		{name: "null accepts null", schema: `type: "null"
      const: null`},
		{name: "array accepts sequence", schema: `type: array
      const: [one, two]`},
		{name: "object accepts mapping", schema: `type: object
      const: {key: value}`},
		{name: "union accepts a matching type", schema: `type: [string, integer]
      const: 42`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      `+tt.schema+`
paths: {}
`)
			assert.Empty(t, errs)
		})
	}
}

func TestOASSchemaCheck_EnumConst_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		enum            string
		constValue      string
		expectedMessage string
	}{
		{
			name:            "const excluded from enum",
			enum:            "[one, two]",
			constValue:      "three",
			expectedMessage: "is not present in `enum` values",
		},
		{
			name:            "single enum value duplicates const",
			enum:            "[one]",
			constValue:      "one",
			expectedMessage: "schema uses both `enum` with single value and `const` - consider using only `const`",
		},
		{
			name:            "multi-value enum conflicts with const",
			enum:            "[one, two]",
			constValue:      "one",
			expectedMessage: "schema uses both `enum` and `const` - this is likely an oversight as `const` restricts to a single value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      enum: `+tt.enum+`
      const: `+tt.constValue+`
paths: {}
`)
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], tt.expectedMessage)
		})
	}
}

func TestOASSchemaCheck_Discriminator_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		discriminator   string
		expectedMessage string
	}{
		{
			name:            "property does not exist",
			discriminator:   "propertyName: kind",
			expectedMessage: "discriminator property `kind` is not defined in schema properties",
		},
		{
			name:            "property name missing",
			discriminator:   "mapping: {}",
			expectedMessage: "discriminator object is missing required `propertyName` field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      discriminator:
        `+tt.discriminator+`
paths: {}
`)
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], tt.expectedMessage)
		})
	}
}

func TestOASSchemaCheck_DiscriminatorAndRequired_PolymorphicProperty_Success(t *testing.T) {
	t.Parallel()

	errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      required: [kind]
      discriminator:
        propertyName: kind
      allOf:
        - type: object
          properties:
            kind:
              type: string
paths: {}
`)
	assert.Empty(t, errs)
}

func TestOASSchemaCheck_ArrayAndObjectConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		schema          string
		expectedMessage string
	}{
		{
			name: "array maximum less than minimum",
			schema: `type: array
      minItems: 3
      maxItems: 1`,
			expectedMessage: "`maxItems` should be greater than or equal to `minItems`",
		},
		{
			name: "array maxContains less than minContains",
			schema: `type: array
      minContains: 4
      maxContains: 2`,
			expectedMessage: "`maxContains` should be greater than or equal to `minContains`",
		},
		{
			name: "object maximum less than minimum",
			schema: `type: object
      minProperties: 3
      maxProperties: 1`,
			expectedMessage: "`maxProperties` should be greater than or equal to `minProperties`",
		},
		{
			name: "object rejects array constraint",
			schema: `type: object
      uniqueItems: true`,
			expectedMessage: "`uniqueItems` constraint is only applicable to array types, not `object`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      `+tt.schema+`
paths: {}
`)
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], tt.expectedMessage)
		})
	}
}

func TestOASSchemaCheck_TypeInappropriateConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		schema          string
		expectedMessage string
	}{
		{
			name: "array rejects numeric constraint",
			schema: `type: array
      minimum: 1`,
			expectedMessage: "`minimum` constraint is only applicable to number/integer types, not `array`",
		},
		{
			name: "string rejects array constraint",
			schema: `type: string
      minItems: 1`,
			expectedMessage: "`minItems` constraint is only applicable to array types, not `string`",
		},
		{
			name: "integer rejects object constraint",
			schema: `type: integer
      minProperties: 1`,
			expectedMessage: "`minProperties` constraint is only applicable to object types, not `number`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := runOASSchemaCheckTest(t, `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      `+tt.schema+`
paths: {}
`)
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], tt.expectedMessage)
		})
	}
}
