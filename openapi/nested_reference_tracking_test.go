package openapi

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNestedSchemaReferenceTracking tests the ability to track which schema first
// referenced a shared nested schema when iterating through paths, operations, and responses.
//
// Given an OpenAPI document where:
// - /get response schema references Schema1, which references SchemaShared
// - /post response schema references Schema2, which also references SchemaShared
//
// When we iterate through the paths and resolve schemas, we should be able to:
//  1. Track that Schema1 first referenced SchemaShared (via /get)
//  2. When we encounter Schema2's reference to SchemaShared (via /post), identify that
//     SchemaShared was already referenced by Schema1
func TestNestedSchemaReferenceTracking(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	yml := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /get:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Schema1"
  /post:
    post:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Schema2"
components:
  schemas:
    Schema1:
      $ref: "#/components/schemas/SchemaShared"
    Schema2:
      $ref: "#/components/schemas/SchemaShared"
    SchemaShared:
      type: object
      properties:
        name:
          type: string
        id:
          type: integer
`

	// Parse the document using openapi.Unmarshal
	doc, validationErrs, err := Unmarshal(ctx, bytes.NewBufferString(yml))
	require.NoError(t, err, "should parse OpenAPI document")
	assert.Empty(t, validationErrs, "should have no validation errors")

	// Setup resolve options
	resolveOpts := ResolveOptions{
		TargetLocation: "test.yaml",
		RootDocument:   doc,
	}

	// Track nested reference discovery order using the clean GetReferenceChain() API
	// Maps: final resolved ref (e.g., "#/components/schemas/SchemaShared") -> info about first discovery
	type NestedRefInfo struct {
		FirstReferencerRef  string // e.g., "#/components/schemas/Schema1" (the intermediate ref)
		DiscoveredViaPath   string // e.g., "/get"
		DiscoveredViaMethod string // e.g., "get"
	}
	nestedRefTracker := make(map[string]*NestedRefInfo)

	// Iterate through paths -> operations -> responses -> content -> schema
	require.NotNil(t, doc.Paths, "should have paths")

	for path, pathItem := range doc.Paths.All() {
		pathItemObj := pathItem.GetObject()
		if pathItemObj == nil {
			continue
		}

		for method, operation := range pathItemObj.All() {
			if operation == nil || operation.Responses.Len() == 0 {
				continue
			}

			methodStr := string(method)
			for statusCode, response := range operation.Responses.All() {
				responseObj := response.GetObject()
				if responseObj == nil || responseObj.Content == nil {
					continue
				}

				for contentType, mediaType := range responseObj.Content.All() {
					if mediaType.Schema == nil {
						continue
					}

					schema := mediaType.Schema

					t.Logf("Found schema at %s %s response %s content %s", methodStr, path, statusCode, contentType)

					if schema.IsReference() {
						schemaRef := string(schema.GetRef())
						t.Logf("  Schema is a reference: %s", schemaRef)

						// Resolve the schema reference
						_, err := schema.Resolve(ctx, oas3.ResolveOptions{
							RootDocument:   doc,
							TargetDocument: doc,
							TargetLocation: resolveOpts.TargetLocation,
						})
						require.NoError(t, err, "should resolve schema reference")

						resolvedSchema := schema.GetResolvedSchema()
						require.NotNil(t, resolvedSchema, "should have resolved schema")

						// Use the clean GetReferenceChain() API to track nested references
						chain := resolvedSchema.GetReferenceChain()
						t.Logf("  Reference chain length: %d", len(chain))

						for i, entry := range chain {
							t.Logf("    Chain[%d]: %s", i, entry.Reference)
						}

						// A chain length > 1 indicates nested references
						// chain[0] = top-level reference (e.g., response schema -> Schema1)
						// chain[1] = nested reference (e.g., Schema1 -> SchemaShared)
						if len(chain) > 1 {
							// Get the final/deepest reference in the chain (the nested one)
							immediateRef := chain[len(chain)-1]
							nestedRefStr := string(immediateRef.Reference)

							// The intermediate referencer is the entry before the final one
							intermediateRef := chain[len(chain)-2]
							intermediateRefStr := string(intermediateRef.Reference)

							if existingInfo, found := nestedRefTracker[nestedRefStr]; found {
								t.Logf("  ALREADY REFERENCED: '%s' was first referenced by '%s' (discovered via %s %s)",
									nestedRefStr, existingInfo.FirstReferencerRef,
									existingInfo.DiscoveredViaMethod, existingInfo.DiscoveredViaPath)
							} else {
								nestedRefTracker[nestedRefStr] = &NestedRefInfo{
									FirstReferencerRef:  intermediateRefStr,
									DiscoveredViaPath:   path,
									DiscoveredViaMethod: methodStr,
								}
								t.Logf("  FIRST NESTED REFERENCE: '%s' first referenced via '%s' (discovered via %s %s)",
									nestedRefStr, intermediateRefStr, methodStr, path)
							}
						}

						// Also demonstrate the convenience methods
						immediateRefEntry := resolvedSchema.GetImmediateReference()
						topLevelRefEntry := resolvedSchema.GetTopLevelReference()

						if immediateRefEntry != nil {
							t.Logf("  Immediate reference: %s", immediateRefEntry.Reference)
						}
						if topLevelRefEntry != nil {
							t.Logf("  Top-level reference: %s", topLevelRefEntry.Reference)
						}
					}
				}
			}
		}
	}

	// Verify our tracking captured the expected results
	assert.Len(t, nestedRefTracker, 1, "should have tracked one nested reference (SchemaShared)")

	sharedInfo, found := nestedRefTracker["#/components/schemas/SchemaShared"]
	require.True(t, found, "should have tracked SchemaShared")
	assert.Equal(t, "#/components/schemas/Schema1", sharedInfo.FirstReferencerRef, "Schema1 should be the first referencer")
	assert.Equal(t, "/get", sharedInfo.DiscoveredViaPath, "should be discovered via /get path")
	assert.Equal(t, "get", sharedInfo.DiscoveredViaMethod, "should be discovered via GET method")

	t.Log("\n=== Summary ===")
	t.Logf("Successfully tracked that SchemaShared was first referenced via %s (via %s %s)",
		sharedInfo.FirstReferencerRef, sharedInfo.DiscoveredViaMethod, sharedInfo.DiscoveredViaPath)
	t.Log("When processing /post POST, we detected that SchemaShared was already tracked.")
}

// TestNestedReferenceChain_ThreeLevels tests reference chain tracking with three levels of nesting.
func TestNestedReferenceChain_ThreeLevels(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	yml := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Level1"
components:
  schemas:
    Level1:
      $ref: "#/components/schemas/Level2"
    Level2:
      $ref: "#/components/schemas/Level3"
    Level3:
      type: object
      properties:
        data:
          type: string
`

	doc, validationErrs, err := Unmarshal(ctx, bytes.NewBufferString(yml))
	require.NoError(t, err, "should parse OpenAPI document")
	assert.Empty(t, validationErrs, "should have no validation errors")

	// Get the response schema
	pathItem, _ := doc.Paths.Get("/users")
	require.NotNil(t, pathItem.GetObject(), "should have path item")

	getOp := pathItem.GetObject().Get()
	require.NotNil(t, getOp, "should have GET operation")

	response, _ := getOp.Responses.Get("200")
	require.NotNil(t, response.GetObject(), "should have response")

	mediaType, _ := response.GetObject().Content.Get("application/json")
	require.NotNil(t, mediaType.Schema, "should have schema")

	schema := mediaType.Schema
	require.True(t, schema.IsReference(), "schema should be a reference")

	// Resolve the schema
	_, err = schema.Resolve(ctx, oas3.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	require.NoError(t, err, "should resolve schema")

	resolvedSchema := schema.GetResolvedSchema()
	require.NotNil(t, resolvedSchema, "should have resolved schema")

	// Get the reference chain - should have 3 entries for 3 levels of references
	chain := resolvedSchema.GetReferenceChain()
	require.Len(t, chain, 3, "should have 3 entries in reference chain")

	// Verify chain order (top-level first, immediate parent last)
	assert.Equal(t, "#/components/schemas/Level1", string(chain[0].Reference), "first entry should be Level1")
	assert.Equal(t, "#/components/schemas/Level2", string(chain[1].Reference), "second entry should be Level2")
	assert.Equal(t, "#/components/schemas/Level3", string(chain[2].Reference), "third entry should be Level3")

	// Verify convenience methods
	immediateRef := resolvedSchema.GetImmediateReference()
	require.NotNil(t, immediateRef, "should have immediate reference")
	assert.Equal(t, "#/components/schemas/Level3", string(immediateRef.Reference), "immediate reference should be Level3")

	topLevelRef := resolvedSchema.GetTopLevelReference()
	require.NotNil(t, topLevelRef, "should have top-level reference")
	assert.Equal(t, "#/components/schemas/Level1", string(topLevelRef.Reference), "top-level reference should be Level1")
}

// TestGetReferenceChain_NoNesting tests that non-nested references return a single-entry chain.
func TestGetReferenceChain_NoNesting(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	yml := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
`

	doc, validationErrs, err := Unmarshal(ctx, bytes.NewBufferString(yml))
	require.NoError(t, err, "should parse OpenAPI document")
	assert.Empty(t, validationErrs, "should have no validation errors")

	pathItem, _ := doc.Paths.Get("/users")
	getOp := pathItem.GetObject().Get()
	response, _ := getOp.Responses.Get("200")
	mediaType, _ := response.GetObject().Content.Get("application/json")
	schema := mediaType.Schema

	// Resolve the schema
	_, err = schema.Resolve(ctx, oas3.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})
	require.NoError(t, err, "should resolve schema")

	resolvedSchema := schema.GetResolvedSchema()
	require.NotNil(t, resolvedSchema, "should have resolved schema")

	// Get the reference chain - should have 1 entry for single-level reference
	chain := resolvedSchema.GetReferenceChain()
	require.Len(t, chain, 1, "should have 1 entry in reference chain for non-nested reference")

	assert.Equal(t, "#/components/schemas/User", string(chain[0].Reference), "entry should be User reference")

	// Immediate and top-level should be the same for non-nested references
	immediateRef := resolvedSchema.GetImmediateReference()
	topLevelRef := resolvedSchema.GetTopLevelReference()
	require.NotNil(t, immediateRef, "should have immediate reference")
	require.NotNil(t, topLevelRef, "should have top-level reference")
	assert.Equal(t, immediateRef.Reference, topLevelRef.Reference, "immediate and top-level should be same for non-nested")
}

// TestGetReferenceChain_InlineSchema tests that inline schemas have no reference chain.
func TestGetReferenceChain_InlineSchema(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	yml := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  name:
                    type: string
`

	doc, validationErrs, err := Unmarshal(ctx, bytes.NewBufferString(yml))
	require.NoError(t, err, "should parse OpenAPI document")
	assert.Empty(t, validationErrs, "should have no validation errors")

	pathItem, _ := doc.Paths.Get("/users")
	getOp := pathItem.GetObject().Get()
	response, _ := getOp.Responses.Get("200")
	mediaType, _ := response.GetObject().Content.Get("application/json")
	schema := mediaType.Schema

	// Inline schema - not a reference
	require.False(t, schema.IsReference(), "inline schema should not be a reference")

	// GetResolvedSchema on non-reference returns the schema itself
	resolvedSchema := schema.GetResolvedSchema()
	require.NotNil(t, resolvedSchema, "should have resolved schema")

	// Inline schema should have no reference chain
	chain := resolvedSchema.GetReferenceChain()
	assert.Nil(t, chain, "inline schema should have no reference chain")

	immediateRef := resolvedSchema.GetImmediateReference()
	assert.Nil(t, immediateRef, "inline schema should have no immediate reference")

	topLevelRef := resolvedSchema.GetTopLevelReference()
	assert.Nil(t, topLevelRef, "inline schema should have no top-level reference")
}
