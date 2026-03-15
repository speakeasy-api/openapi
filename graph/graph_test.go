package graph_test

import (
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestGraph(t *testing.T) *graph.SchemaGraph {
	t.Helper()

	f, err := os.Open("../oq/testdata/petstore.yaml")
	require.NoError(t, err)
	defer f.Close()

	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, f, openapi.WithSkipValidation())
	require.NoError(t, err)
	require.NotNil(t, doc)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "../oq/testdata/petstore.yaml",
	})

	return graph.Build(ctx, idx)
}

func TestBuild_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	assert.NotEmpty(t, g.Schemas, "should have schema nodes")
	assert.NotEmpty(t, g.Operations, "should have operation nodes")
}

func TestBuild_ComponentSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	componentNames := make(map[string]bool)
	for _, s := range g.Schemas {
		if s.IsComponent {
			componentNames[s.Name] = true
		}
	}

	assert.True(t, componentNames["Pet"])
	assert.True(t, componentNames["Owner"])
	assert.True(t, componentNames["Address"])
	assert.True(t, componentNames["Error"])
	assert.True(t, componentNames["Shape"])
	assert.True(t, componentNames["Circle"])
	assert.True(t, componentNames["Square"])
	assert.True(t, componentNames["Unused"])
}

func TestBuild_SchemaByName_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, ok := g.SchemaByName("Pet")
	assert.True(t, ok)
	assert.Equal(t, "Pet", pet.Name)
	assert.Equal(t, "object", pet.Type)
	assert.True(t, pet.IsComponent)

	_, ok = g.SchemaByName("NonExistent")
	assert.False(t, ok)
}

func TestBuild_Edges_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, _ := g.SchemaByName("Pet")
	edges := g.OutEdges(pet.ID)

	// Pet has properties: id, name, tag, status, owner
	assert.Len(t, edges, 5, "Pet should have 5 out-edges")

	edgeLabels := make(map[string]graph.EdgeKind)
	for _, e := range edges {
		edgeLabels[e.Label] = e.Kind
	}
	assert.Equal(t, graph.EdgeProperty, edgeLabels["id"])
	assert.Equal(t, graph.EdgeProperty, edgeLabels["name"])
	assert.Equal(t, graph.EdgeProperty, edgeLabels["tag"])
	assert.Equal(t, graph.EdgeProperty, edgeLabels["status"])
	assert.Equal(t, graph.EdgeProperty, edgeLabels["owner"])
}

func TestBuild_Ancestors_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	addr, _ := g.SchemaByName("Address")
	ancestors := g.Ancestors(addr.ID)
	assert.NotEmpty(t, ancestors, "Address should have ancestors")

	ancestorNames := make(map[string]bool)
	for _, id := range ancestors {
		ancestorNames[g.Schemas[id].Name] = true
	}

	assert.True(t, ancestorNames["Owner"], "Owner should be an ancestor of Address")
}

func TestBuild_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	opNames := make(map[string]bool)
	for _, op := range g.Operations {
		opNames[op.Name] = true
	}

	assert.True(t, opNames["listPets"])
	assert.True(t, opNames["createPet"])
	assert.True(t, opNames["showPetById"])
	assert.True(t, opNames["listOwners"])
}

func TestBuild_OperationSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	for _, op := range g.Operations {
		if op.OperationID == "listPets" {
			schemas := g.OperationSchemas(op.ID)
			assert.NotEmpty(t, schemas, "listPets should reference schemas")
			assert.Positive(t, op.SchemaCount)
			return
		}
	}
	t.Fatal("listPets operation not found")
}

func TestBuild_ShortestBidiPath_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, _ := g.SchemaByName("Pet")
	addr, _ := g.SchemaByName("Address")
	hops := g.ShortestBidiPath(pet.ID, addr.ID)
	assert.NotEmpty(t, hops, "should find bidi path from Pet to Address")
	assert.Equal(t, pet.ID, hops[0].Node)
	assert.Equal(t, addr.ID, hops[len(hops)-1].Node)

	// Each hop after the first should have direction info
	for i, hop := range hops[1:] {
		assert.NotNil(t, hop.Edge, "hop %d should have an edge", i+1)
	}
}

func TestBuild_ShortestBidiPath_NoPath_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	unused, _ := g.SchemaByName("Unused")
	pet, _ := g.SchemaByName("Pet")
	hops := g.ShortestBidiPath(unused.ID, pet.ID)
	assert.Empty(t, hops, "Unused should not reach Pet")
}

func TestBuild_ShortestBidiPath_SameNode_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, _ := g.SchemaByName("Pet")
	hops := g.ShortestBidiPath(pet.ID, pet.ID)
	assert.Len(t, hops, 1, "path from node to itself should be length 1")
	assert.Equal(t, pet.ID, hops[0].Node)
}

func TestBuild_Metrics_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, _ := g.SchemaByName("Pet")
	assert.Equal(t, 5, pet.PropertyCount, "Pet should have 5 properties")
	assert.Equal(t, 5, pet.OutDegree, "Pet should have 5 out-edges")
	assert.Positive(t, pet.InDegree, "Pet should be referenced")
	assert.NotEmpty(t, pet.Hash, "Pet should have a hash")

	shape, _ := g.SchemaByName("Shape")
	assert.Equal(t, 2, shape.UnionWidth, "Shape should have union_width 2 (oneOf)")

	unused, _ := g.SchemaByName("Unused")
	assert.Equal(t, 0, unused.InDegree, "Unused should have no incoming edges from other schemas")
}

func TestBuild_InEdges_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Owner is referenced by Pet via the "owner" property (possibly through a $ref node)
	owner, _ := g.SchemaByName("Owner")
	inEdges := g.InEdges(owner.ID)
	assert.NotEmpty(t, inEdges, "Owner should have incoming edges")

	// Verify the InEdges returns edges with correct To field
	for _, e := range inEdges {
		assert.Equal(t, owner.ID, e.To, "InEdge To should match the queried node")
	}
}

func TestBuild_SchemaOperations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, _ := g.SchemaByName("Pet")
	ops := g.SchemaOperations(pet.ID)
	assert.NotEmpty(t, ops, "Pet should be referenced by operations")
}

func TestBuild_SchemaOpCount_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	pet, _ := g.SchemaByName("Pet")
	count := g.SchemaOpCount(pet.ID)
	assert.Positive(t, count, "Pet should have operations referencing it")

	unused, _ := g.SchemaByName("Unused")
	count = g.SchemaOpCount(unused.ID)
	assert.Equal(t, 0, count, "Unused should have no operations")
}

func TestBuild_StronglyConnectedComponents_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	sccs := g.StronglyConnectedComponents()
	// Petstore shouldn't have cycles, so SCCs should be empty (no multi-node components)
	assert.Empty(t, sccs, "petstore should have no strongly connected components")
}
