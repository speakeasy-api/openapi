// Package oq implements a pipeline query language for OpenAPI schema graphs.
//
// Queries are written as pipeline expressions like:
//
//	schemas.components | where depth > 5 | sort depth desc | take 10 | select name, depth
package oq

import (
	"fmt"

	"github.com/speakeasy-api/openapi/graph"
)

// ResultKind distinguishes between schema and operation result rows.
type ResultKind int

const (
	SchemaResult ResultKind = iota
	OperationResult
)

// Row represents a single result in the pipeline.
type Row struct {
	Kind      ResultKind
	SchemaIdx int // index into SchemaGraph.Schemas
	OpIdx     int // index into SchemaGraph.Operations

	// Edge annotations (populated by 1-hop traversal stages)
	EdgeKind  string // edge type: "property", "items", "allOf", "oneOf", "ref", etc.
	EdgeLabel string // edge label: property name, array index, etc.
	EdgeFrom  string // source node name
}

// Result is the output of a query execution.
type Result struct {
	Rows       []Row
	Fields     []string // projected fields (empty = all)
	IsCount    bool
	Count      int
	Groups     []GroupResult
	Explain    string // human-readable pipeline explanation
	FormatHint string // format preference from format stage (table, json, markdown)
}

// GroupResult represents a group-by aggregation result.
type GroupResult struct {
	Key   string
	Count int
	Names []string
}

// Execute parses and executes a query against the given graph.
func Execute(query string, g *graph.SchemaGraph) (*Result, error) {
	stages, err := Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return run(stages, g)
}

// --- AST ---

// StageKind represents the type of pipeline stage.
type StageKind int

const (
	StageSource StageKind = iota
	StageWhere
	StageSelect
	StageSort
	StageTake
	StageUnique
	StageGroupBy
	StageCount
	StageRefsOut
	StageRefsIn
	StageReachable
	StageAncestors
	StageProperties
	StageUnionMembers
	StageItems
	StageOps
	StageSchemas
	StageExplain
	StageFields
	StageSample
	StagePath
	StageTop
	StageBottom
	StageFormat
	StageConnected
	StageBlastRadius
	StageNeighbors
	StageOrphans
	StageLeaves
	StageCycles
	StageClusters
	StageTagBoundary
	StageSharedRefs
)

// Stage represents a single stage in the query pipeline.
type Stage struct {
	Kind      StageKind
	Source    string   // for StageSource
	Expr      string   // for StageWhere
	Fields    []string // for StageSelect, StageGroupBy
	SortField string   // for StageSort
	SortDesc  bool     // for StageSort
	Limit     int      // for StageTake, StageSample, StageTop, StageBottom
	PathFrom  string   // for StagePath
	PathTo    string   // for StagePath
	Format    string   // for StageFormat
}
