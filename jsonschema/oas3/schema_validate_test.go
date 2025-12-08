package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid simple string schema",
			yml: `
type: string
title: Simple String
description: A simple string schema
`,
		},
		{
			name: "valid string schema with all string properties",
			yml: `
type: string
title: Complete String Schema
description: A comprehensive string schema
pattern: "^[a-zA-Z]+$"
format: email
minLength: 5
maxLength: 100
`,
		},
		{
			name: "valid number schema with all numeric properties",
			yml: `
type: number
title: Number Schema
description: A comprehensive number schema
minimum: 0.0
maximum: 100.0
exclusiveMinimum: 0.1
exclusiveMaximum: 99.9
multipleOf: 0.5
`,
		},
		{
			name: "valid integer schema with all integer properties",
			yml: `
type: integer
title: Integer Schema
description: A comprehensive integer schema
minimum: 1
maximum: 1000
exclusiveMinimum: 0
exclusiveMaximum: 1001
multipleOf: 2
`,
		},
		{
			name: "valid array schema with all array properties",
			yml: `
type: array
title: Array Schema
description: A comprehensive array schema
items:
  type: string
  pattern: "^[a-z]+$"
minItems: 1
maxItems: 10
uniqueItems: true
prefixItems:
  - type: string
  - type: integer
contains:
  type: string
minContains: 1
maxContains: 5
unevaluatedItems:
  type: boolean
`,
		},
		{
			name: "valid object schema with all object properties",
			yml: `
type: object
title: Object Schema
description: A comprehensive object schema
properties:
  name:
    type: string
  age:
    type: integer
    minimum: 0
  active:
    type: boolean
required:
  - name
minProperties: 1
maxProperties: 10
additionalProperties:
  type: string
patternProperties:
  "^x-":
    type: string
propertyNames:
  pattern: "^[a-zA-Z_][a-zA-Z0-9_]*$"
unevaluatedProperties:
  type: boolean
dependentSchemas:
  name:
    properties:
      fullName:
        type: string
`,
		},
		{
			name: "valid schema with composition keywords",
			yml: `
title: Composition Schema
description: Schema using composition keywords
allOf:
  - type: object
    properties:
      name:
        type: string
  - type: object
    properties:
      age:
        type: integer
oneOf:
  - properties:
      type:
        const: user
  - properties:
      type:
        const: admin
anyOf:
  - type: string
  - type: number
not:
  type: "null"
`,
		},
		{
			name: "valid schema with conditional keywords",
			yml: `
type: object
title: Conditional Schema
description: Schema with conditional logic
if:
  properties:
    type:
      const: premium
then:
  properties:
    features:
      type: array
      minItems: 5
else:
  properties:
    features:
      type: array
      maxItems: 3
`,
		},
		{
			name: "valid schema with discriminator",
			yml: `
type: object
title: Pet Schema
description: Schema with discriminator
discriminator:
  propertyName: petType
  mapping:
    dog: "#/components/schemas/Dog"
    cat: "#/components/schemas/Cat"
oneOf:
  - $ref: "#/components/schemas/Dog"
  - $ref: "#/components/schemas/Cat"
`,
		},
		{
			name: "valid schema with external docs",
			yml: `
type: string
title: Documented String
description: A string with external documentation
externalDocs:
  description: More information
  url: https://example.com/docs
`,
		},
		{
			name: "valid schema with XML metadata",
			yml: `
type: object
title: XML Schema
description: Schema with XML metadata
xml:
  name: user
  namespace: https://example.com/schema
  prefix: ex
  attribute: false
  wrapped: true
`,
		},
		{
			name: "valid schema with enum and const",
			yml: `
type: string
title: Enum Schema
description: Schema with enumeration
enum:
  - "red"
  - "green"
  - "blue"
`,
		},
		{
			name: "valid schema with const value",
			yml: `
title: Const Schema
description: Schema with constant value
const: "fixed-value"
`,
		},
		{
			name: "valid schema with default and examples",
			yml: `
type: string
title: Default Schema
description: Schema with default and examples
default: "default-value"
examples:
  - "example1"
  - "example2"
  - "example3"
example: "single-example"
`,
		},
		{
			name: "valid schema with OpenAPI specific properties",
			yml: `
type: string
title: OpenAPI Schema
description: Schema with OpenAPI-specific properties
nullable: true
readOnly: false
writeOnly: false
deprecated: false
`,
		},
		{
			name: "valid schema with reference",
			yml: `
$ref: "#/components/schemas/User"
`,
		},
		{
			name: "valid schema with anchor",
			yml: `
type: object
title: Anchored Schema
$anchor: user-schema
properties:
  name:
    type: string
`,
		},
		{
			name: "valid schema with schema keyword",
			yml: `
$schema: "https://json-schema.org/draft/2020-12/schema"
type: object
title: Schema with Schema Keyword
properties:
  name:
    type: string
`,
		},
		{
			name: "valid schema with extensions",
			yml: `
type: string
title: Extended Schema
description: Schema with custom extensions
x-test: some-value
x-custom: custom-value
x-validation: strict
`,
		},
		{
			name: "valid complex nested schema",
			yml: `
type: object
title: Complex Nested Schema
description: A complex schema with nested structures
properties:
  user:
    type: object
    properties:
      profile:
        type: object
        properties:
          name:
            type: string
            minLength: 1
          contacts:
            type: array
            items:
              type: object
              properties:
                type:
                  type: string
                  enum: ["email", "phone"]
                value:
                  type: string
              required: ["type", "value"]
        required: ["name"]
    required: ["profile"]
  metadata:
    type: object
    additionalProperties:
      oneOf:
        - type: string
        - type: number
        - type: boolean
required: ["user"]
`,
		},
		{
			name: "valid schema with $ref and additional properties (OpenAPI 3.1)",
			yml: `
$ref: "#/components/schemas/User"
required: ["name", "email"]
description: "User schema with additional validation requirements"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := schema.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, schema.Valid, "expected schema to be valid")
		})
	}
}

func TestSchema_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing external docs URL",
			yml: `
type: string
title: Missing External Docs URL
externalDocs:
  description: More information
`,
			wantErrs: []string{
				"[2:1] schema.externalDocs missing property 'url'",
				"[5:3] externalDocumentation.url is missing",
			},
		},
		{
			name: "invalid type property",
			yml: `
type: invalid_type
title: Invalid Type
`,
			wantErrs: []string{
				"[2:7] schema.type expected array, got string",
				"[2:7] schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
			},
		},
		{
			name: "negative minLength",
			yml: `
type: string
minLength: -1
`,
			wantErrs: []string{"[3:12] schema.minLength minimum: got -1, want 0"},
		},
		{
			name: "negative multipleOf",
			yml: `
type: number
multipleOf: -1
`,
			wantErrs: []string{"[3:13] schema.multipleOf exclusiveMinimum: got -1, want 0"},
		},
		{
			name: "zero multipleOf",
			yml: `
type: number
multipleOf: 0
`,
			wantErrs: []string{"[3:13] schema.multipleOf exclusiveMinimum: got 0, want 0"},
		},
		{
			name: "invalid additionalProperties type",
			yml: `
type: object
additionalProperties: "invalid"
`,
			wantErrs: []string{
				"[2:1] schema.additionalProperties expected one of [boolean, object], got string",
				"[2:1] schema.additionalProperties expected one of [boolean, object], got string",
				"[3:23] schema.additionalProperties failed to validate either Schema [schema.additionalProperties expected object, got `invalid`] or bool [schema.additionalProperties line 3: cannot unmarshal !!str `invalid` into bool]",
			},
		},
		{
			name: "negative minItems",
			yml: `
type: array
minItems: -1
`,
			wantErrs: []string{"[3:11] schema.minItems minimum: got -1, want 0"},
		},
		{
			name: "negative minProperties",
			yml: `
type: object
minProperties: -1
`,
			wantErrs: []string{"[3:16] schema.minProperties minimum: got -1, want 0"},
		},
		{
			name: "invalid items type",
			yml: `
type: array
items: "invalid"
`,
			wantErrs: []string{
				"[2:1] schema.items expected one of [boolean, object], got string",
				"[2:1] schema.items expected one of [boolean, object], got string",
				"[3:8] schema.items failed to validate either Schema [schema.items expected object, got `invalid`] or bool [schema.items line 3: cannot unmarshal !!str `invalid` into bool]",
			},
		},
		{
			name: "invalid required not array",
			yml: `
type: object
required: "invalid"
`,
			wantErrs: []string{
				"[2:1] schema.required expected array, got string",
				"[3:11] schema.required expected sequence, got `invalid`",
			},
		},
		{
			name: "invalid allOf not array",
			yml: `
allOf: "invalid"
`,
			wantErrs: []string{
				"[2:1] schema.allOf expected array, got string",
				"[2:8] schema.allOf expected sequence, got `invalid`",
			},
		},
		{
			name: "invalid anyOf not array",
			yml: `
anyOf: "invalid"
`,
			wantErrs: []string{
				"[2:1] schema.anyOf expected array, got string",
				"[2:8] schema.anyOf expected sequence, got `invalid`",
			},
		},
		{
			name: "invalid oneOf not array",
			yml: `
oneOf: "invalid"
`,
			wantErrs: []string{
				"[2:1] schema.oneOf expected array, got string",
				"[2:8] schema.oneOf expected sequence, got `invalid`",
			},
		},
		{
			name: "$ref with additional properties not allowed in OpenAPI 3.0",
			yml: `
$schema: "https://spec.openapis.org/oas/3.0/dialect/2024-10-18"
$ref: "#/components/schemas/User"
required: ["name", "email"]
`,
			wantErrs: []string{"[2:1] schema. additional properties '$ref' not allowed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := schema.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)
			validation.SortValidationErrors(allErrors)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}
