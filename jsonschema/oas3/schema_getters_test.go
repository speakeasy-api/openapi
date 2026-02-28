package oas3_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSchema_GetExclusiveMaximum_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema   *oas3.Schema
		expected oas3.ExclusiveMaximum
	}{
		{
			name:     "nil schema returns nil",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "schema with no exclusive maximum returns nil",
			schema:   &oas3.Schema{},
			expected: nil,
		},
		{
			name: "schema with exclusive maximum returns value",
			schema: &oas3.Schema{
				ExclusiveMaximum: oas3.NewExclusiveMaximumFromFloat64(100.0),
			},
			expected: oas3.NewExclusiveMaximumFromFloat64(100.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.schema.GetExclusiveMaximum()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.True(t, result.IsEqual(tt.expected))
			}
		})
	}
}

func TestSchema_GetExclusiveMinimum_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema   *oas3.Schema
		expected oas3.ExclusiveMinimum
	}{
		{
			name:     "nil schema returns nil",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "schema with no exclusive minimum returns nil",
			schema:   &oas3.Schema{},
			expected: nil,
		},
		{
			name: "schema with exclusive minimum returns value",
			schema: &oas3.Schema{
				ExclusiveMinimum: oas3.NewExclusiveMinimumFromFloat64(0.0),
			},
			expected: oas3.NewExclusiveMinimumFromFloat64(0.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.schema.GetExclusiveMinimum()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.True(t, result.IsEqual(tt.expected))
			}
		})
	}
}

func TestSchema_GetDiscriminator_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema   *oas3.Schema
		expected *oas3.Discriminator
	}{
		{
			name:     "nil schema returns nil",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "schema with no discriminator returns nil",
			schema:   &oas3.Schema{},
			expected: nil,
		},
		{
			name: "schema with discriminator returns value",
			schema: &oas3.Schema{
				Discriminator: &oas3.Discriminator{PropertyName: "type"},
			},
			expected: &oas3.Discriminator{PropertyName: "type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.schema.GetDiscriminator()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchema_GetExamples_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema   *oas3.Schema
		expected []values.Value
	}{
		{
			name:     "nil schema returns nil",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "schema with no examples returns nil",
			schema:   &oas3.Schema{},
			expected: nil,
		},
		{
			name: "schema with examples returns values",
			schema: &oas3.Schema{
				Examples: []values.Value{
					&yaml.Node{Kind: yaml.ScalarNode, Value: "example1"},
				},
			},
			expected: []values.Value{
				&yaml.Node{Kind: yaml.ScalarNode, Value: "example1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.schema.GetExamples()
			assert.Len(t, result, len(tt.expected))
		})
	}
}

func TestSchema_GetPrefixItems_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetPrefixItems())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetPrefixItems())

	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("test")})
	schemaWithPrefixItems := &oas3.Schema{
		PrefixItems: []*oas3.JSONSchema[oas3.Referenceable]{schema},
	}
	result := schemaWithPrefixItems.GetPrefixItems()
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestSchema_GetContains_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetContains())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetContains())

	childSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("contained")})
	schemaWithContains := &oas3.Schema{Contains: childSchema}
	assert.Equal(t, childSchema, schemaWithContains.GetContains())
}

func TestSchema_GetMinContains_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMinContains())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMinContains())

	minContains := int64(1)
	schemaWithMinContains := &oas3.Schema{MinContains: &minContains}
	assert.Equal(t, &minContains, schemaWithMinContains.GetMinContains())
}

func TestSchema_GetMaxContains_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMaxContains())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMaxContains())

	maxContains := int64(10)
	schemaWithMaxContains := &oas3.Schema{MaxContains: &maxContains}
	assert.Equal(t, &maxContains, schemaWithMaxContains.GetMaxContains())
}

func TestSchema_GetIf_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetIf())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetIf())

	ifSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("if")})
	schemaWithIf := &oas3.Schema{If: ifSchema}
	assert.Equal(t, ifSchema, schemaWithIf.GetIf())
}

func TestSchema_GetElse_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetElse())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetElse())

	elseSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("else")})
	schemaWithElse := &oas3.Schema{Else: elseSchema}
	assert.Equal(t, elseSchema, schemaWithElse.GetElse())
}

func TestSchema_GetThen_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetThen())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetThen())

	thenSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("then")})
	schemaWithThen := &oas3.Schema{Then: thenSchema}
	assert.Equal(t, thenSchema, schemaWithThen.GetThen())
}

func TestSchema_GetDependentSchemas_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetDependentSchemas())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetDependentSchemas())

	depSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("dependent")})
	depSchemas := sequencedmap.New(sequencedmap.NewElem("dep1", depSchema))
	schemaWithDeps := &oas3.Schema{DependentSchemas: depSchemas}
	assert.Equal(t, depSchemas, schemaWithDeps.GetDependentSchemas())
}

func TestSchema_GetPatternProperties_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetPatternProperties())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetPatternProperties())

	patternSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("pattern")})
	patternProps := sequencedmap.New(sequencedmap.NewElem("^[a-z]+$", patternSchema))
	schemaWithPatterns := &oas3.Schema{PatternProperties: patternProps}
	assert.Equal(t, patternProps, schemaWithPatterns.GetPatternProperties())
}

func TestSchema_GetPropertyNames_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetPropertyNames())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetPropertyNames())

	propNamesSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Pattern: pointer.From("^[a-z]+$")})
	schemaWithPropNames := &oas3.Schema{PropertyNames: propNamesSchema}
	assert.Equal(t, propNamesSchema, schemaWithPropNames.GetPropertyNames())
}

func TestSchema_GetUnevaluatedItems_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetUnevaluatedItems())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetUnevaluatedItems())

	unevalItemsSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("unevaluated")})
	schemaWithUnevalItems := &oas3.Schema{UnevaluatedItems: unevalItemsSchema}
	assert.Equal(t, unevalItemsSchema, schemaWithUnevalItems.GetUnevaluatedItems())
}

func TestSchema_GetUnevaluatedProperties_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetUnevaluatedProperties())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetUnevaluatedProperties())

	unevalPropsSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("unevaluated")})
	schemaWithUnevalProps := &oas3.Schema{UnevaluatedProperties: unevalPropsSchema}
	assert.Equal(t, unevalPropsSchema, schemaWithUnevalProps.GetUnevaluatedProperties())
}

func TestSchema_GetItems_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetItems())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetItems())

	itemsSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Type: oas3.NewTypeFromString("string")})
	schemaWithItems := &oas3.Schema{Items: itemsSchema}
	assert.Equal(t, itemsSchema, schemaWithItems.GetItems())
}

func TestSchema_GetAnchor_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Empty(t, nilSchema.GetAnchor())

	emptySchema := &oas3.Schema{}
	assert.Empty(t, emptySchema.GetAnchor())

	anchor := "myAnchor"
	schemaWithAnchor := &oas3.Schema{Anchor: &anchor}
	assert.Equal(t, anchor, schemaWithAnchor.GetAnchor())
}

func TestSchema_GetID_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Empty(t, nilSchema.GetID())

	emptySchema := &oas3.Schema{}
	assert.Empty(t, emptySchema.GetID())

	id := "https://example.com/schemas/user"
	schemaWithID := &oas3.Schema{ID: &id}
	assert.Equal(t, id, schemaWithID.GetID())
}

func TestSchema_GetMultipleOf_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMultipleOf())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMultipleOf())

	multipleOf := 5.0
	schemaWithMultipleOf := &oas3.Schema{MultipleOf: &multipleOf}
	assert.Equal(t, &multipleOf, schemaWithMultipleOf.GetMultipleOf())
}

func TestSchema_GetAdditionalProperties_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetAdditionalProperties())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetAdditionalProperties())

	additionalPropsSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Type: oas3.NewTypeFromString("string")})
	schemaWithAdditionalProps := &oas3.Schema{AdditionalProperties: additionalPropsSchema}
	assert.Equal(t, additionalPropsSchema, schemaWithAdditionalProps.GetAdditionalProperties())
}

func TestSchema_GetDefault_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetDefault())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetDefault())

	defaultValue := &yaml.Node{Kind: yaml.ScalarNode, Value: "default"}
	schemaWithDefault := &oas3.Schema{Default: defaultValue}
	assert.Equal(t, defaultValue, schemaWithDefault.GetDefault())
}

func TestSchema_GetConst_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetConst())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetConst())

	constValue := &yaml.Node{Kind: yaml.ScalarNode, Value: "constant"}
	schemaWithConst := &oas3.Schema{Const: constValue}
	assert.Equal(t, constValue, schemaWithConst.GetConst())
}

func TestSchema_GetExternalDocs_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetExternalDocs())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetExternalDocs())

	externalDocs := &oas3.ExternalDocumentation{URL: "https://example.com"}
	schemaWithExternalDocs := &oas3.Schema{ExternalDocs: externalDocs}
	assert.Equal(t, externalDocs, schemaWithExternalDocs.GetExternalDocs())
}

func TestSchema_GetExample_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetExample())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetExample())

	exampleValue := &yaml.Node{Kind: yaml.ScalarNode, Value: "example"}
	schemaWithExample := &oas3.Schema{Example: exampleValue}
	assert.Equal(t, exampleValue, schemaWithExample.GetExample())
}

func TestSchema_GetSchema_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Empty(t, nilSchema.GetSchema())

	emptySchema := &oas3.Schema{}
	assert.Empty(t, emptySchema.GetSchema())

	schemaURI := "https://json-schema.org/draft/2020-12/schema"
	schemaWithSchemaURI := &oas3.Schema{Schema: &schemaURI}
	assert.Equal(t, schemaURI, schemaWithSchemaURI.GetSchema())
}

func TestSchema_GetXML_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetXML())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetXML())

	xmlMetadata := &oas3.XML{Name: pointer.From("MyElement")}
	schemaWithXML := &oas3.Schema{XML: xmlMetadata}
	assert.Equal(t, xmlMetadata, schemaWithXML.GetXML())
}

func TestJSONSchema_NewJSONSchemaFromReference_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/schemas/User")
	schema := oas3.NewJSONSchemaFromReference(ref)

	assert.NotNil(t, schema)
	assert.True(t, schema.IsReference())
	concreteSchema := schema.GetSchema()
	assert.NotNil(t, concreteSchema)
	assert.Equal(t, ref, concreteSchema.GetRef())
}

func TestJSONSchema_NewJSONSchemaFromBool_Success(t *testing.T) {
	t.Parallel()

	trueSchema := oas3.NewJSONSchemaFromBool(true)
	assert.NotNil(t, trueSchema)
	assert.True(t, trueSchema.IsBool())
	assert.True(t, *trueSchema.GetBool())

	falseSchema := oas3.NewJSONSchemaFromBool(false)
	assert.NotNil(t, falseSchema)
	assert.True(t, falseSchema.IsBool())
	assert.False(t, *falseSchema.GetBool())
}

func TestJSONSchema_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	schema := &oas3.Schema{}
	// Should return empty extensions if not set
	exts := schema.GetExtensions()
	assert.NotNil(t, exts)
	assert.Equal(t, 0, exts.Len())

	// With extensions set
	extensions := extensions.New()
	extensions.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value"})
	schemaWithExts := &oas3.Schema{Extensions: extensions}
	result := schemaWithExts.GetExtensions()
	assert.Equal(t, extensions, result)
}

func TestJSONSchema_GetReference_Success(t *testing.T) {
	t.Parallel()

	// Schema without reference
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("Test")})
	ref := schema.GetReference()
	assert.Equal(t, references.Reference(""), ref)

	// Schema with reference
	refSchema := oas3.NewJSONSchemaFromReference("#/components/schemas/User")
	ref = refSchema.GetReference()
	assert.Equal(t, references.Reference("#/components/schemas/User"), ref)
}

func TestJSONSchema_GetResolvedObject_Success(t *testing.T) {
	t.Parallel()

	// Non-reference schema - GetResolvedObject converts to Concrete type
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("Test")})
	resolved := schema.GetResolvedObject()
	assert.NotNil(t, resolved)
	assert.Equal(t, "Test", resolved.GetSchema().GetTitle())
}

func TestJSONSchema_GetReferenceResolutionInfo_Success(t *testing.T) {
	t.Parallel()

	// Non-reference schema returns nil info (it's not a reference)
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("Test")})
	info := schema.GetReferenceResolutionInfo()
	// For non-reference schemas, this returns nil
	assert.Nil(t, info)
}

func TestSchema_GetAllOf_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetAllOf())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetAllOf())

	childSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("child")})
	schemaWithAllOf := &oas3.Schema{AllOf: []*oas3.JSONSchema[oas3.Referenceable]{childSchema}}
	result := schemaWithAllOf.GetAllOf()
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestSchema_GetOneOf_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetOneOf())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetOneOf())

	childSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("child")})
	schemaWithOneOf := &oas3.Schema{OneOf: []*oas3.JSONSchema[oas3.Referenceable]{childSchema}}
	result := schemaWithOneOf.GetOneOf()
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestSchema_GetAnyOf_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetAnyOf())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetAnyOf())

	childSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("child")})
	schemaWithAnyOf := &oas3.Schema{AnyOf: []*oas3.JSONSchema[oas3.Referenceable]{childSchema}}
	result := schemaWithAnyOf.GetAnyOf()
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestSchema_GetNot_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetNot())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetNot())

	notSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("not")})
	schemaWithNot := &oas3.Schema{Not: notSchema}
	assert.Equal(t, notSchema, schemaWithNot.GetNot())
}

func TestSchema_GetProperties_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetProperties())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetProperties())

	propSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("prop")})
	props := sequencedmap.New(sequencedmap.NewElem("name", propSchema))
	schemaWithProps := &oas3.Schema{Properties: props}
	assert.Equal(t, props, schemaWithProps.GetProperties())
}

func TestSchema_GetDefs_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetDefs())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetDefs())

	defSchema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{Title: pointer.From("def")})
	defs := sequencedmap.New(sequencedmap.NewElem("MyDef", defSchema))
	schemaWithDefs := &oas3.Schema{Defs: defs}
	assert.Equal(t, defs, schemaWithDefs.GetDefs())
}

func TestSchema_GetTitle_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Empty(t, nilSchema.GetTitle())

	emptySchema := &oas3.Schema{}
	assert.Empty(t, emptySchema.GetTitle())

	schemaWithTitle := &oas3.Schema{Title: pointer.From("My Title")}
	assert.Equal(t, "My Title", schemaWithTitle.GetTitle())
}

func TestSchema_GetMaximum_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMaximum())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMaximum())

	maximum := 100.0
	schemaWithMaximum := &oas3.Schema{Maximum: &maximum}
	assert.Equal(t, &maximum, schemaWithMaximum.GetMaximum())
}

func TestSchema_GetMinimum_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMinimum())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMinimum())

	minimum := 0.0
	schemaWithMinimum := &oas3.Schema{Minimum: &minimum}
	assert.Equal(t, &minimum, schemaWithMinimum.GetMinimum())
}

func TestSchema_GetMaxLength_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMaxLength())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMaxLength())

	maxLength := int64(100)
	schemaWithMaxLength := &oas3.Schema{MaxLength: &maxLength}
	assert.Equal(t, &maxLength, schemaWithMaxLength.GetMaxLength())
}

func TestSchema_GetMinLength_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMinLength())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMinLength())

	minLength := int64(1)
	schemaWithMinLength := &oas3.Schema{MinLength: &minLength}
	assert.Equal(t, &minLength, schemaWithMinLength.GetMinLength())
}

func TestSchema_GetPattern_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Empty(t, nilSchema.GetPattern())

	emptySchema := &oas3.Schema{}
	assert.Empty(t, emptySchema.GetPattern())

	schemaWithPattern := &oas3.Schema{Pattern: pointer.From("^[a-z]+$")}
	assert.Equal(t, "^[a-z]+$", schemaWithPattern.GetPattern())
}

func TestSchema_GetFormat_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Empty(t, nilSchema.GetFormat())

	emptySchema := &oas3.Schema{}
	assert.Empty(t, emptySchema.GetFormat())

	schemaWithFormat := &oas3.Schema{Format: pointer.From("date-time")}
	assert.Equal(t, "date-time", schemaWithFormat.GetFormat())
}

func TestSchema_GetMaxItems_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMaxItems())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMaxItems())

	maxItems := int64(10)
	schemaWithMaxItems := &oas3.Schema{MaxItems: &maxItems}
	assert.Equal(t, &maxItems, schemaWithMaxItems.GetMaxItems())
}

func TestSchema_GetUniqueItems_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.False(t, nilSchema.GetUniqueItems())

	emptySchema := &oas3.Schema{}
	assert.False(t, emptySchema.GetUniqueItems())

	uniqueItems := true
	schemaWithUniqueItems := &oas3.Schema{UniqueItems: &uniqueItems}
	assert.True(t, schemaWithUniqueItems.GetUniqueItems())
}

func TestSchema_GetMaxProperties_Success(t *testing.T) {
	t.Parallel()

	nilSchema := (*oas3.Schema)(nil)
	assert.Nil(t, nilSchema.GetMaxProperties())

	emptySchema := &oas3.Schema{}
	assert.Nil(t, emptySchema.GetMaxProperties())

	maxProps := int64(5)
	schemaWithMaxProps := &oas3.Schema{MaxProperties: &maxProps}
	assert.Equal(t, &maxProps, schemaWithMaxProps.GetMaxProperties())
}
