package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
$ref: "#/components/schemas/BaseUser"
type: object
title: Comprehensive User Schema
description: A comprehensive schema representing a user with all possible properties
$anchor: user-schema
$id: "https://example.com/schemas/user"
$schema: "https://json-schema.org/draft/2020-12/schema"
format: object
pattern: "^user_"
multipleOf: 1.0
minimum: 0.0
maximum: 1000.0
exclusiveMinimum: true
exclusiveMaximum: false
minLength: 1
maxLength: 255
minItems: 0
maxItems: 100
uniqueItems: true
minProperties: 1
maxProperties: 50
minContains: 1
maxContains: 10
nullable: true
readOnly: false
writeOnly: false
deprecated: false
default: "default-user"
const: "constant-value"
enum:
  - "admin"
  - "user"
  - "guest"
examples:
  - "example1"
  - "example2"
example: "single-example"
properties:
  id:
    type: integer
    description: User ID
    minimum: 1
  name:
    type: string
    description: User's full name
    minLength: 1
    maxLength: 100
  email:
    type: string
    format: email
    description: User's email address
  age:
    type: integer
    minimum: 0
    maximum: 150
  tags:
    type: array
    items:
      type: string
    minItems: 0
    maxItems: 10
    uniqueItems: true
  metadata:
    type: object
    additionalProperties:
      type: string
required:
  - id
  - name
  - email
additionalProperties:
  type: string
patternProperties:
  "^x-":
    type: string
propertyNames:
  pattern: "^[a-zA-Z_][a-zA-Z0-9_]*$"
unevaluatedProperties:
  type: boolean
unevaluatedItems:
  type: string
dependentSchemas:
  name:
    properties:
      fullName:
        type: string
allOf:
  - type: object
    properties:
      baseField:
        type: string
oneOf:
  - properties:
      type:
        const: premium
  - properties:
      type:
        const: basic
anyOf:
  - type: object
  - type: string
not:
  type: "null"
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
items:
  type: string
prefixItems:
  - type: string
  - type: integer
contains:
  type: string
discriminator:
  propertyName: userType
  mapping:
    admin: "#/components/schemas/AdminUser"
    regular: "#/components/schemas/RegularUser"
    guest: "#/components/schemas/GuestUser"
externalDocs:
  description: Comprehensive user documentation
  url: https://example.com/user-docs
xml:
  name: user
  namespace: https://example.com/schema
  prefix: usr
  attribute: false
  wrapped: true
x-test: some-value
x-custom: custom-value
x-validation: strict
x-metadata:
  version: "1.0"
  author: "test"
`

	var schema oas3.Schema

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Test basic string fields
	require.Equal(t, "#/components/schemas/BaseUser", string(schema.GetRef()))
	require.Equal(t, "Comprehensive User Schema", schema.GetTitle())
	require.Equal(t, "A comprehensive schema representing a user with all possible properties", schema.GetDescription())
	require.Equal(t, "object", schema.GetFormat())
	require.Equal(t, "^user_", schema.GetPattern())

	// Test anchor, $id, and schema
	require.NotNil(t, schema.Anchor)
	require.Equal(t, "user-schema", *schema.Anchor)
	require.NotNil(t, schema.ID)
	require.Equal(t, "https://example.com/schemas/user", *schema.ID)
	require.Equal(t, "https://example.com/schemas/user", schema.GetID())
	require.NotNil(t, schema.Schema)
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", *schema.Schema)

	// Test numeric constraints
	require.NotNil(t, schema.MultipleOf)
	require.InDelta(t, 1.0, *schema.MultipleOf, 0.001)
	require.InDelta(t, 0.0, *schema.GetMinimum(), 0.001)
	require.InDelta(t, 1000.0, *schema.GetMaximum(), 0.001)
	require.NotNil(t, schema.ExclusiveMinimum)
	require.NotNil(t, schema.ExclusiveMaximum)

	// Test string constraints
	require.Equal(t, int64(1), *schema.GetMinLength())
	require.Equal(t, int64(255), *schema.GetMaxLength())

	// Test array constraints
	require.Equal(t, int64(0), schema.GetMinItems())
	require.Equal(t, int64(100), *schema.GetMaxItems())
	require.True(t, schema.GetUniqueItems())
	require.Equal(t, int64(1), *schema.MinContains)
	require.Equal(t, int64(10), *schema.MaxContains)

	// Test object constraints
	require.Equal(t, int64(1), *schema.GetMinProperties())
	require.Equal(t, int64(50), *schema.GetMaxProperties())

	// Test OpenAPI specific properties
	require.True(t, schema.GetNullable())
	require.False(t, schema.GetReadOnly())
	require.False(t, schema.GetWriteOnly())
	require.False(t, schema.GetDeprecated())

	// Test default, const, and examples
	require.NotNil(t, schema.Default)
	require.NotNil(t, schema.Const)
	require.NotNil(t, schema.Example)
	require.Len(t, schema.Examples, 2)

	// Test enum
	require.Len(t, schema.GetEnum(), 3)

	// Test type
	types := schema.GetType()
	require.Len(t, types, 1)
	require.Equal(t, oas3.SchemaTypeObject, types[0])

	// Test properties
	require.NotNil(t, schema.Properties)
	require.Equal(t, 6, schema.Properties.Len())

	idSchema, ok := schema.Properties.Get("id")
	require.True(t, ok)
	require.NotNil(t, idSchema)
	require.NotNil(t, idSchema.GetRootNode())

	nameSchema, ok := schema.Properties.Get("name")
	require.True(t, ok)
	require.NotNil(t, nameSchema)

	emailSchema, ok := schema.Properties.Get("email")
	require.True(t, ok)
	require.NotNil(t, emailSchema)

	ageSchema, ok := schema.Properties.Get("age")
	require.True(t, ok)
	require.NotNil(t, ageSchema)

	tagsSchema, ok := schema.Properties.Get("tags")
	require.True(t, ok)
	require.NotNil(t, tagsSchema)

	metadataSchema, ok := schema.Properties.Get("metadata")
	require.True(t, ok)
	require.NotNil(t, metadataSchema)

	// Test required
	require.Len(t, schema.Required, 3)
	require.Contains(t, schema.Required, "id")
	require.Contains(t, schema.Required, "name")
	require.Contains(t, schema.Required, "email")

	// Test additional properties and pattern properties
	require.NotNil(t, schema.AdditionalProperties)
	require.NotNil(t, schema.PatternProperties)
	require.Equal(t, 1, schema.PatternProperties.Len())

	// Test property names and unevaluated properties
	require.NotNil(t, schema.PropertyNames)
	require.NotNil(t, schema.UnevaluatedProperties)
	require.NotNil(t, schema.UnevaluatedItems)

	// Test dependent schemas
	require.NotNil(t, schema.DependentSchemas)
	require.Equal(t, 1, schema.DependentSchemas.Len())

	// Test composition keywords
	require.Len(t, schema.GetAllOf(), 1)
	require.Len(t, schema.GetOneOf(), 2)
	require.Len(t, schema.GetAnyOf(), 2)
	require.NotNil(t, schema.GetNot())

	// Test conditional keywords
	require.NotNil(t, schema.If)
	require.NotNil(t, schema.Then)
	require.NotNil(t, schema.Else)

	// Test array-specific keywords
	require.NotNil(t, schema.Items)
	require.Len(t, schema.PrefixItems, 2)
	require.NotNil(t, schema.Contains)

	// Test discriminator
	require.NotNil(t, schema.Discriminator)
	require.Equal(t, "userType", schema.Discriminator.GetPropertyName())

	mapping := schema.Discriminator.GetMapping()
	require.NotNil(t, mapping)
	require.Equal(t, 3, mapping.Len())

	adminRef, ok := mapping.Get("admin")
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/AdminUser", adminRef)

	regularRef, ok := mapping.Get("regular")
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/RegularUser", regularRef)

	guestRef, ok := mapping.Get("guest")
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/GuestUser", guestRef)

	// Test external docs
	require.NotNil(t, schema.ExternalDocs)
	require.Equal(t, "Comprehensive user documentation", schema.ExternalDocs.GetDescription())
	require.Equal(t, "https://example.com/user-docs", schema.ExternalDocs.GetURL())

	// Test XML metadata
	require.NotNil(t, schema.XML)
	require.Equal(t, "user", schema.XML.GetName())
	require.Equal(t, "https://example.com/schema", schema.XML.GetNamespace())
	require.Equal(t, "usr", schema.XML.GetPrefix())
	require.False(t, schema.XML.GetAttribute())
	require.True(t, schema.XML.GetWrapped())

	// Test extensions
	extensions := schema.GetExtensions()
	require.NotNil(t, extensions)

	ext, ok := extensions.Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)

	ext, ok = extensions.Get("x-custom")
	require.True(t, ok)
	require.Equal(t, "custom-value", ext.Value)

	ext, ok = extensions.Get("x-validation")
	require.True(t, ok)
	require.Equal(t, "strict", ext.Value)

	ext, ok = extensions.Get("x-metadata")
	require.True(t, ok)
	require.NotNil(t, ext.Value)
}

func TestSchema_Unmarshal_MergeKeys_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yml           string
		expectedProps []string
		checkOverride bool
		overrideKey   string
		overrideDesc  string
	}{
		{
			name: "merge key in properties inherits from anchor",
			yml: `
type: object
properties:
  id: &base_id
    type: integer
    description: Base ID
  name:
    type: string
    description: User name
`,
			expectedProps: []string{"id", "name"},
		},
		{
			name: "merge key expands aliased mapping into properties",
			yml: `
type: object
properties:
  <<:
    id:
      type: integer
      description: Base ID
    name:
      type: string
      description: Base name
  email:
    type: string
    description: Email address
`,
			expectedProps: []string{"id", "name", "email"},
		},
		{
			name: "merge key with alias expands into properties",
			// NOTE: YAML anchors/aliases must be defined in the same document.
			// We define the anchor at a top-level key and reference it in properties.
			yml: `
type: object
x-base-props: &base_props
  id:
    type: integer
    description: Base ID
  name:
    type: string
    description: Base name
properties:
  <<: *base_props
  email:
    type: string
    description: Email address
`,
			expectedProps: []string{"id", "name", "email"},
		},
		{
			name: "merge key with override in properties",
			yml: `
type: object
x-base-props: &base_props
  id:
    type: integer
    description: Base ID
  name:
    type: string
    description: Base name
properties:
  <<: *base_props
  name:
    type: string
    description: Overridden name
  email:
    type: string
    description: Email address
`,
			expectedProps: []string{"id", "name", "email"},
			checkOverride: true,
			overrideKey:   "name",
			overrideDesc:  "Overridden name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should have no validation errors")

			require.NotNil(t, schema.Properties, "properties should not be nil")

			_, hasMergeKey := schema.Properties.Get("<<")
			assert.False(t, hasMergeKey, "merge key '<<' should not appear as a property name")

			for _, prop := range tt.expectedProps {
				propSchema, ok := schema.Properties.Get(prop)
				assert.True(t, ok, "property %q should exist", prop)
				assert.NotNil(t, propSchema, "property %q schema should not be nil", prop)
			}

			assert.Equal(t, len(tt.expectedProps), schema.Properties.Len(), "should have expected number of properties")

			if tt.checkOverride {
				propSchema, ok := schema.Properties.Get(tt.overrideKey)
				require.True(t, ok, "override property %q should exist", tt.overrideKey)
				require.True(t, propSchema.IsSchema(), "override property %q should be a schema", tt.overrideKey)
				assert.Equal(t, tt.overrideDesc, propSchema.GetSchema().GetDescription(), "overridden property should have the overriding description")
			}
		})
	}
}

func TestSchema_Unmarshal_MergeKeysAtModelLevel_Success(t *testing.T) {
	t.Parallel()

	yml := `
x-base: &base
  type: object
  description: Base schema description
<<: *base
title: Extended Schema
properties:
  id:
    type: integer
`

	var schema oas3.Schema

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	types := schema.GetType()
	require.Len(t, types, 1, "should have type from merge")
	assert.Equal(t, oas3.SchemaTypeObject, types[0], "merged type should be object")
	assert.Equal(t, "Base schema description", schema.GetDescription(), "should have description from merge")
	assert.Equal(t, "Extended Schema", schema.GetTitle(), "explicit title should be present")

	require.NotNil(t, schema.Properties, "properties should not be nil")
	_, ok := schema.Properties.Get("id")
	assert.True(t, ok, "property 'id' should exist")
}

func TestSchema_Unmarshal_NestedMergeChain_Success(t *testing.T) {
	t.Parallel()

	// base1 defines {id, name}, base2 merges base1 and adds {email},
	// final schema merges base2 and adds {age}.
	// All four properties should be present via recursive merge resolution.
	yml := `
x-base1: &base1
  id:
    type: integer
  name:
    type: string
x-base2: &base2
  <<: *base1
  email:
    type: string
type: object
properties:
  <<: *base2
  age:
    type: integer
`

	var schema oas3.Schema

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	require.NotNil(t, schema.Properties, "properties should not be nil")

	_, hasMergeKey := schema.Properties.Get("<<")
	assert.False(t, hasMergeKey, "merge key '<<' should not appear as a property name")

	expectedProps := []string{"id", "name", "email", "age"}
	for _, prop := range expectedProps {
		propSchema, ok := schema.Properties.Get(prop)
		assert.True(t, ok, "property %q should exist", prop)
		assert.NotNil(t, propSchema, "property %q schema should not be nil", prop)
	}

	assert.Equal(t, len(expectedProps), schema.Properties.Len(), "should have all 4 properties from nested merge chain")
}

func TestSchema_Unmarshal_QuotedLiteralMergeKey_Success(t *testing.T) {
	t.Parallel()

	// A quoted '<<' is a regular string key, not a merge key.
	// It should be treated as a normal property name.
	yml := `
type: object
properties:
  '<<':
    type: string
    description: This is a literal property named <<
  name:
    type: string
`

	var schema oas3.Schema

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	require.NotNil(t, schema.Properties, "properties should not be nil")
	assert.Equal(t, 2, schema.Properties.Len(), "should have 2 properties")

	propSchema, ok := schema.Properties.Get("<<")
	assert.True(t, ok, "quoted '<<' should be a normal property")
	assert.NotNil(t, propSchema, "quoted '<<' property schema should not be nil")

	_, ok = schema.Properties.Get("name")
	assert.True(t, ok, "property 'name' should exist")
}
