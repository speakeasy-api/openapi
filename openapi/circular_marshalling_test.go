package openapi_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

// TestCircularReferenceMarshalling tests if the marshaller can handle circular references
// without infinite recursion. This isolates the issue from any inlining code.
func TestCircularReferenceMarshalling(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// OpenAPI document with circular references
	openAPIDoc := `{
		"openapi": "3.1.1",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"manager": {
							"$ref": "#/components/schemas/Manager"
						}
					}
				},
				"Manager": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"reports": {
							"type": "array",
							"items": {
								"$ref": "#/components/schemas/User"
							}
						}
					}
				}
			}
		}
	}`

	t.Log("1. Parsing OpenAPI document...")
	reader := strings.NewReader(openAPIDoc)
	doc, _, err := openapi.Unmarshal(ctx, reader)
	require.NoError(t, err, "failed to parse OpenAPI document")

	t.Log("2. Extracting User schema...")
	target, err := jsonpointer.GetTarget(doc, jsonpointer.JSONPointer("/components/schemas/User"))
	require.NoError(t, err, "failed to extract schema")

	schema, ok := target.(*oas3.JSONSchema[oas3.Referenceable])
	require.True(t, ok, "target is not a JSONSchema: %T", target)

	t.Log("3. Resolving references...")
	resolveOpts := oas3.ResolveOptions{
		TargetLocation: "openapi.json",
		RootDocument:   doc,
	}
	_, err = schema.Resolve(ctx, resolveOpts)
	require.NoError(t, err, "failed to resolve references")

	t.Log("4. Marshalling schema back to JSON...")
	// This is where the infinite recursion happens if it's a marshaller bug
	var buffer strings.Builder
	err = openapi.Marshal(ctx, doc, &buffer)
	require.NoError(t, err, "failed to marshal schema - this indicates a marshaller bug with circular references")

	actualJSON := buffer.String()
	t.Logf("✓ Marshalled successfully, result length: %d characters", len(actualJSON))

	// Basic sanity check that we got some JSON back
	require.NotEmpty(t, actualJSON, "marshalled JSON should not be empty")
	require.Contains(t, actualJSON, "User", "marshalled JSON should contain schema content")
}

// TestCircularReferenceFullDocumentMarshalling tests marshalling the entire OpenAPI document
// after resolving references to see if the issue is specific to individual schemas
func TestCircularReferenceFullDocumentMarshalling(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// OpenAPI document with circular references
	openAPIDoc := `{
		"openapi": "3.1.1",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"manager": {
							"$ref": "#/components/schemas/Manager"
						}
					}
				},
				"Manager": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"reports": {
							"type": "array",
							"items": {
								"$ref": "#/components/schemas/User"
							}
						}
					}
				}
			}
		}
	}`

	t.Log("1. Parsing OpenAPI document...")
	reader := strings.NewReader(openAPIDoc)
	doc, _, err := openapi.Unmarshal(ctx, reader)
	require.NoError(t, err, "failed to parse OpenAPI document")
	require.NotNil(t, doc, "OpenAPI document should not be nil")

	t.Log("2. Resolving all references in the document...")
	// Resolve references in all schemas
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.All() {
			t.Logf("Resolving schema: %s", name)
			resolveOpts := oas3.ResolveOptions{
				TargetLocation: "openapi.json",
				RootDocument:   doc,
			}
			_, err = schema.Resolve(ctx, resolveOpts)
			require.NoError(t, err, "failed to resolve references for schema %s", name)
		}
	}

	t.Log("3. Marshalling entire document back to JSON...")
	// This tests if the entire document can be marshalled after resolving circular references
	var buffer strings.Builder
	err = openapi.Marshal(ctx, doc, &buffer)
	require.NoError(t, err, "failed to marshal full document - this indicates a marshaller bug with circular references")

	actualJSON := buffer.String()
	t.Logf("✓ Marshalled successfully, result length: %d characters", len(actualJSON))

	// Basic sanity check that we got some JSON back
	require.NotEmpty(t, actualJSON, "marshalled JSON should not be empty")
	require.Contains(t, actualJSON, "User", "marshalled JSON should contain schema content")
	require.Contains(t, actualJSON, "Manager", "marshalled JSON should contain schema content")
}
