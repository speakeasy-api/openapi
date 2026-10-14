package openapi_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIndex_GetDocumentLocation_Success(t *testing.T) {
	t.Parallel()
	doc, _, err := openapi.Unmarshal(t.Context(), strings.NewReader("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\n"))
	require.NoError(t, err, "main document should parse")
	doc.InitCache()
	first := &yaml.Node{Kind: yaml.ScalarNode, Value: "same", Line: 3, Column: 5}
	second := &yaml.Node{Kind: yaml.ScalarNode, Value: "same", Line: 3, Column: 5}
	doc.StoreExternalDocumentInCache("/spec/first.yaml", &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{first}})
	doc.StoreExternalDocumentInCache("https://schemas.example.com/second.yaml", &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{second}})
	doc.StoreExternalDocumentInCache("/spec/main.yaml", doc.GetRootNode())
	doc.StoreExternalDocumentInCache("/spec/non-yaml", "not a parsed YAML document")
	idx := &openapi.Index{Doc: doc}

	assert.Equal(t, "/spec/first.yaml", idx.GetDocumentLocation(first), "source identity should be based on the original node")
	assert.Equal(t, "https://schemas.example.com/second.yaml", idx.GetDocumentLocation(second), "identical coordinates in another document must stay distinct")
	assert.Empty(t, idx.GetDocumentLocation(doc.GetRootNode()), "the main document should retain its existing attribution")
	assert.Empty(t, idx.GetDocumentLocation(doc.GetCore().Info.ValueNode), "main-document descendants should not acquire an external filename")
	assert.Empty(t, idx.GetDocumentLocation(&yaml.Node{Line: 3, Column: 5}), "unknown nodes must not be guessed from coordinates")
}

func TestIndex_GetDocumentLocation_AliasCycleAndConcurrentReads(t *testing.T) {
	t.Parallel()
	doc := &openapi.OpenAPI{}
	doc.InitCache()
	root := &yaml.Node{Kind: yaml.MappingNode}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: root}
	root.Content = []*yaml.Node{alias}
	doc.StoreExternalDocumentInCache("/spec/recursive.yaml", root)
	idx := &openapi.Index{Doc: doc}

	for range 8 {
		t.Run("concurrent lookup", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/spec/recursive.yaml", idx.GetDocumentLocation(alias), "alias cycles should terminate and concurrent first reads should be safe")
			assert.Equal(t, "/spec/recursive.yaml", idx.GetDocumentLocation(root), "alias targets should retain their source")
		})
	}
}

func TestIndex_GetDocumentLocation_UsesResolutionCacheOwner(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    External:
      $ref: './schema.yaml'
`))
	require.NoError(t, err, "main document should parse")
	cacheOwner := &openapi.OpenAPI{}
	fs := NewMockVirtualFS()
	fs.AddFile("/spec/schema.yaml", "type: string\n")
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument: cacheOwner, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
	})
	require.False(t, idx.HasErrors(), "external schema should resolve through the supplied cache owner")
	require.Len(t, idx.ExternalSchemas, 1, "external schema should be indexed")
	node := idx.ExternalSchemas[0].Node.GetSchema().GetCore().Type.ValueNode
	assert.Equal(t, "/spec/schema.yaml", idx.GetDocumentLocation(node), "provenance must use the actual resolution cache, not assume the indexed document owns it")
}

func TestIndex_SourceDiagnostics_ReferenceChain(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    External:
      $ref: './alias.yaml'
`))
	require.NoError(t, err, "main document should parse")
	fs := NewMockVirtualFS()
	fs.AddFile("/spec/alias.yaml", "$ref: './types.yaml#/Broken'\n")
	fs.AddFile("/spec/types.yaml", "Broken:\n  type: invalid\n")
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
	})
	require.NotEmpty(t, idx.GetValidationErrors(), "the invalid target should produce validation diagnostics")
	for _, err := range idx.GetValidationErrors() {
		var finding *validation.Error
		require.ErrorAs(t, err, &finding, "validation diagnostic should expose metadata")
		assert.Equal(t, "/spec/types.yaml", finding.DocumentLocation, "target diagnostics must point to the final source, not the alias")
		assert.Equal(t, 2, finding.GetLineNumber(), "the final source's coordinates should be retained")
	}
}

func TestIndex_GetDocumentLocation_Empty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		idx  *openapi.Index
		node *yaml.Node
	}{
		{name: "nil index", node: &yaml.Node{}},
		{name: "missing document", idx: &openapi.Index{}, node: &yaml.Node{}},
		{name: "nil node", idx: &openapi.Index{Doc: &openapi.OpenAPI{}}},
		{name: "no cached documents", idx: &openapi.Index{Doc: &openapi.OpenAPI{}}, node: &yaml.Node{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, tt.idx.GetDocumentLocation(tt.node), "absent source information should be safe and remain unknown")
		})
	}
}
