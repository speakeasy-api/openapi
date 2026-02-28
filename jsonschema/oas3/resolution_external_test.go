package oas3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestResolveExternalAnchorReference_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		schemaJSON     string
		ref            string
		expectedAnchor string
		expectedType   SchemaType
	}{
		{
			name: "anchor in external document root $defs",
			schemaJSON: `{
				"$defs": {
					"foo": {
						"$anchor": "myAnchor",
						"type": "string"
					}
				}
			}`,
			ref:            "schema.json#myAnchor",
			expectedAnchor: "myAnchor",
			expectedType:   SchemaTypeString,
		},
		{
			name: "anchor with $id in document",
			schemaJSON: `{
				"$id": "https://example.com/schemas/root.json",
				"$defs": {
					"bar": {
						"$anchor": "barAnchor",
						"type": "integer"
					}
				}
			}`,
			ref:            "schema.json#barAnchor",
			expectedAnchor: "barAnchor",
			expectedType:   SchemaTypeInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			// Create HTTP server to serve the schema
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.schemaJSON))
			}))
			defer server.Close()

			root := NewMockResolutionTarget()
			schema := createSchemaWithRef(tt.ref)

			opts := ResolveOptions{
				TargetLocation: server.URL + "/root.json",
				RootDocument:   root,
			}

			validationErrs, err := schema.Resolve(ctx, opts)

			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			resolved := schema.GetResolvedSchema()
			require.NotNil(t, resolved)
			require.True(t, resolved.IsSchema())

			resolvedSchema := resolved.GetSchema()
			require.NotNil(t, resolvedSchema)

			types := resolvedSchema.GetType()
			require.NotEmpty(t, types)
			assert.Equal(t, tt.expectedType, types[0])
		})
	}
}

func TestResolveExternalAnchorReference_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schemaJSON   string
		ref          string
		errorContain string
	}{
		{
			name: "anchor not found in external document",
			schemaJSON: `{
				"$defs": {
					"foo": {
						"$anchor": "existingAnchor",
						"type": "string"
					}
				}
			}`,
			ref:          "schema.json#nonExistentAnchor",
			errorContain: "anchor not found",
		},
		{
			name:         "empty schema document",
			schemaJSON:   `{}`,
			ref:          "schema.json#someAnchor",
			errorContain: "anchor not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.schemaJSON))
			}))
			defer server.Close()

			root := NewMockResolutionTarget()
			schema := createSchemaWithRef(tt.ref)

			opts := ResolveOptions{
				TargetLocation: server.URL + "/root.json",
				RootDocument:   root,
			}

			_, err := schema.Resolve(ctx, opts)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContain)
		})
	}
}

func TestResolveExternalRefWithFragment_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schemaJSON   string
		ref          string
		expectedType SchemaType
	}{
		{
			name: "resolve $defs via JSON pointer",
			schemaJSON: `{
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"name": { "type": "string" }
						}
					}
				}
			}`,
			ref:          "schema.json#/$defs/User",
			expectedType: SchemaTypeObject,
		},
		{
			name: "resolve root document without fragment",
			schemaJSON: `{
				"type": "array",
				"items": { "type": "string" }
			}`,
			ref:          "schema.json",
			expectedType: SchemaTypeArray,
		},
		{
			name: "resolve properties via JSON pointer",
			schemaJSON: `{
				"type": "object",
				"properties": {
					"id": { "type": "integer" }
				}
			}`,
			ref:          "schema.json#/properties/id",
			expectedType: SchemaTypeInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.schemaJSON))
			}))
			defer server.Close()

			root := NewMockResolutionTarget()
			schema := createSchemaWithRef(tt.ref)

			opts := ResolveOptions{
				TargetLocation: server.URL + "/root.json",
				RootDocument:   root,
			}

			validationErrs, err := schema.Resolve(ctx, opts)

			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			resolved := schema.GetResolvedSchema()
			require.NotNil(t, resolved)
			require.True(t, resolved.IsSchema())

			resolvedSchema := resolved.GetSchema()
			require.NotNil(t, resolvedSchema)

			types := resolvedSchema.GetType()
			require.NotEmpty(t, types)
			assert.Equal(t, tt.expectedType, types[0])
		})
	}
}

func TestResolveExternalRefWithFragment_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schemaJSON   string
		ref          string
		errorContain string
	}{
		{
			name: "invalid JSON pointer path",
			schemaJSON: `{
				"$defs": {
					"User": { "type": "object" }
				}
			}`,
			ref:          "schema.json#/$defs/NonExistent",
			errorContain: "failed to navigate",
		},
		{
			name:         "deeply nested invalid path",
			schemaJSON:   `{"type": "object"}`,
			ref:          "schema.json#/properties/foo/bar/baz",
			errorContain: "failed to navigate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.schemaJSON))
			}))
			defer server.Close()

			root := NewMockResolutionTarget()
			schema := createSchemaWithRef(tt.ref)

			opts := ResolveOptions{
				TargetLocation: server.URL + "/root.json",
				RootDocument:   root,
			}

			_, err := schema.Resolve(ctx, opts)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContain)
		})
	}
}

func TestNavigateJSONPointer_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schemaYAML   string
		jsonPointer  string
		expectedType SchemaType
	}{
		{
			name: "navigate to $defs entry",
			schemaYAML: `
type: object
$defs:
  Name:
    type: string
`,
			jsonPointer:  "/$defs/Name",
			expectedType: SchemaTypeString,
		},
		{
			name: "navigate to properties entry",
			schemaYAML: `
type: object
properties:
  count:
    type: integer
`,
			jsonPointer:  "/properties/count",
			expectedType: SchemaTypeInteger,
		},
		{
			name: "empty pointer returns root",
			schemaYAML: `
type: boolean
`,
			jsonPointer:  "",
			expectedType: SchemaTypeBoolean,
		},
		{
			name: "root pointer returns root",
			schemaYAML: `
type: number
`,
			jsonPointer:  "/",
			expectedType: SchemaTypeNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			schema, err := parseTestSchema(t, tt.schemaYAML)
			require.NoError(t, err)

			result, err := navigateJSONPointer(ctx, schema, jsonpointer.JSONPointer(tt.jsonPointer))

			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.IsSchema())

			types := result.GetSchema().GetType()
			require.NotEmpty(t, types)
			assert.Equal(t, tt.expectedType, types[0])
		})
	}
}

func TestNavigateJSONPointer_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schemaYAML   string
		jsonPointer  string
		errorContain string
	}{
		{
			name:         "nil schema",
			schemaYAML:   "",
			jsonPointer:  "/properties/name",
			errorContain: "cannot navigate within nil",
		},
		{
			name: "invalid path segment",
			schemaYAML: `
type: object
properties:
  name:
    type: string
`,
			jsonPointer:  "/properties/nonexistent",
			errorContain: "failed to navigate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var schema *JSONSchemaReferenceable
			if tt.schemaYAML != "" {
				var err error
				schema, err = parseTestSchema(t, tt.schemaYAML)
				require.NoError(t, err)
			}

			_, err := navigateJSONPointer(ctx, schema, jsonpointer.JSONPointer(tt.jsonPointer))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContain)
		})
	}
}

func TestSetupRemoteSchemaRegistry_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		schemaYAML      string
		documentBaseURI string
		expectedIDs     []string
		expectedAnchors []string
	}{
		{
			name: "registers $id from root",
			schemaYAML: `
$id: https://example.com/schemas/user.json
type: object
`,
			documentBaseURI: "https://example.com/schemas/user.json",
			expectedIDs:     []string{"https://example.com/schemas/user.json"},
			expectedAnchors: nil,
		},
		{
			name: "registers $anchor from nested schema",
			schemaYAML: `
type: object
$defs:
  Name:
    $anchor: nameAnchor
    type: string
`,
			documentBaseURI: "https://example.com/root.json",
			expectedIDs:     nil,
			expectedAnchors: []string{"nameAnchor"},
		},
		{
			name: "registers both $id and $anchor",
			schemaYAML: `
$id: https://example.com/combined.json
type: object
$defs:
  Item:
    $anchor: item
    type: object
`,
			documentBaseURI: "https://example.com/combined.json",
			expectedIDs:     []string{"https://example.com/combined.json"},
			expectedAnchors: []string{"item"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			schema, err := parseTestSchema(t, tt.schemaYAML)
			require.NoError(t, err)

			setupRemoteSchemaRegistry(ctx, schema, tt.documentBaseURI)

			registry := schema.GetSchemaRegistry()
			require.NotNil(t, registry)

			// Verify IDs were registered
			for _, id := range tt.expectedIDs {
				found := registry.LookupByID(id)
				assert.NotNil(t, found, "expected $id %s to be registered", id)
			}

			// Verify anchors were registered
			for _, anchor := range tt.expectedAnchors {
				found := registry.LookupByAnchor(tt.documentBaseURI, anchor)
				assert.NotNil(t, found, "expected $anchor %s to be registered", anchor)
			}
		})
	}
}

func TestSetupRemoteSchemaRegistry_NilAndBooleanSchema(t *testing.T) {
	t.Parallel()

	t.Run("nil schema does not panic", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		assert.NotPanics(t, func() {
			setupRemoteSchemaRegistry(ctx, nil, "https://example.com/test.json")
		})
	})

	t.Run("boolean schema does not panic", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		boolSchema := NewJSONSchemaFromBool(true)
		assert.NotPanics(t, func() {
			setupRemoteSchemaRegistry(ctx, boolSchema, "https://example.com/test.json")
		})
	})
}

// parseTestSchema is a helper to parse a YAML schema string into a JSONSchema
func parseTestSchema(t *testing.T, yamlContent string) (*JSONSchemaReferenceable, error) {
	t.Helper()

	if yamlContent == "" {
		return nil, nil
	}

	schema := &JSONSchemaReferenceable{}
	ctx := context.Background()

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &node); err != nil {
		return nil, err
	}

	_, err := marshaller.UnmarshalNode(ctx, "", &node, schema)
	if err != nil {
		return nil, err
	}

	return schema, nil
}
