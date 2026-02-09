package openapi_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
)

// BenchmarkBuildIndex_ManyRefsToSameSchema creates a synthetic spec with many
// references to the same schema to benchmark for quadratic/exponential re-walking.
// Mimics the Glean spec pattern: heavy schemas referenced many times,
// self-references, deep nesting, and a large union type (StructuredResult).
func BenchmarkBuildIndex_ManyRefsToSameSchema(b *testing.B) {
	numPaths := 200

	var pathsBuilder strings.Builder
	for i := 0; i < numPaths; i++ {
		fmt.Fprintf(&pathsBuilder, `  /items/%d:
    get:
      operationId: getItem%d
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SearchResult'
`, i, i)
	}

	// Build many leaf schemas (like Glean's 480 schemas)
	var leafSchemas strings.Builder
	numLeafSchemas := 20
	for i := 0; i < numLeafSchemas; i++ {
		fmt.Fprintf(&leafSchemas, `    LeafSchema%d:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        document:
          $ref: '#/components/schemas/Document'
        person:
          $ref: '#/components/schemas/Person'
`, i)
	}

	// Build StructuredResult union with many branches
	var unionProps strings.Builder
	for i := 0; i < numLeafSchemas; i++ {
		fmt.Fprintf(&unionProps, `        leaf%d:
          $ref: '#/components/schemas/LeafSchema%d'
`, i, i)
	}

	spec := fmt.Sprintf(`openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths:
%s
components:
  schemas:
    SearchResult:
      type: object
      properties:
        query:
          type: string
        results:
          type: array
          items:
            $ref: '#/components/schemas/StructuredResult'
        clusteredResults:
          type: array
          items:
            $ref: '#/components/schemas/SearchResult'
    StructuredResult:
      type: object
      properties:
        document:
          $ref: '#/components/schemas/Document'
        person:
          $ref: '#/components/schemas/Person'
        collection:
          $ref: '#/components/schemas/Collection'
%s
    Document:
      type: object
      properties:
        id:
          type: string
        title:
          type: string
        containerDocument:
          $ref: '#/components/schemas/Document'
        parentDocument:
          $ref: '#/components/schemas/Document'
        content:
          type: object
          properties:
            body:
              type: string
            format:
              type: string
        metadata:
          $ref: '#/components/schemas/DocumentMetadata'
        author:
          $ref: '#/components/schemas/Person'
    DocumentMetadata:
      type: object
      properties:
        created:
          type: string
        updated:
          type: string
        owner:
          $ref: '#/components/schemas/Person'
        relatedDocs:
          type: array
          items:
            $ref: '#/components/schemas/Document'
        tags:
          type: array
          items:
            type: string
        category:
          type: string
        priority:
          type: integer
        status:
          type: string
        labels:
          type: array
          items:
            type: string
        customFields:
          type: object
          additionalProperties:
            type: string
    Person:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
        documents:
          type: array
          items:
            $ref: '#/components/schemas/Document'
        manager:
          $ref: '#/components/schemas/Person'
        team:
          $ref: '#/components/schemas/Collection'
    Collection:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        items:
          type: array
          items:
            $ref: '#/components/schemas/Document'
        owner:
          $ref: '#/components/schemas/Person'
        subcollections:
          type: array
          items:
            $ref: '#/components/schemas/Collection'
%s`, pathsBuilder.String(), unionProps.String(), leafSchemas.String())

	ctx := context.Background()

	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(spec)))
	if err != nil {
		b.Fatalf("Failed to unmarshal: %v", err)
	}

	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}

	b.ResetTimer()
	for range b.N {
		openapi.BuildIndex(ctx, doc, resolveOpts)
	}
}
