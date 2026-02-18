package analyze_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCCToMermaid(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	require.NotEmpty(t, ca.SCCs)

	out := analyze.SCCToMermaid(g, ca, 0)
	assert.Contains(t, out, "graph LR")
	// Should contain at least one node from the SCC
	assert.Contains(t, out, ca.SCCs[0].NodeIDs[0])
	// Should contain edges (-->)
	assert.Contains(t, out, "-->")
}

func TestSCCToMermaid_OutOfBounds(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	assert.Empty(t, analyze.SCCToMermaid(g, ca, -1))
	assert.Empty(t, analyze.SCCToMermaid(g, ca, 999))
}

func TestEgoGraphToMermaid(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)

	out := analyze.EgoGraphToMermaid(g, "Person", 1)
	assert.Contains(t, out, "graph LR")
	assert.Contains(t, out, "Person")
	// Person is the center — should use double-circle notation
	assert.Contains(t, out, "((Person))")
	// Should include neighbors
	assert.Contains(t, out, "Company")
}

func TestEgoGraphToMermaid_UnknownNode(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)

	assert.Empty(t, analyze.EgoGraphToMermaid(g, "NonExistent", 1))
}

func TestDAGOverviewToMermaid(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	out := analyze.DAGOverviewToMermaid(g, ca, 0)
	assert.Contains(t, out, "graph TD")
	// Should contain SCC nodes
	assert.Contains(t, out, "scc")
}

func TestRenderASCIIGraph(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	out := analyze.RenderASCIIGraph(ca.DAGCondensation, 80)
	assert.NotEmpty(t, out)
	// Should contain box drawing characters
	assert.Contains(t, out, "┌")
	assert.Contains(t, out, "│")
}

func TestRenderASCIIGraph_NilDAG(t *testing.T) {
	out := analyze.RenderASCIIGraph(nil, 80)
	assert.Contains(t, out, "no schemas")
}

func TestRenderASCIIEgoGraph(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)

	out := analyze.RenderASCIIEgoGraph(g, "Person", 2, 80)
	assert.Contains(t, out, "Person")
	assert.Contains(t, out, "Company")
	// Should have box drawing for center node
	assert.Contains(t, out, "╔")
}

func TestRenderASCIIEgoGraph_UnknownNode(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)

	out := analyze.RenderASCIIEgoGraph(g, "NonExistent", 2, 80)
	assert.Contains(t, out, "not found")
}

func TestRenderASCIISCC(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	require.NotEmpty(t, ca.SCCs)

	out := analyze.RenderASCIISCC(g, ca, 0)
	assert.Contains(t, out, "SCC #1")
	assert.Contains(t, out, ca.SCCs[0].NodeIDs[0])
}
