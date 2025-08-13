package oas3

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_Success(t *testing.T) {
	t.Parallel()
	// Create a simple schema for testing
	schema := NewJSONSchemaFromSchema[Referenceable](&Schema{
		Type: NewTypeFromString("object"),
		Properties: sequencedmap.New(
			sequencedmap.NewElem("name", NewJSONSchemaFromSchema[Referenceable](&Schema{
				Type: NewTypeFromString("string"),
			})),
			sequencedmap.NewElem("age", NewJSONSchemaFromSchema[Referenceable](&Schema{
				Type: NewTypeFromString("integer"),
			})),
		),
	})

	ctx := t.Context()
	var visitedSchemas []*JSONSchema[Referenceable]
	var visitedLocations []string

	// Walk the schema and collect visited items
	for item := range Walk(ctx, schema) {
		err := item.Match(SchemaMatcher{
			Schema: func(s *JSONSchema[Referenceable]) error {
				visitedSchemas = append(visitedSchemas, s)
				visitedLocations = append(visitedLocations, string(item.Location.ToJSONPointer()))
				return nil
			},
		})
		require.NoError(t, err)
	}

	// Verify we visited the expected schemas
	assert.Len(t, visitedSchemas, 3, "Should visit root schema and 2 property schemas")
	assert.Contains(t, visitedLocations, "/", "Should visit root schema")
	assert.Contains(t, visitedLocations, "/properties/name", "Should visit name property schema")
	assert.Contains(t, visitedLocations, "/properties/age", "Should visit age property schema")
}

func TestWalkExternalDocs_Success(t *testing.T) {
	t.Parallel()
	// Create external docs for testing
	externalDocs := &ExternalDocumentation{
		URL:         "https://example.com/docs",
		Description: pointer.From("Example documentation"),
	}

	ctx := t.Context()
	var visitedItems []string

	// Walk the external docs and collect visited items
	for item := range WalkExternalDocs(ctx, externalDocs) {
		err := item.Match(SchemaMatcher{
			ExternalDocs: func(ed *ExternalDocumentation) error {
				visitedItems = append(visitedItems, "externalDocs")
				return nil
			},
			Extensions: func(ext *extensions.Extensions) error {
				visitedItems = append(visitedItems, "extensions")
				return nil
			},
		})
		require.NoError(t, err)
	}

	// Verify we visited the expected items
	assert.Contains(t, visitedItems, "externalDocs", "Should visit external docs")
	assert.Contains(t, visitedItems, "extensions", "Should visit extensions")
}

func TestWalk_NilSchema(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	count := 0

	// Walk a nil schema - should not yield any items
	for range Walk(ctx, nil) {
		count++
	}

	assert.Equal(t, 0, count, "Walking nil schema should yield no items")
}
