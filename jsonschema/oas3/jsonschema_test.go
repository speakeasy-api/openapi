package oas3_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
)

func TestJSONSchema_DocumentMetadata_Success(t *testing.T) {
	t.Parallel()
	parent := &oas3.Schema{}
	doc := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{ID: pointer.From("https://example.com/root.json#anchor")})
	assert.Equal(t, "https://example.com/root.json", doc.GetDocumentBaseURI(), "document base should come from the root ID without its fragment")
	registry := doc.GetSchemaRegistry()
	assert.NotNil(t, registry, "standalone documents should lazily create their registry")
	assert.Same(t, registry, doc.GetSchemaRegistry(), "repeated registry lookups should return the same registry")
	doc.SetDocumentBaseURI("https://example.com/override.json#fragment")
	assert.Equal(t, "https://example.com/override.json", doc.GetDocumentBaseURI(), "an explicit document base should override the root ID")
	replacement := oas3.NewSchemaRegistry("https://example.com/override.json")
	doc.SetSchemaRegistry(replacement)
	assert.Same(t, replacement, doc.GetSchemaRegistry(), "explicit registry replacement should be retained")
	doc.SetEnclosingSchema(parent)
	assert.Same(t, parent, doc.GetEnclosingSchema(), "enclosing schema should preserve its identity")
	doc.SetEnclosingSchema(nil)
	assert.Nil(t, doc.GetEnclosingSchema(), "enclosing schema should be clearable")
	doc.SetDocumentBaseURI("")
	assert.Equal(t, "https://example.com/root.json", doc.GetDocumentBaseURI(), "clearing the explicit base should restore the root ID")
	assert.Empty(t, oas3.NewJSONSchemaFromBool(true).GetDocumentBaseURI(), "boolean schemas should have no implicit document base")
}

func TestJSONSchema_NilMetadata_ReturnsDefaults(t *testing.T) {
	t.Parallel()
	var doc *oas3.JSONSchema[oas3.Referenceable]
	doc.SetSchemaRegistry(oas3.NewSchemaRegistry("https://example.com"))
	doc.SetDocumentBaseURI("https://example.com")
	doc.SetEnclosingSchema(&oas3.Schema{})
	assert.Nil(t, doc.GetSchemaRegistry(), "nil schemas should not allocate registries")
	assert.Empty(t, doc.GetDocumentBaseURI(), "nil schemas should have no base URI")
	assert.Nil(t, doc.GetEnclosingSchema(), "nil schemas should have no enclosing schema")
}

func TestSchema_OwningDocument_Success(t *testing.T) {
	t.Parallel()
	schema := &oas3.Schema{}
	assert.Nil(t, schema.GetOwningDocument(), "new schemas should not have an owning document")
	doc := oas3.NewJSONSchemaFromBool(true)
	schema.SetOwningDocument(doc)
	assert.Same(t, doc, schema.GetOwningDocument(), "document providers should retain their identity")
	assert.Same(t, doc.GetSchemaRegistry(), schema.GetSchemaRegistry(), "child schemas should use their owning document registry")
	schema.SetOwningDocument("not a document provider")
	assert.Same(t, doc, schema.GetOwningDocument(), "unsupported providers should leave the existing owner unchanged")
	schema.SetOwningDocument(nil)
	assert.Nil(t, schema.GetOwningDocument(), "nil should explicitly clear document ownership")
	assert.Nil(t, schema.GetSchemaRegistry(), "cleared ownership should no longer expose a registry")

	var absent *oas3.Schema
	absent.SetOwningDocument(doc)
	absent.SetEffectiveBaseURI("https://example.com")
	absent.SetParent(doc)
	assert.Nil(t, absent.GetOwningDocument(), "nil schemas should ignore document setters")
	assert.Empty(t, absent.GetEffectiveBaseURI(), "nil schemas should ignore base URI setters")
	assert.Nil(t, absent.GetParent(), "nil schemas should ignore parent setters")
}
