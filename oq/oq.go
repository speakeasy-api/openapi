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
	return ExecuteWithSearchPaths(query, g, nil)
}

// ExecuteWithSearchPaths parses and executes a query, searching for modules in the given paths.
func ExecuteWithSearchPaths(query string, g *graph.SchemaGraph, searchPaths []string) (*Result, error) {
	decls, err := parseDeclarations(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Resolve includes
	for _, inc := range decls.Includes {
		defs, loadErr := LoadModule(inc, searchPaths)
		if loadErr != nil {
			return nil, fmt.Errorf("include %q: %w", inc, loadErr)
		}
		decls.Defs = append(decls.Defs, defs...)
	}

	// Text-level def expansion before parsing pipeline
	pipelineText, err := ExpandDefs(decls.PipelineText, decls.Defs)
	if err != nil {
		return nil, fmt.Errorf("def expansion: %w", err)
	}

	if pipelineText == "" {
		return &Result{}, nil
	}

	stages, err := parsePipeline(pipelineText)
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
	StageLast
	StageLet
)

// Stage represents a single stage in the query pipeline.
type Stage struct {
	Kind      StageKind
	Source    string   // for StageSource
	Expr      string   // for StageWhere, StageLet
	Fields    []string // for StageSelect, StageGroupBy
	SortField string   // for StageSort
	SortDesc  bool     // for StageSort
	Limit     int      // for StageTake, StageLast, StageSample, StageTop, StageBottom
	PathFrom  string   // for StagePath
	PathTo    string   // for StagePath
	Format    string   // for StageFormat
	VarName   string   // for StageLet
}

// Query represents a parsed query with optional includes, defs, and pipeline stages.
type Query struct {
	Includes []string
	Defs     []FuncDef
	Stages   []Stage
}

// FuncDef represents a user-defined function.
type FuncDef struct {
	Name   string
	Params []string // with $ prefix
	Body   string   // raw pipeline text
}
