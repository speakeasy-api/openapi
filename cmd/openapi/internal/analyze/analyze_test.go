package analyze_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadDoc(t *testing.T, path string) *openapi.OpenAPI {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	doc, _, err := openapi.Unmarshal(context.Background(), f, openapi.WithSkipValidation())
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

func TestBuildGraph_SimpleSpec(t *testing.T) {
	doc := loadDoc(t, "../../../../openapi/testdata/test.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)

	assert.NotEmpty(t, g.Nodes)
	assert.Greater(t, len(g.Nodes), 0)

	// User references UserPreferences
	userEdges := g.OutEdges["User"]
	assert.NotEmpty(t, userEdges, "User should have outgoing edges")

	found := false
	for _, e := range userEdges {
		if e.To == "UserPreferences" {
			found = true
			assert.Equal(t, analyze.EdgeProperty, e.Kind)
		}
	}
	assert.True(t, found, "User should reference UserPreferences")
}

func TestBuildGraph_CyclicSpec(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)

	assert.Equal(t, 10, len(g.Nodes))
	assert.Greater(t, len(g.Edges), 0)

	// TreeNode has self-references
	treeEdges := g.OutEdges["TreeNode"]
	selfRefCount := 0
	for _, e := range treeEdges {
		if e.To == "TreeNode" {
			selfRefCount++
		}
	}
	assert.GreaterOrEqual(t, selfRefCount, 2, "TreeNode should have at least 2 self-references (parent + children)")
}

func TestAnalyzeCycles_DetectsSCCs(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	assert.Greater(t, len(ca.SCCs), 0, "Should detect SCCs")
	assert.Greater(t, len(ca.Cycles), 0, "Should detect cycles")
	assert.NotEmpty(t, ca.NodesInCycles, "Should have nodes in cycles")

	// Person <-> Company is a required-only cycle
	hasRequiredCycle := false
	for _, c := range ca.Cycles {
		if c.HasRequiredOnlyPath {
			hasRequiredCycle = true
			break
		}
	}
	assert.True(t, hasRequiredCycle, "Should detect required-only cycle (Person <-> Company)")
}

func TestAnalyzeCycles_NoCyclesInSimpleSpec(t *testing.T) {
	doc := loadDoc(t, "../../../../openapi/testdata/test.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)

	assert.Empty(t, ca.SCCs, "Simple spec should have no non-trivial SCCs")
	assert.Empty(t, ca.Cycles, "Simple spec should have no cycles")
}

func TestComputeMetrics(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)
	metrics := analyze.ComputeMetrics(g, ca)

	// Person has high fan-in (referenced by Company, Dog, Department, Event.data.anyOf)
	personMetrics := metrics["Person"]
	require.NotNil(t, personMetrics)
	assert.Equal(t, 4, personMetrics.FanIn)
	assert.True(t, personMetrics.InSCC)

	// BigSchema has high property count
	bigMetrics := metrics["BigSchema"]
	require.NotNil(t, bigMetrics)
	assert.Equal(t, 31, bigMetrics.PropertyCount)
	assert.Equal(t, 31, bigMetrics.DeepPropertyCount) // all at root level
	assert.Equal(t, 0, bigMetrics.NestingDepth)
	assert.Equal(t, 0, bigMetrics.CompositionDepth)
	assert.Equal(t, 0, bigMetrics.UnionSiteCount)

	// Animal has 1 union site (oneOf, width 2, no discriminator)
	animalMetrics := metrics["Animal"]
	require.NotNil(t, animalMetrics)
	assert.Equal(t, 1, animalMetrics.UnionSiteCount)
	assert.Equal(t, 2, animalMetrics.MaxUnionWidth)
	assert.Equal(t, 2, animalMetrics.VariantProduct)
	assert.Equal(t, 1, animalMetrics.CompositionDepth)

	// Event has 1 union site (anyOf at data property, width 2)
	eventMetrics := metrics["Event"]
	require.NotNil(t, eventMetrics)
	assert.Equal(t, 1, eventMetrics.UnionSiteCount)
	assert.Equal(t, 2, eventMetrics.MaxUnionWidth)
	assert.Equal(t, 1, eventMetrics.CompositionDepth)
}

func TestAssessCodegen(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	g := analyze.BuildGraph(context.Background(), doc)
	ca := analyze.AnalyzeCycles(g)
	metrics := analyze.ComputeMetrics(g, ca)
	report := analyze.AssessCodegen(g, ca, metrics)

	// Person and Company should be red (required cycle)
	assert.Equal(t, analyze.CodegenRed, report.PerSchema["Person"].Tier)
	assert.Equal(t, analyze.CodegenRed, report.PerSchema["Company"].Tier)

	// Animal should be yellow (oneOf without discriminator)
	assert.Equal(t, analyze.CodegenYellow, report.PerSchema["Animal"].Tier)

	// Event has anyOf inside its inline `data` property — now detected as red
	assert.Equal(t, analyze.CodegenRed, report.PerSchema["Event"].Tier)

	// BigSchema should be yellow (high property count)
	assert.Equal(t, analyze.CodegenYellow, report.PerSchema["BigSchema"].Tier)

	assert.Greater(t, report.RedCount, 0)
	assert.Greater(t, report.YellowCount, 0)
	assert.Greater(t, report.GreenCount, 0)
}

func TestGenerateSuggestions(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	report := analyze.Analyze(context.Background(), doc)

	assert.NotEmpty(t, report.Suggestions, "Should generate suggestions")

	// Should have cycle break suggestions
	hasCutEdge := false
	hasDiscriminator := false
	hasPropertyReduction := false
	for _, s := range report.Suggestions {
		switch s.Type {
		case analyze.SuggestionCutEdge:
			hasCutEdge = true
		case analyze.SuggestionAddDiscriminator:
			hasDiscriminator = true
		case analyze.SuggestionReducePropertyCount:
			hasPropertyReduction = true
		}
	}
	assert.True(t, hasCutEdge, "Should suggest cutting edges to break cycles")
	assert.True(t, hasDiscriminator, "Should suggest adding discriminator to Animal")
	assert.True(t, hasPropertyReduction, "Should suggest splitting BigSchema")
}

func TestAnalyze_FullPipeline(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	report := analyze.Analyze(context.Background(), doc)

	assert.Equal(t, "Cyclic Schema Test", report.DocumentTitle)
	assert.Equal(t, "1.0.0", report.DocumentVersion)
	assert.Equal(t, 10, report.TotalSchemas)
	assert.Greater(t, report.TotalEdges, 0)
	assert.Greater(t, report.SCCCount, 0)
	assert.Greater(t, report.SchemasInCyclesPct, 0.0)
	assert.Less(t, report.CompatibilityScore, 100.0)
	assert.NotEmpty(t, report.TopFanIn)
	assert.NotEmpty(t, report.TopComplex)
}

func TestWriteJSON(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	report := analyze.Analyze(context.Background(), doc)

	var buf bytes.Buffer
	err := analyze.WriteJSON(&buf, report)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Cyclic Schema Test")
	assert.Contains(t, output, `"sccCount": 3`)
	assert.Contains(t, output, `"codegenTier"`)
	assert.Contains(t, output, `"complexityScore"`)
	assert.Contains(t, output, `"rank"`)
	assert.Contains(t, output, `"deepPropertyCount"`)
}

func TestWriteDOT(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	report := analyze.Analyze(context.Background(), doc)

	var buf bytes.Buffer
	analyze.WriteDOT(&buf, report)

	output := buf.String()
	assert.Contains(t, output, "digraph schemas")
	assert.Contains(t, output, "Person")
	assert.Contains(t, output, "Company")
	assert.Contains(t, output, "->")
	assert.Contains(t, output, "fillcolor")
	// Red tier schemas should have red fill
	assert.Contains(t, output, "#f8d7da")
	// Green tier schemas should have green fill
	assert.Contains(t, output, "#d4edda")
}

func TestWriteText(t *testing.T) {
	doc := loadDoc(t, "testdata/cyclic.openapi.yaml")
	report := analyze.Analyze(context.Background(), doc)

	var buf bytes.Buffer
	analyze.WriteText(&buf, report)

	output := buf.String()
	assert.Contains(t, output, "Schema Complexity Report")
	assert.Contains(t, output, "CYCLE HEALTH")
	assert.Contains(t, output, "CODEGEN COMPATIBILITY")
	assert.Contains(t, output, "MOST COMPLEX")
	assert.Contains(t, output, "score=")
	assert.Contains(t, output, "SUGGESTIONS")
}
