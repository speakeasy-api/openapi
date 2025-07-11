package oas31_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestSchema_Validate_Success(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schema oas31.Schema
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &schema)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := schema.Validate(context.Background())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, schema.Valid, "expected schema to be valid")
		})
	}
}

func TestSchema_Validate_Error(t *testing.T) {
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
  description: "More information"
`,
			wantErrs: []string{"[5:3] field url is missing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schema oas31.Schema

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &schema)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := schema.Validate(context.Background())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
			}

			for _, expectedErr := range tt.wantErrs {
				found := false
				for _, errMsg := range errMessages {
					if strings.Contains(errMsg, expectedErr) {
						found = true
						break
					}
				}
				require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
			}
		})
	}
}
