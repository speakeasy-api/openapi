package oas3

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_Defs_Success(t *testing.T) {
	t.Parallel()

	t.Run("resolve reference to top-level $defs", func(t *testing.T) {
		t.Parallel()

		// Load test schema with $defs
		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/defs_schema.json")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/defs_schema.json",
			RootDocument:   root,
		}

		// Navigate to the user property which references #/$defs/User
		rootSchema := root.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		properties := rootSchema.GetSchema().GetProperties()
		require.NotNil(t, properties)

		userProperty, exists := properties.Get("user")
		require.True(t, exists)
		require.True(t, userProperty.IsReference())

		// Resolve the reference
		validationErrs, err := userProperty.Resolve(t.Context(), opts)
		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema
		result := userProperty.GetResolvedSchema()
		require.NotNil(t, result)
		assert.True(t, result.IsSchema())

		// Verify it's the User schema
		resolvedSchema := result.GetSchema()
		assert.Equal(t, []SchemaType{SchemaTypeObject}, resolvedSchema.GetType())

		// Verify it has the expected properties
		resolvedProperties := resolvedSchema.GetProperties()
		require.NotNil(t, resolvedProperties)

		nameProperty, exists := resolvedProperties.Get("name")
		require.True(t, exists)
		assert.True(t, nameProperty.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeString}, nameProperty.GetSchema().GetType())

		ageProperty, exists := resolvedProperties.Get("age")
		require.True(t, exists)
		assert.True(t, ageProperty.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeInteger}, ageProperty.GetSchema().GetType())
	})

	t.Run("resolve chained references through $defs", func(t *testing.T) {
		t.Parallel()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/defs_schema.json")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/defs_schema.json",
			RootDocument:   root,
		}

		// Navigate to the user property which references User, which itself references Address
		rootSchema := root.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		properties := rootSchema.GetSchema().GetProperties()
		require.NotNil(t, properties)

		userProperty, exists := properties.Get("user")
		require.True(t, exists)
		require.True(t, userProperty.IsReference())

		// Resolve the User reference
		validationErrs, err := userProperty.Resolve(t.Context(), opts)
		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved User schema
		userResult := userProperty.GetResolvedSchema()
		require.NotNil(t, userResult)
		assert.True(t, userResult.IsSchema())

		// Verify the User schema has an address property that references Address
		userSchema := userResult.GetSchema()
		userProperties := userSchema.GetProperties()
		require.NotNil(t, userProperties)

		addressProperty, exists := userProperties.Get("address")
		require.True(t, exists)

		// The address property should be a reference to #/$defs/Address
		assert.True(t, addressProperty.IsReference())
		assert.Equal(t, "#/$defs/Address", string(addressProperty.GetRef()))
	})

	t.Run("resolve chained reference (ref to ref)", func(t *testing.T) {
		t.Parallel()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/defs_schema.json")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/defs_schema.json",
			RootDocument:   root,
		}

		// Navigate to the chainedRef property which references ChainedRef -> ChainedTarget
		rootSchema := root.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		properties := rootSchema.GetSchema().GetProperties()
		require.NotNil(t, properties)

		chainedProperty, exists := properties.Get("chainedRef")
		require.True(t, exists)
		require.True(t, chainedProperty.IsReference())

		// Resolve the chained reference
		validationErrs, err := chainedProperty.Resolve(t.Context(), opts)
		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema - should be the final ChainedTarget
		result := chainedProperty.GetResolvedSchema()
		require.NotNil(t, result)
		assert.True(t, result.IsSchema())

		// Verify it's the ChainedTarget schema
		resolvedSchema := result.GetSchema()
		assert.Equal(t, []SchemaType{SchemaTypeObject}, resolvedSchema.GetType())

		// Verify it has the expected properties
		resolvedProperties := resolvedSchema.GetProperties()
		require.NotNil(t, resolvedProperties)

		valueProperty, exists := resolvedProperties.Get("value")
		require.True(t, exists)
		assert.True(t, valueProperty.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeString}, valueProperty.GetSchema().GetType())

		descProperty, exists := resolvedProperties.Get("description")
		require.True(t, exists)
		assert.True(t, descProperty.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeString}, descProperty.GetSchema().GetType())
	})

	t.Run("resolve reference from within nested schema with local $defs", func(t *testing.T) {
		t.Parallel()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/defs_schema.json")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/defs_schema.json",
			RootDocument:   root,
		}

		// Navigate to the NestedSchema which has its own $defs
		rootSchema := root.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		nestedSchema, ok := rootSchema.GetSchema().GetDefs().Get("NestedSchema")
		require.True(t, ok)

		nestedSchemaResolved := nestedSchema.MustGetResolvedSchema()
		require.True(t, nestedSchemaResolved.IsSchema())

		// Get the localRef property which should reference the local $defs/LocalDef
		properties := nestedSchemaResolved.GetSchema().GetProperties()
		require.NotNil(t, properties)

		localRef, exists := properties.Get("localRef")
		require.True(t, exists)
		require.True(t, localRef.IsReference())
		assert.Equal(t, "#/$defs/LocalDef", string(localRef.GetRef()))

		// Now resolve the localRef - this should find LocalDef in the nested schema's $defs
		validationErrs, err := localRef.Resolve(t.Context(), opts)
		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved localRef schema
		localRefResolved := localRef.GetResolvedSchema()
		require.NotNil(t, localRefResolved)
		require.True(t, localRefResolved.IsSchema())

		// Verify it's the LocalDef schema
		resolvedSchema := localRefResolved.GetSchema()
		assert.Equal(t, []SchemaType{SchemaTypeObject}, resolvedSchema.GetType())

		// Verify it has the expected properties
		resolvedProperties := resolvedSchema.GetProperties()
		require.NotNil(t, resolvedProperties)

		localValueProperty, exists := resolvedProperties.Get("localValue")
		require.True(t, exists)
		assert.True(t, localValueProperty.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeString}, localValueProperty.GetSchema().GetType())
	})

	t.Run("$defs getter method works correctly", func(t *testing.T) {
		t.Parallel()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/defs_schema.json")
		require.NoError(t, err)

		// Get the $defs from the root schema
		require.True(t, root.IsSchema())
		schema := root.GetSchema()
		defs := schema.GetDefs()
		require.NotNil(t, defs)

		// Verify we have the expected definitions
		userDef, exists := defs.Get("User")
		require.True(t, exists)
		assert.True(t, userDef.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeObject}, userDef.GetSchema().GetType())

		addressDef, exists := defs.Get("Address")
		require.True(t, exists)
		assert.True(t, addressDef.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeObject}, addressDef.GetSchema().GetType())

		nestedDef, exists := defs.Get("NestedSchema")
		require.True(t, exists)
		assert.True(t, nestedDef.IsSchema())
		assert.Equal(t, []SchemaType{SchemaTypeObject}, nestedDef.GetSchema().GetType())
	})
}

func TestSchema_Defs_Error(t *testing.T) {
	t.Parallel()

	t.Run("reference to non-existent $defs entry", func(t *testing.T) {
		t.Parallel()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/defs_schema.json")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/defs_schema.json",
			RootDocument:   root,
		}

		// Navigate to the nonExistentRef property which references a non-existent definition
		rootSchema := root.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		properties := rootSchema.GetSchema().GetProperties()
		require.NotNil(t, properties)

		nonExistentProperty, exists := properties.Get("nonExistentRef")
		require.True(t, exists)
		require.True(t, nonExistentProperty.IsReference())

		// Try to resolve the non-existent reference
		validationErrs, err := nonExistentProperty.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Contains(t, err.Error(), "definition not found")
	})
}

func TestSchema_Defs_Equality(t *testing.T) {
	t.Parallel()

	t.Run("schemas with same $defs are equal", func(t *testing.T) {
		t.Parallel()

		// Create two identical schemas with $defs
		schema1 := &Schema{
			Type: NewTypeFromString(SchemaTypeObject),
		}
		schema1.Defs = createTestDefs()

		schema2 := &Schema{
			Type: NewTypeFromString(SchemaTypeObject),
		}
		schema2.Defs = createTestDefs()

		assert.True(t, schema1.IsEqual(schema2))
	})

	t.Run("schemas with different $defs are not equal", func(t *testing.T) {
		t.Parallel()

		// Create two schemas with different $defs
		schema1 := &Schema{
			Type: NewTypeFromString(SchemaTypeObject),
		}
		schema1.Defs = createTestDefs()

		schema2 := &Schema{
			Type: NewTypeFromString(SchemaTypeObject),
		}
		// schema2 has no $defs

		assert.False(t, schema1.IsEqual(schema2))
	})
}

// Helper function to create test $defs
func createTestDefs() *sequencedmap.Map[string, *JSONSchema[Referenceable]] {
	defs := sequencedmap.New[string, *JSONSchema[Referenceable]]()

	userSchema := &Schema{
		Type: NewTypeFromString(SchemaTypeObject),
	}
	defs.Set("User", NewJSONSchemaFromSchema[Referenceable](userSchema))

	return defs
}

func TestSchema_ExternalDefs_Success(t *testing.T) {
	t.Parallel()

	t.Run("resolve reference to external $defs", func(t *testing.T) {
		t.Parallel()

		// Create a test schema that references external $defs
		testSchemaContent := `{
			"type": "object",
			"properties": {
				"externalUser": {
					"$ref": "external_defs.json#/$defs/ExternalUser"
				}
			}
		}`

		// Parse the test schema
		testSchema := &JSONSchema[Referenceable]{}
		validationErrs, err := marshaller.Unmarshal(t.Context(), strings.NewReader(testSchemaContent), testSchema)
		require.NoError(t, err, "should parse test schema")
		require.Empty(t, validationErrs, "should have no validation errors")

		opts := ResolveOptions{
			TargetLocation: "testdata/test_schema.json", // Base location for resolution
			RootDocument:   testSchema,
		}

		// Navigate to the externalUser property which references external $defs
		rootSchema := testSchema.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		properties := rootSchema.GetSchema().GetProperties()
		require.NotNil(t, properties)

		externalUserProperty, exists := properties.Get("externalUser")
		require.True(t, exists)
		require.True(t, externalUserProperty.IsReference())

		// Resolve the external reference
		validationErrs, err = externalUserProperty.Resolve(t.Context(), opts)
		require.NoError(t, err, "should resolve external $defs reference")
		assert.Nil(t, validationErrs, "should have no validation errors")

		// Get the resolved schema
		result := externalUserProperty.GetResolvedSchema()
		require.NotNil(t, result, "resolved schema should not be nil")
		assert.True(t, result.IsSchema(), "resolved schema should be a schema, not a reference")

		// Verify the resolved schema has the expected structure
		resolvedSchema := result.GetSchema()
		assert.Equal(t, []SchemaType{SchemaTypeObject}, resolvedSchema.GetType(), "resolved schema should be object type")

		resolvedProperties := resolvedSchema.GetProperties()
		require.NotNil(t, resolvedProperties, "resolved schema should have properties")

		// Verify ExternalUser properties
		idProperty, exists := resolvedProperties.Get("id")
		require.True(t, exists, "resolved schema should have id property")
		assert.True(t, idProperty.IsSchema(), "id property should be a schema")
		assert.Equal(t, []SchemaType{SchemaTypeInteger}, idProperty.GetSchema().GetType(), "id should be integer type")

		nameProperty, exists := resolvedProperties.Get("name")
		require.True(t, exists, "resolved schema should have name property")
		assert.True(t, nameProperty.IsSchema(), "name property should be a schema")
		assert.Equal(t, []SchemaType{SchemaTypeString}, nameProperty.GetSchema().GetType(), "name should be string type")
	})

	t.Run("resolve reference to non-existent external $defs", func(t *testing.T) {
		t.Parallel()

		// Create a test schema that references non-existent external $defs
		testSchemaContent := `{
			"type": "object",
			"properties": {
				"nonExistentUser": {
					"$ref": "external_defs.json#/$defs/NonExistent"
				}
			}
		}`

		// Parse the test schema
		testSchema := &JSONSchema[Referenceable]{}
		validationErrs, err := marshaller.Unmarshal(t.Context(), strings.NewReader(testSchemaContent), testSchema)
		require.NoError(t, err, "should parse test schema")
		require.Empty(t, validationErrs, "should have no validation errors")

		opts := ResolveOptions{
			TargetLocation: "testdata/test_schema.json", // Base location for resolution
			RootDocument:   testSchema,
		}

		// Navigate to the nonExistentUser property which references non-existent external $defs
		rootSchema := testSchema.MustGetResolvedSchema()
		require.True(t, rootSchema.IsSchema())

		properties := rootSchema.GetSchema().GetProperties()
		require.NotNil(t, properties)

		nonExistentProperty, exists := properties.Get("nonExistentUser")
		require.True(t, exists)
		require.True(t, nonExistentProperty.IsReference())

		// Try to resolve the non-existent external reference
		validationErrs, err = nonExistentProperty.Resolve(t.Context(), opts)
		require.Error(t, err, "should return error for non-existent external reference")
		assert.Nil(t, validationErrs, "validation errors should be nil on resolution error")
	})
}
