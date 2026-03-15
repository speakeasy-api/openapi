// Package oq implements a pipeline query language for OpenAPI schema graphs.
//
// Queries are written as pipeline expressions:
//
//	schemas | where(depth > 5) | sort-by(depth, desc) | take(10) | select name, depth
package oq

import (
	"fmt"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
)

// ResultKind distinguishes between schema and operation result rows.
type ResultKind int

const (
	SchemaResult ResultKind = iota
	OperationResult
	GroupRowResult
	ParameterResult
	ResponseResult
	RequestBodyResult
	ContentTypeResult
	HeaderResult
	SecuritySchemeResult
	SecurityRequirementResult
	ServerResult
	TagResult
	LinkResult
)

// Row represents a single result in the pipeline.
type Row struct {
	Kind      ResultKind
	SchemaIdx int // index into SchemaGraph.Schemas
	OpIdx     int // index into SchemaGraph.Operations

	// Edge annotations (populated by traversal stages)
	Via       string // edge type: "property", "items", "allOf", "oneOf", "ref", etc.
	Key       string // edge key: property name, array index, etc.
	From      string // source node name (the node that contains the reference)
	Target    string // target/seed node name (the node traversal originated from)
	Direction string // "→" (outgoing) or "←" (incoming) — set by bidi traversals

	// BFS depth (populated by depth-limited traversals)
	BFSDepth int

	// Group annotations (populated by group-by stages)
	GroupKey   string   // group key value
	GroupCount int      // number of members in the group
	GroupNames []string // member names

	// Navigation objects (populated by navigation stages)
	Parameter      *openapi.Parameter
	Response       *openapi.Response
	RequestBody    *openapi.RequestBody
	ContentType    *openapi.MediaType
	Header         *openapi.Header
	SecurityScheme *openapi.SecurityScheme

	// Propagated context from parent navigation stages
	StatusCode    string   // propagated from response rows to content-types/headers
	MediaTypeName string   // media type key (e.g., "application/json")
	HeaderName    string   // header name
	ComponentKey  string   // component key name or parameter name
	SchemeName    string   // security scheme name
	Scopes        []string // security requirement scopes
	SourceOpIdx   int      // operation this row originated from (-1 if N/A)

	// Server, Tag, Link objects (populated by server/tag/link sources/stages)
	Server       *openapi.Server
	Tag          *openapi.Tag
	Link         *openapi.Link
	LinkName     string // link name within response
	CallbackName string // callback name within operation
}

// Result is the output of a query execution.
type Result struct {
	Rows       []Row
	Fields     []string // projected fields (empty = all)
	IsCount    bool
	Count      int
	Groups     []GroupResult
	Explain    string // human-readable pipeline explanation
	FormatHint string // format preference from format stage (table, json, markdown, toon)
	EmitYAML   bool   // emit raw YAML nodes instead of formatted output
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
	StageRefs
	StageProperties
	StageItems
	StageToOperations
	StageToSchemas
	StageExplain
	StageFields
	StageSample
	StagePath
	StageHighest
	StageLowest
	StageFormat
	StageBlastRadius
	StageOrphans
	StageLeaves
	StageCycles
	StageClusters
	StageCrossTag
	StageSharedRefs
	StageLast
	StageLet
	StageOrigin
	StageToYAML
	StageParameters
	StageResponses
	StageRequestBody
	StageContentTypes
	StageHeaders
	StageToSchema             // singular: extract schema from nav row
	StageOperation            // back-navigate to source operation
	StageSecurity             // operation security requirements
	StageMembers              // union members (allOf/oneOf/anyOf children) or group row expansion
	StageCallbacks            // operation callbacks → operations
	StageLinks                // response links
	StageAdditionalProperties // schema additional properties traversal
	StagePatternProperties    // schema pattern properties traversal
)

// Stage represents a single stage in the query pipeline.
type Stage struct {
	Kind      StageKind
	Source    string   // for StageSource
	Expr      string   // for StageWhere, StageLet
	Fields    []string // for StageSelect, StageGroupBy
	SortField string   // for StageSort
	SortDesc  bool     // for StageSort
	Limit     int      // for StageTake, StageLast, StageSample, StageHighest, StageLowest, StageRefs
	PathFrom  string   // for StagePath
	PathTo    string   // for StagePath
	Format    string   // for StageFormat
	VarName   string   // for StageLet
	RefsDir   string   // for StageRefs: "out", "in", or "" (bidi)
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
