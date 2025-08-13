package oas3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test resolution of external document with internal references
func TestJSONSchema_Resolve_ExternalWithInternalRefs(t *testing.T) {
	t.Parallel()

	t.Run("external document with internal $defs references", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem
		fs := NewMockVirtualFS()

		// Add external document with internal references
		fs.AddFile("testdata/external_with_refs.json", `{
			"$defs": {
				"Person": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"address": {
							"$ref": "#/$defs/Address"
						}
					}
				},
				"Address": {
					"type": "object",
					"properties": {
						"street": {
							"type": "string"
						},
						"city": {
							"type": "string"
						}
					}
				}
			}
		}`)

		// Create root document with reference to external document
		root := NewMockResolutionTarget()
		ref := "external_with_refs.json#/$defs/Person"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		// First resolution - get the Person schema
		validationErrs, err := schema.Resolve(t.Context(), opts)
		require.NoError(t, err, "first resolution should succeed")
		assert.Nil(t, validationErrs)

		// Get the resolved schema
		result := schema.GetResolvedSchema()
		require.NotNil(t, result, "resolved schema should not be nil")
		assert.True(t, result.IsLeft(), "should be a schema object")

		personSchema := result.GetLeft()
		require.NotNil(t, personSchema, "person schema should not be nil")

		// Check that Person has properties
		props := personSchema.Properties
		require.NotNil(t, props, "person should have properties")

		// Get the address property which has an internal reference
		addressProp, exists := props.Get("address")
		require.True(t, exists, "address property should exist")
		require.NotNil(t, addressProp, "address property should not be nil")

		// Check if it's a reference
		if addressProp.IsReference() {
			t.Logf("Address property is a reference: %s", addressProp.GetRef())

			// Try to resolve the address reference
			// This reference (#/$defs/Address) should resolve within the external document
			addressValidationErrs, addressErr := addressProp.Resolve(t.Context(), opts)

			// Log any error for debugging
			if addressErr != nil {
				t.Logf("Failed to resolve address reference: %v", addressErr)
			}

			require.NoError(t, addressErr, "address reference should resolve successfully")
			assert.Nil(t, addressValidationErrs)

			// Get the resolved address schema
			addressResolved := addressProp.GetResolvedSchema()
			require.NotNil(t, addressResolved, "resolved address schema should not be nil")
			assert.True(t, addressResolved.IsLeft(), "address should be a schema object")

			addressSchema := addressResolved.GetLeft()
			require.NotNil(t, addressSchema, "address schema should not be nil")

			// Verify address has the expected properties
			addressProps := addressSchema.Properties
			require.NotNil(t, addressProps, "address should have properties")

			_, hasStreet := addressProps.Get("street")
			assert.True(t, hasStreet, "address should have street property")

			_, hasCity := addressProps.Get("city")
			assert.True(t, hasCity, "address should have city property")
		} else {
			t.Log("Address property is not a reference - it may have been inlined during resolution")
			// The reference may have been automatically resolved
			// Check if it's directly an object
			assert.True(t, addressProp.IsLeft(), "address should be a schema object")
			addressSchema := addressProp.GetLeft()
			require.NotNil(t, addressSchema, "address schema should not be nil")
		}
	})

	t.Run("external document with circular internal references", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem
		fs := NewMockVirtualFS()

		// Add external document with circular internal references
		fs.AddFile("testdata/external_circular.json", `{
			"$defs": {
				"TreeNode": {
					"type": "object",
					"properties": {
						"value": {
							"type": "string"
						},
						"children": {
							"type": "array",
							"items": {
								"$ref": "#/$defs/TreeNode"
							}
						}
					}
				}
			}
		}`)

		// Create root document with reference to external document
		root := NewMockResolutionTarget()
		ref := "external_circular.json#/$defs/TreeNode"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		// First resolution - get the TreeNode schema
		validationErrs, err := schema.Resolve(t.Context(), opts)
		require.NoError(t, err, "first resolution should succeed")
		assert.Nil(t, validationErrs)

		// Get the resolved schema
		result := schema.GetResolvedSchema()
		require.NotNil(t, result, "resolved schema should not be nil")
		assert.True(t, result.IsLeft(), "should be a schema object")

		treeNodeSchema := result.GetLeft()
		require.NotNil(t, treeNodeSchema, "tree node schema should not be nil")

		// Check that TreeNode has properties
		props := treeNodeSchema.Properties
		require.NotNil(t, props, "tree node should have properties")

		// Get the children property
		childrenProp, exists := props.Get("children")
		require.True(t, exists, "children property should exist")
		require.NotNil(t, childrenProp, "children property should not be nil")

		// Check the items property of the array
		assert.True(t, childrenProp.IsLeft(), "children should be a schema object")
		childrenSchema := childrenProp.GetLeft()
		require.NotNil(t, childrenSchema, "children schema should not be nil")
		// Check that it's an array type
		schemaTypes := childrenSchema.GetType()
		if len(schemaTypes) > 0 {
			assert.Equal(t, SchemaTypeArray, schemaTypes[0], "children should be an array")
		}

		// Get the items schema
		items := childrenSchema.Items
		require.NotNil(t, items, "children should have items")

		// Check if items is a reference
		if items.IsReference() {
			t.Logf("Items is a reference: %s", items.GetRef())

			// Try to resolve the items reference
			// This should work even though it's circular
			itemsOpts := opts
			// The items reference should resolve against the external document
			// Since this was loaded from external_circular.json, that should be the context

			itemsValidationErrs, itemsErr := items.Resolve(t.Context(), itemsOpts)

			// For circular references, we expect this to either:
			// 1. Succeed if the resolution handles circularity
			// 2. Fail with a circular reference error
			if itemsErr != nil {
				t.Logf("Items resolution error (expected for circular): %v", itemsErr)
				// Check if it's a circular reference error
				assert.Contains(t, itemsErr.Error(), "circular", "should be a circular reference error")
			} else {
				// If it succeeds, the resolved schema should be valid
				assert.Nil(t, itemsValidationErrs)
				itemsResolved := items.GetResolvedSchema()
				assert.NotNil(t, itemsResolved, "resolved items schema should not be nil if resolution succeeded")
			}
		} else {
			t.Log("Items is not a reference - it may have been resolved during initial resolution")
		}
	})
}
