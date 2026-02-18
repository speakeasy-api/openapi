package analyze

import (
	"context"

	"github.com/speakeasy-api/openapi/openapi"
)

// Report is the top-level analysis result tying all analysis together.
type Report struct {
	// DocumentTitle is the title from the OpenAPI info object.
	DocumentTitle string
	// DocumentVersion is the version from the OpenAPI info object.
	DocumentVersion string
	// OpenAPIVersion is the OpenAPI spec version (e.g., "3.1.0").
	OpenAPIVersion string

	// Schema counts
	TotalSchemas    int
	TotalEdges      int
	ComponentCount  int
	InlineCount     int

	// Graph is the extracted schema reference graph.
	Graph *Graph
	// Cycles is the cycle and SCC analysis.
	Cycles *CycleAnalysis
	// Metrics is per-schema complexity metrics.
	Metrics map[string]*SchemaMetrics
	// Codegen is the code generation difficulty assessment.
	Codegen *CodegenReport
	// Suggestions is the list of actionable refactoring suggestions.
	Suggestions []*Suggestion

	// Summary statistics
	SCCCount              int
	LargestSCCSize        int
	SchemasInCyclesPct    float64
	RequiredOnlyCycles    int
	CompatibilityScore    float64
	DAGDepth              int
	TopFanIn              []*SchemaMetrics
	TopFanOut             []*SchemaMetrics
	TopComplex            []*SchemaMetrics
}

// Analyze runs the full analysis pipeline on an OpenAPI document and returns a Report.
func Analyze(ctx context.Context, doc *openapi.OpenAPI) *Report {
	r := &Report{}

	// Document metadata
	if doc.Info.Title != "" {
		r.DocumentTitle = doc.Info.Title
	}
	if doc.Info.Version != "" {
		r.DocumentVersion = doc.Info.Version
	}
	r.OpenAPIVersion = doc.OpenAPI

	// Step 1: Build graph
	r.Graph = BuildGraph(ctx, doc)
	r.TotalEdges = len(r.Graph.Edges)

	// Count schemas
	for _, n := range r.Graph.Nodes {
		r.TotalSchemas++
		if n.IsComponent {
			r.ComponentCount++
		} else {
			r.InlineCount++
		}
	}

	// Step 2: Cycle analysis
	r.Cycles = AnalyzeCycles(r.Graph)
	r.SCCCount = len(r.Cycles.SCCs)
	r.LargestSCCSize = r.Cycles.LargestSCCSize
	if r.TotalSchemas > 0 {
		r.SchemasInCyclesPct = float64(len(r.Cycles.NodesInCycles)) / float64(r.TotalSchemas) * 100
	}
	for _, c := range r.Cycles.Cycles {
		if c.HasRequiredOnlyPath {
			r.RequiredOnlyCycles++
		}
	}
	if r.Cycles.DAGCondensation != nil {
		r.DAGDepth = r.Cycles.DAGCondensation.Depth
	}

	// Step 3: Metrics
	r.Metrics = ComputeMetrics(r.Graph, r.Cycles)

	// Step 4: Codegen assessment
	r.Codegen = AssessCodegen(r.Graph, r.Cycles, r.Metrics)
	r.CompatibilityScore = r.Codegen.CompatibilityScore

	// Step 5: Suggestions
	r.Suggestions = GenerateSuggestions(r.Graph, r.Cycles, r.Metrics, r.Codegen)

	// Step 6: Top-N rankings
	r.TopFanIn = TopSchemasByFanIn(r.Metrics, 5)
	r.TopFanOut = TopSchemasByFanOut(r.Metrics, 5)
	r.TopComplex = TopSchemasByComplexity(r.Metrics, 5)

	return r
}
