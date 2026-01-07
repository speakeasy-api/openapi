package oas3

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTryResolveLocalDefs_NilAndBooleanSchema(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		var s *JSONSchemaReferenceable
		ref := references.Reference("#/$defs/Foo")
		opts := references.ResolveOptions{}

		result := s.tryResolveLocalDefs(ctx, ref, opts)

		assert.Nil(t, result)
	})

	t.Run("boolean schema returns nil", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		s := NewJSONSchemaFromBool(true)
		ref := references.Reference("#/$defs/Foo")
		opts := references.ResolveOptions{}

		result := s.tryResolveLocalDefs(ctx, ref, opts)

		assert.Nil(t, result)
	})
}

func TestTryResolveLocalDefs_NoID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		schemaYAML string
	}{
		{
			name: "schema without $id returns nil",
			schemaYAML: `
type: object
$defs:
  Foo:
    type: string
`,
		},
		{
			name: "schema without $defs returns nil",
			schemaYAML: `
type: object
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			schema, err := parseDefTestSchema(t, tt.schemaYAML)
			require.NoError(t, err)

			ref := references.Reference("#/$defs/Foo")
			opts := references.ResolveOptions{}

			result := schema.tryResolveLocalDefs(ctx, ref, opts)

			assert.Nil(t, result)
		})
	}
}

func TestTryResolveLocalDefs_WithID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		schemaYAML     string
		refPath        string
		expectedResult bool
		expectedType   SchemaType
	}{
		{
			name: "resolves $defs entry with $id",
			schemaYAML: `
$id: https://example.com/schema.json
type: object
$defs:
  User:
    type: object
`,
			refPath:        "#/$defs/User",
			expectedResult: true,
			expectedType:   SchemaTypeObject,
		},
		{
			name: "returns nil for non-existent def",
			schemaYAML: `
$id: https://example.com/schema.json
type: object
$defs:
  User:
    type: object
`,
			refPath:        "#/$defs/NonExistent",
			expectedResult: false,
		},
		{
			name: "returns nil for def with remaining path",
			schemaYAML: `
$id: https://example.com/schema.json
type: object
$defs:
  User:
    type: object
    properties:
      name:
        type: string
`,
			refPath:        "#/$defs/User/properties/name",
			expectedResult: false, // Not supported in tryResolveLocalDefs
		},
		{
			name: "returns nil for non-$defs prefix",
			schemaYAML: `
$id: https://example.com/schema.json
type: object
properties:
  foo:
    type: string
`,
			refPath:        "#/properties/foo",
			expectedResult: false,
		},
		{
			name: "handles URL-encoded def key with tilde",
			schemaYAML: `
$id: https://example.com/schema.json
type: object
$defs:
  "complex~name":
    type: integer
`,
			refPath:        "#/$defs/complex~0name",
			expectedResult: true,
			expectedType:   SchemaTypeInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			schema, err := parseDefTestSchema(t, tt.schemaYAML)
			require.NoError(t, err)

			ref := references.Reference(tt.refPath)
			opts := references.ResolveOptions{}

			result := schema.tryResolveLocalDefs(ctx, ref, opts)

			if tt.expectedResult {
				require.NotNil(t, result)
				assert.NotNil(t, result.Object)
				if result.Object.IsSchema() {
					types := result.Object.GetSchema().GetType()
					if len(types) > 0 {
						assert.Equal(t, tt.expectedType, types[0])
					}
				}
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestResolveDefsReference_NotDefsReference(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	schema, err := parseDefTestSchema(t, `
type: object
properties:
  name:
    type: string
`)
	require.NoError(t, err)

	// Create a reference to /properties/name (not a $defs reference)
	ref := references.Reference("#/properties/name")
	opts := references.ResolveOptions{}

	_, _, err = schema.resolveDefsReference(ctx, ref, opts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a $defs reference")
}

func TestTryResolveLocalDefs_WithEffectiveBaseURI(t *testing.T) {
	t.Parallel()

	t.Run("resolves with effective base URI different from document base", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		schemaYAML := `
type: object
$defs:
  Item:
    type: string
`
		schema, err := parseDefTestSchema(t, schemaYAML)
		require.NoError(t, err)

		// Set up a registry with a document base URI
		registry := NewSchemaRegistry("https://example.com/root.json")
		schema.SetSchemaRegistry(registry)

		// Set a different effective base URI on the inner schema
		innerSchema := schema.GetSchema()
		require.NotNil(t, innerSchema)
		innerSchema.SetEffectiveBaseURI("https://example.com/different.json")

		ref := references.Reference("#/$defs/Item")
		opts := references.ResolveOptions{}

		result := schema.tryResolveLocalDefs(ctx, ref, opts)

		require.NotNil(t, result)
		assert.NotNil(t, result.Object)
	})
}

// parseDefTestSchema is a helper to parse a YAML schema string into a JSONSchema
func parseDefTestSchema(t *testing.T, yamlContent string) (*JSONSchemaReferenceable, error) {
	t.Helper()

	if yamlContent == "" {
		return nil, nil
	}

	schema := &JSONSchemaReferenceable{}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &node); err != nil {
		return nil, err
	}

	ctx := t.Context()
	_, err := marshaller.UnmarshalNode(ctx, "", &node, schema)
	if err != nil {
		return nil, err
	}

	return schema, nil
}
