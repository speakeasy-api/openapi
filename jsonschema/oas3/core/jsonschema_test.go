package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestJSONSchema_Unmarshal_BooleanValue_Success(t *testing.T) {
	t.Parallel()

	// Test case that reproduces the additionalProperties: false issue
	// This should unmarshal as a boolean (Right type) when Left type (Schema) fails with validation errors
	ctx := t.Context()

	// YAML with just a boolean value (like additionalProperties: false)
	testYaml := `false`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	// Test the exact JSONSchema type structure
	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")

	// Should have chosen the Right type (bool)
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsRight, "JSONSchema should be Right type (bool)")
	assert.False(t, target.IsLeft, "JSONSchema should not be Left type (Schema)")
	assert.False(t, target.Right.Value, "JSONSchema should have unmarshaled boolean value correctly")
}

func TestJSONSchema_Unmarshal_SchemaObject_Success(t *testing.T) {
	t.Parallel()

	// Test case that ensures schema objects still work correctly
	ctx := t.Context()

	// YAML with a schema object
	testYaml := `
type: string
minLength: 1
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	// Test the exact JSONSchema type structure
	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")

	// Should have chosen the Left type (Schema)
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")
	assert.False(t, target.IsRight, "JSONSchema should not be Right type (bool)")

	// Verify the schema was unmarshaled correctly
	require.NotNil(t, target.Left.Value.Type.Value, "Schema type should be set")
	assert.True(t, target.Left.Value.Type.Value.IsRight, "Type should be Right type (string)")
	assert.Equal(t, "string", target.Left.Value.Type.Value.Right.Value, "Type should be 'string'")
}

func TestJSONSchema_Unmarshal_TypeArray_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// YAML with type as array (tests EitherValue[[]marshaller.Node[string], string])
	testYaml := `
type: [string, number]
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")

	// Verify type array was unmarshaled
	require.NotNil(t, target.Left.Value.Type.Value, "Type should be set")
	assert.True(t, target.Left.Value.Type.Value.IsLeft, "Type should be Left type (array)")
	assert.Len(t, target.Left.Value.Type.Value.Left.Value, 2, "Should have 2 types")
}

func TestJSONSchema_Unmarshal_PropertiesWithAdditionalProperties_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// YAML with properties and additionalProperties (tests sequencedmap and nested schemas)
	testYaml := `
type: object
properties:
  name:
    type: string
  age:
    type: integer
additionalProperties:
  type: string
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")
	require.NotNil(t, target, "JSONSchema should not be nil")

	// Verify properties map
	require.NotNil(t, target.Left.Value.Properties.Value, "Properties should be set")
	assert.Equal(t, 2, target.Left.Value.Properties.Value.Len(), "Should have 2 properties")

	// Verify additionalProperties schema
	require.NotNil(t, target.Left.Value.AdditionalProperties.Value, "AdditionalProperties should be set")
	assert.True(t, target.Left.Value.AdditionalProperties.Value.IsLeft, "AdditionalProperties should be schema")
}

func TestJSONSchema_Unmarshal_WithDiscriminator_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// YAML with discriminator (tests Discriminator type registration)
	testYaml := `
type: object
discriminator:
  propertyName: petType
  mapping:
    dog: "#/components/schemas/Dog"
    cat: "#/components/schemas/Cat"
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")

	// Verify discriminator was unmarshaled
	require.NotNil(t, target.Left.Value.Discriminator.Value, "Discriminator should be set")
	assert.Equal(t, "petType", target.Left.Value.Discriminator.Value.PropertyName.Value, "Should parse propertyName")
	require.NotNil(t, target.Left.Value.Discriminator.Value.Mapping.Value, "Mapping should be set")
	assert.Equal(t, 2, target.Left.Value.Discriminator.Value.Mapping.Value.Len(), "Should have 2 mappings")
}

func TestJSONSchema_Unmarshal_WithExternalDocs_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// YAML with externalDocs (tests ExternalDocumentation type registration)
	testYaml := `
type: string
description: A user identifier
externalDocs:
  url: https://example.com/docs/user-id
  description: User ID documentation
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")

	// Verify externalDocs was unmarshaled
	require.NotNil(t, target.Left.Value.ExternalDocs.Value, "ExternalDocs should be set")
	assert.Equal(t, "https://example.com/docs/user-id", target.Left.Value.ExternalDocs.Value.URL.Value, "Should parse URL")
	require.NotNil(t, target.Left.Value.ExternalDocs.Value.Description.Value, "Description should be set")
	assert.Equal(t, "User ID documentation", *target.Left.Value.ExternalDocs.Value.Description.Value, "Should parse description")
}

func TestJSONSchema_Unmarshal_WithXML_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// YAML with xml (tests XML type registration)
	testYaml := `
type: object
xml:
  name: Person
  namespace: http://example.com/schema
  prefix: per
  wrapped: true
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")

	// Verify xml was unmarshaled
	require.NotNil(t, target.Left.Value.XML.Value, "XML should be set")
	require.NotNil(t, target.Left.Value.XML.Value.Name.Value, "Name should be set")
	assert.Equal(t, "Person", *target.Left.Value.XML.Value.Name.Value, "Should parse name")
	require.NotNil(t, target.Left.Value.XML.Value.Namespace.Value, "Namespace should be set")
	assert.Equal(t, "http://example.com/schema", *target.Left.Value.XML.Value.Namespace.Value, "Should parse namespace")
	require.NotNil(t, target.Left.Value.XML.Value.Prefix.Value, "Prefix should be set")
	assert.Equal(t, "per", *target.Left.Value.XML.Value.Prefix.Value, "Should parse prefix")
	require.NotNil(t, target.Left.Value.XML.Value.Wrapped.Value, "Wrapped should be set")
	assert.True(t, *target.Left.Value.XML.Value.Wrapped.Value, "Should parse wrapped as true")
}

func TestJSONSchema_Unmarshal_ComplexSchema_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// YAML with multiple nested features to test all registrations together
	testYaml := `
type: object
properties:
  id:
    type: string
    xml:
      attribute: true
  name:
    type: string
discriminator:
  propertyName: type
externalDocs:
  url: https://example.com/docs
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")

	// Verify properties
	require.NotNil(t, target.Left.Value.Properties.Value, "Properties should be set")
	assert.Equal(t, 2, target.Left.Value.Properties.Value.Len(), "Should have 2 properties")

	// Verify id property has xml
	idProp, found := target.Left.Value.Properties.Value.Get("id")
	require.True(t, found, "Should find id property")
	require.NotNil(t, idProp, "id property should not be nil")
	require.NotNil(t, idProp.Left.Value.XML.Value, "id should have XML")
	require.NotNil(t, idProp.Left.Value.XML.Value.Attribute.Value, "XML attribute should be set")
	assert.True(t, *idProp.Left.Value.XML.Value.Attribute.Value, "XML attribute should be true")

	// Verify discriminator
	require.NotNil(t, target.Left.Value.Discriminator.Value, "Discriminator should be set")
	assert.Equal(t, "type", target.Left.Value.Discriminator.Value.PropertyName.Value, "Should parse discriminator propertyName")

	// Verify externalDocs
	require.NotNil(t, target.Left.Value.ExternalDocs.Value, "ExternalDocs should be set")
	assert.Equal(t, "https://example.com/docs", target.Left.Value.ExternalDocs.Value.URL.Value, "Should parse externalDocs URL")
}
