package oas3

import (
	"strconv"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_AllKeywordsAndStops_Success(t *testing.T) {
	t.Parallel()

	child := NewJSONSchemaFromBool(true)
	schema := NewJSONSchemaFromSchema[Referenceable](&Schema{
		AllOf: []*JSONSchemaReferenceable{child}, OneOf: []*JSONSchemaReferenceable{child}, AnyOf: []*JSONSchemaReferenceable{child},
		Discriminator: &Discriminator{PropertyName: "kind", Extensions: extensions.New()},
		PrefixItems:   []*JSONSchemaReferenceable{child}, Contains: child, If: child, Then: child, Else: child,
		DependentSchemas:  sequencedmap.New(sequencedmap.NewElem("id", child)),
		PatternProperties: sequencedmap.New(sequencedmap.NewElem("^x-", child)),
		PropertyNames:     child, UnevaluatedItems: child, UnevaluatedProperties: child,
		Items: child, ContentSchema: child, Not: child,
		Properties: sequencedmap.New(sequencedmap.NewElem("name", child)),
		Defs:       sequencedmap.New(sequencedmap.NewElem("Pet", child)), AdditionalProperties: child,
		ExternalDocs: &ExternalDocumentation{URL: "https://example.com", Extensions: extensions.New()},
		XML:          &XML{Name: pointer.From("pet"), Extensions: extensions.New()}, Extensions: extensions.New(),
	})
	want := []string{
		"/", "/allOf/0", "/oneOf/0", "/anyOf/0", "/discriminator", "/discriminator",
		"/prefixItems/0", "/contains", "/if", "/then", "/else", "/dependentSchemas/id", "/patternProperties/^x-",
		"/propertyNames", "/unevaluatedItems", "/unevaluatedProperties", "/items", "/contentSchema", "/not",
		"/properties/name", "/$defs/Pet", "/additionalProperties", "/externalDocs", "/externalDocs", "/xml", "/xml", "/",
	}
	var visited []string
	for item := range Walk(t.Context(), schema) {
		visited = append(visited, item.Location.ToJSONPointer().String())
		assert.Same(t, schema, item.Schema, "nested items should retain the root schema")
		err := item.Match(SchemaMatcher{Any: func(model any) error {
			assert.NotNil(t, model, "each walk item should match its model")
			return nil
		}})
		require.NoError(t, err, "matching a walk item should succeed")
	}
	assert.Equal(t, want, visited, "all schema keywords and metadata should have exact traversal locations")

	for stopAt, location := range want {
		t.Run(strconv.Itoa(stopAt)+":"+location, func(t *testing.T) {
			t.Parallel()
			var prefix []string
			Walk(t.Context(), schema)(func(item SchemaWalkItem) bool {
				prefix = append(prefix, item.Location.ToJSONPointer().String())
				return len(prefix) <= stopAt
			})
			assert.Equal(t, want[:stopAt+1], prefix, "stopping at any keyword or metadata node should stop the entire walk")
		})
	}
}

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
