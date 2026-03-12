// Package oq implements a pipeline query language for OpenAPI schema graphs.
//
// Queries are written as pipeline expressions like:
//
//	schemas.components | where depth > 5 | sort depth desc | take 10 | select name, depth
package oq

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/oq/expr"
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

// Parse splits a pipeline query string into stages.
func Parse(query string) ([]Stage, error) {
	// Split by pipe, respecting quoted strings
	parts := splitPipeline(query)
	if len(parts) == 0 {
		return nil, errors.New("empty query")
	}

	var stages []Stage

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if i == 0 {
			// First part is a source
			stages = append(stages, Stage{Kind: StageSource, Source: part})
			continue
		}

		stage, err := parseStage(part)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}

	return stages, nil
}

func parseStage(s string) (Stage, error) {
	// Extract the keyword
	keyword, rest := splitFirst(s)
	keyword = strings.ToLower(keyword)

	switch keyword {
	case "where":
		if rest == "" {
			return Stage{}, errors.New("where requires an expression")
		}
		return Stage{Kind: StageWhere, Expr: rest}, nil

	case "select":
		if rest == "" {
			return Stage{}, errors.New("select requires field names")
		}
		fields := parseCSV(rest)
		return Stage{Kind: StageSelect, Fields: fields}, nil

	case "sort":
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			return Stage{}, errors.New("sort requires a field name")
		}
		desc := false
		if len(parts) >= 2 && strings.ToLower(parts[1]) == "desc" {
			desc = true
		}
		return Stage{Kind: StageSort, SortField: parts[0], SortDesc: desc}, nil

	case "take", "head":
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return Stage{}, fmt.Errorf("take requires a number: %w", err)
		}
		return Stage{Kind: StageTake, Limit: n}, nil

	case "unique":
		return Stage{Kind: StageUnique}, nil

	case "group-by":
		if rest == "" {
			return Stage{}, errors.New("group-by requires a field name")
		}
		fields := parseCSV(rest)
		return Stage{Kind: StageGroupBy, Fields: fields}, nil

	case "count":
		return Stage{Kind: StageCount}, nil

	case "refs-out":
		return Stage{Kind: StageRefsOut}, nil

	case "refs-in":
		return Stage{Kind: StageRefsIn}, nil

	case "reachable":
		return Stage{Kind: StageReachable}, nil

	case "ancestors":
		return Stage{Kind: StageAncestors}, nil

	case "properties":
		return Stage{Kind: StageProperties}, nil

	case "union-members":
		return Stage{Kind: StageUnionMembers}, nil

	case "items":
		return Stage{Kind: StageItems}, nil

	case "ops":
		return Stage{Kind: StageOps}, nil

	case "schemas":
		return Stage{Kind: StageSchemas}, nil

	case "explain":
		return Stage{Kind: StageExplain}, nil

	case "fields":
		return Stage{Kind: StageFields}, nil

	case "sample":
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return Stage{}, fmt.Errorf("sample requires a number: %w", err)
		}
		return Stage{Kind: StageSample, Limit: n}, nil

	case "path":
		from, to := parseTwoArgs(rest)
		if from == "" || to == "" {
			return Stage{}, errors.New("path requires two schema names")
		}
		return Stage{Kind: StagePath, PathFrom: from, PathTo: to}, nil

	case "top":
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			return Stage{}, errors.New("top requires a number and a field name")
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return Stage{}, fmt.Errorf("top requires a number: %w", err)
		}
		return Stage{Kind: StageTop, Limit: n, SortField: parts[1]}, nil

	case "bottom":
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			return Stage{}, errors.New("bottom requires a number and a field name")
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return Stage{}, fmt.Errorf("bottom requires a number: %w", err)
		}
		return Stage{Kind: StageBottom, Limit: n, SortField: parts[1]}, nil

	case "format":
		f := strings.TrimSpace(rest)
		if f != "table" && f != "json" && f != "markdown" && f != "toon" {
			return Stage{}, fmt.Errorf("format must be table, json, markdown, or toon, got %q", f)
		}
		return Stage{Kind: StageFormat, Format: f}, nil

	default:
		return Stage{}, fmt.Errorf("unknown stage: %q", keyword)
	}
}

// --- Executor ---

func run(stages []Stage, g *graph.SchemaGraph) (*Result, error) {
	if len(stages) == 0 {
		return &Result{}, nil
	}

	// Check if explain stage is present
	for _, stage := range stages {
		if stage.Kind == StageExplain {
			return &Result{Explain: buildExplain(stages)}, nil
		}
	}

	// Execute source stage
	result, err := execSource(stages[0], g)
	if err != nil {
		return nil, err
	}

	// Execute remaining stages
	for _, stage := range stages[1:] {
		result, err = execStage(stage, result, g)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func execSource(stage Stage, g *graph.SchemaGraph) (*Result, error) {
	result := &Result{}
	switch stage.Source {
	case "schemas":
		for i := range g.Schemas {
			result.Rows = append(result.Rows, Row{Kind: SchemaResult, SchemaIdx: i})
		}
	case "schemas.components":
		for i, s := range g.Schemas {
			if s.IsComponent {
				result.Rows = append(result.Rows, Row{Kind: SchemaResult, SchemaIdx: i})
			}
		}
	case "schemas.inline":
		for i, s := range g.Schemas {
			if s.IsInline {
				result.Rows = append(result.Rows, Row{Kind: SchemaResult, SchemaIdx: i})
			}
		}
	case "operations":
		for i := range g.Operations {
			result.Rows = append(result.Rows, Row{Kind: OperationResult, OpIdx: i})
		}
	default:
		return nil, fmt.Errorf("unknown source: %q", stage.Source)
	}
	return result, nil
}

func execStage(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	switch stage.Kind {
	case StageWhere:
		return execWhere(stage, result, g)
	case StageSelect:
		result.Fields = stage.Fields
		return result, nil
	case StageSort:
		return execSort(stage, result, g)
	case StageTake:
		return execTake(stage, result)
	case StageUnique:
		return execUnique(result)
	case StageGroupBy:
		return execGroupBy(stage, result, g)
	case StageCount:
		return &Result{IsCount: true, Count: len(result.Rows)}, nil
	case StageRefsOut:
		return execTraversal(result, g, traverseRefsOut)
	case StageRefsIn:
		return execTraversal(result, g, traverseRefsIn)
	case StageReachable:
		return execTraversal(result, g, traverseReachable)
	case StageAncestors:
		return execTraversal(result, g, traverseAncestors)
	case StageProperties:
		return execTraversal(result, g, traverseProperties)
	case StageUnionMembers:
		return execTraversal(result, g, traverseUnionMembers)
	case StageItems:
		return execTraversal(result, g, traverseItems)
	case StageOps:
		return execSchemasToOps(result, g)
	case StageSchemas:
		return execOpsToSchemas(result, g)
	case StageFields:
		return execFields(result)
	case StageSample:
		return execSample(stage, result)
	case StagePath:
		return execPath(stage, g)
	case StageTop:
		// Expand to sort desc + take
		sorted, err := execSort(Stage{Kind: StageSort, SortField: stage.SortField, SortDesc: true}, result, g)
		if err != nil {
			return nil, err
		}
		return execTake(Stage{Kind: StageTake, Limit: stage.Limit}, sorted)
	case StageBottom:
		// Expand to sort asc + take
		sorted, err := execSort(Stage{Kind: StageSort, SortField: stage.SortField, SortDesc: false}, result, g)
		if err != nil {
			return nil, err
		}
		return execTake(Stage{Kind: StageTake, Limit: stage.Limit}, sorted)
	case StageFormat:
		result.FormatHint = stage.Format
		return result, nil
	default:
		return nil, fmt.Errorf("unimplemented stage kind: %d", stage.Kind)
	}
}

func execWhere(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	predicate, err := expr.Parse(stage.Expr)
	if err != nil {
		return nil, fmt.Errorf("where expression error: %w", err)
	}

	filtered := &Result{Fields: result.Fields}
	for _, row := range result.Rows {
		r := rowAdapter{row: row, g: g}
		val := predicate.Eval(r)
		if val.Kind == expr.KindBool && val.Bool {
			filtered.Rows = append(filtered.Rows, row)
		}
	}
	return filtered, nil
}

func execSort(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	sort.SliceStable(result.Rows, func(i, j int) bool {
		vi := fieldValue(result.Rows[i], stage.SortField, g)
		vj := fieldValue(result.Rows[j], stage.SortField, g)

		cmp := compareValues(vi, vj)
		if stage.SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
	return result, nil
}

func execTake(stage Stage, result *Result) (*Result, error) {
	if stage.Limit < len(result.Rows) {
		result.Rows = result.Rows[:stage.Limit]
	}
	return result, nil
}

func execUnique(result *Result) (*Result, error) {
	seen := make(map[string]bool)
	filtered := &Result{Fields: result.Fields}
	for _, row := range result.Rows {
		key := rowKey(row)
		if !seen[key] {
			seen[key] = true
			filtered.Rows = append(filtered.Rows, row)
		}
	}
	return filtered, nil
}

func execGroupBy(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	if len(stage.Fields) == 0 {
		return nil, errors.New("group-by requires at least one field")
	}
	field := stage.Fields[0]

	type group struct {
		count int
		names []string
	}
	groups := make(map[string]*group)
	var order []string

	for _, row := range result.Rows {
		v := fieldValue(row, field, g)
		key := valueToString(v)
		grp, exists := groups[key]
		if !exists {
			grp = &group{}
			groups[key] = grp
			order = append(order, key)
		}
		grp.count++
		nameV := fieldValue(row, "name", g)
		grp.names = append(grp.names, valueToString(nameV))
	}

	grouped := &Result{Fields: result.Fields}
	for _, key := range order {
		grp, ok := groups[key]
		if !ok {
			continue
		}
		grouped.Groups = append(grouped.Groups, GroupResult{
			Key:   key,
			Count: grp.count,
			Names: grp.names,
		})
	}
	return grouped, nil
}

// --- Traversal ---

type traversalFunc func(row Row, g *graph.SchemaGraph) []Row

func execTraversal(result *Result, g *graph.SchemaGraph, fn traversalFunc) (*Result, error) {
	out := &Result{Fields: result.Fields}
	seen := make(map[string]bool)
	for _, row := range result.Rows {
		for _, newRow := range fn(row, g) {
			key := rowKey(newRow)
			if !seen[key] {
				seen[key] = true
				out.Rows = append(out.Rows, newRow)
			}
		}
	}
	return out, nil
}

func traverseRefsOut(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		result = append(result, Row{Kind: SchemaResult, SchemaIdx: int(edge.To)})
	}
	return result
}

func traverseRefsIn(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	var result []Row
	for _, edge := range g.InEdges(graph.NodeID(row.SchemaIdx)) {
		result = append(result, Row{Kind: SchemaResult, SchemaIdx: int(edge.From)})
	}
	return result
}

func traverseReachable(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	ids := g.Reachable(graph.NodeID(row.SchemaIdx))
	result := make([]Row, len(ids))
	for i, id := range ids {
		result[i] = Row{Kind: SchemaResult, SchemaIdx: int(id)}
	}
	return result
}

func traverseAncestors(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	ids := g.Ancestors(graph.NodeID(row.SchemaIdx))
	result := make([]Row, len(ids))
	for i, id := range ids {
		result[i] = Row{Kind: SchemaResult, SchemaIdx: int(id)}
	}
	return result
}

func traverseProperties(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeProperty {
			result = append(result, Row{Kind: SchemaResult, SchemaIdx: int(edge.To)})
		}
	}
	return result
}

func traverseUnionMembers(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeAllOf || edge.Kind == graph.EdgeOneOf || edge.Kind == graph.EdgeAnyOf {
			// Follow through $ref nodes transparently
			target := resolveRefTarget(int(edge.To), g)
			result = append(result, Row{Kind: SchemaResult, SchemaIdx: target})
		}
	}
	return result
}

func traverseItems(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeItems {
			result = append(result, Row{Kind: SchemaResult, SchemaIdx: int(edge.To)})
		}
	}
	return result
}

// resolveRefTarget follows EdgeRef edges to get the actual target node.
// If the node at idx is a $ref wrapper, returns the target component's index.
// Otherwise returns idx unchanged.
func resolveRefTarget(idx int, g *graph.SchemaGraph) int {
	if idx < 0 || idx >= len(g.Schemas) {
		return idx
	}
	node := &g.Schemas[idx]
	if !node.HasRef {
		return idx
	}
	// Follow EdgeRef edges
	for _, edge := range g.OutEdges(graph.NodeID(idx)) {
		if edge.Kind == graph.EdgeRef {
			return int(edge.To)
		}
	}
	return idx
}

func execSchemasToOps(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	seen := make(map[int]bool)
	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		opIDs := g.SchemaOperations(graph.NodeID(row.SchemaIdx))
		for _, opID := range opIDs {
			idx := int(opID)
			if !seen[idx] {
				seen[idx] = true
				out.Rows = append(out.Rows, Row{Kind: OperationResult, OpIdx: idx})
			}
		}
	}
	return out, nil
}

func execOpsToSchemas(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	seen := make(map[int]bool)
	for _, row := range result.Rows {
		if row.Kind != OperationResult {
			continue
		}
		schemaIDs := g.OperationSchemas(graph.NodeID(row.OpIdx))
		for _, sid := range schemaIDs {
			idx := int(sid)
			if !seen[idx] {
				seen[idx] = true
				out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: idx})
			}
		}
	}
	return out, nil
}

// --- Field access ---

type rowAdapter struct {
	row Row
	g   *graph.SchemaGraph
}

func (r rowAdapter) Field(name string) expr.Value {
	return fieldValue(r.row, name, r.g)
}

// FieldValuePublic returns the value of a named field for the given row.
// Exported for testing and external consumers.
func FieldValuePublic(row Row, name string, g *graph.SchemaGraph) expr.Value {
	return fieldValue(row, name, g)
}

func fieldValue(row Row, name string, g *graph.SchemaGraph) expr.Value {
	switch row.Kind {
	case SchemaResult:
		if row.SchemaIdx < 0 || row.SchemaIdx >= len(g.Schemas) {
			return expr.NullVal()
		}
		s := &g.Schemas[row.SchemaIdx]
		switch name {
		case "name":
			return expr.StringVal(s.Name)
		case "type":
			return expr.StringVal(s.Type)
		case "depth":
			return expr.IntVal(s.Depth)
		case "in_degree":
			return expr.IntVal(s.InDegree)
		case "out_degree":
			return expr.IntVal(s.OutDegree)
		case "union_width":
			return expr.IntVal(s.UnionWidth)
		case "property_count":
			return expr.IntVal(s.PropertyCount)
		case "is_component":
			return expr.BoolVal(s.IsComponent)
		case "is_inline":
			return expr.BoolVal(s.IsInline)
		case "is_circular":
			return expr.BoolVal(s.IsCircular)
		case "has_ref":
			return expr.BoolVal(s.HasRef)
		case "hash":
			return expr.StringVal(s.Hash)
		case "path":
			return expr.StringVal(s.Path)
		}
	case OperationResult:
		if row.OpIdx < 0 || row.OpIdx >= len(g.Operations) {
			return expr.NullVal()
		}
		o := &g.Operations[row.OpIdx]
		switch name {
		case "name":
			return expr.StringVal(o.Name)
		case "method":
			return expr.StringVal(o.Method)
		case "path":
			return expr.StringVal(o.Path)
		case "operation_id":
			return expr.StringVal(o.OperationID)
		case "schema_count":
			return expr.IntVal(o.SchemaCount)
		case "component_count":
			return expr.IntVal(o.ComponentCount)
		case "tag":
			if o.Operation != nil && len(o.Operation.Tags) > 0 {
				return expr.StringVal(o.Operation.Tags[0])
			}
			return expr.StringVal("")
		case "parameter_count":
			if o.Operation != nil {
				return expr.IntVal(len(o.Operation.Parameters))
			}
			return expr.IntVal(0)
		case "deprecated":
			if o.Operation != nil {
				return expr.BoolVal(o.Operation.Deprecated != nil && *o.Operation.Deprecated)
			}
			return expr.BoolVal(false)
		case "description":
			if o.Operation != nil {
				return expr.StringVal(o.Operation.GetDescription())
			}
			return expr.StringVal("")
		case "summary":
			if o.Operation != nil {
				return expr.StringVal(o.Operation.GetSummary())
			}
			return expr.StringVal("")
		}
	}
	return expr.NullVal()
}

func compareValues(a, b expr.Value) int {
	if a.Kind == expr.KindInt && b.Kind == expr.KindInt {
		if a.Int < b.Int {
			return -1
		}
		if a.Int > b.Int {
			return 1
		}
		return 0
	}
	sa := valueToString(a)
	sb := valueToString(b)
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}

func valueToString(v expr.Value) string {
	switch v.Kind {
	case expr.KindString:
		return v.Str
	case expr.KindInt:
		return strconv.Itoa(v.Int)
	case expr.KindBool:
		return strconv.FormatBool(v.Bool)
	default:
		return ""
	}
}

func rowKey(row Row) string {
	if row.Kind == SchemaResult {
		return "s:" + strconv.Itoa(row.SchemaIdx)
	}
	return "o:" + strconv.Itoa(row.OpIdx)
}

// --- Explain ---

func buildExplain(stages []Stage) string {
	var sb strings.Builder
	for i, stage := range stages {
		if stage.Kind == StageExplain {
			continue
		}
		if i == 0 {
			fmt.Fprintf(&sb, "Source: %s\n", stage.Source)
		} else {
			desc := describeStage(stage)
			fmt.Fprintf(&sb, "  → %s\n", desc)
		}
	}
	return sb.String()
}

func describeStage(stage Stage) string {
	switch stage.Kind {
	case StageWhere:
		return "Filter: where " + stage.Expr
	case StageSelect:
		return "Project: select " + strings.Join(stage.Fields, ", ")
	case StageSort:
		dir := "ascending"
		if stage.SortDesc {
			dir = "descending"
		}
		return "Sort: " + stage.SortField + " " + dir
	case StageTake:
		return "Limit: take " + strconv.Itoa(stage.Limit)
	case StageUnique:
		return "Unique: deduplicate rows"
	case StageGroupBy:
		return "Group: group-by " + strings.Join(stage.Fields, ", ")
	case StageCount:
		return "Count: count rows"
	case StageRefsOut:
		return "Traverse: outgoing references"
	case StageRefsIn:
		return "Traverse: incoming references"
	case StageReachable:
		return "Traverse: all reachable nodes"
	case StageAncestors:
		return "Traverse: all ancestor nodes"
	case StageProperties:
		return "Traverse: property children"
	case StageUnionMembers:
		return "Traverse: union members"
	case StageItems:
		return "Traverse: array items"
	case StageOps:
		return "Navigate: schemas to operations"
	case StageSchemas:
		return "Navigate: operations to schemas"
	case StageFields:
		return "Terminal: list available fields"
	case StageSample:
		return "Sample: random " + strconv.Itoa(stage.Limit) + " rows"
	case StagePath:
		return "Path: shortest path from " + stage.PathFrom + " to " + stage.PathTo
	case StageTop:
		return "Top: " + strconv.Itoa(stage.Limit) + " by " + stage.SortField + " descending"
	case StageBottom:
		return "Bottom: " + strconv.Itoa(stage.Limit) + " by " + stage.SortField + " ascending"
	case StageFormat:
		return "Format: " + stage.Format
	default:
		return "Unknown stage"
	}
}

// --- Fields ---

func execFields(result *Result) (*Result, error) {
	var sb strings.Builder
	kind := SchemaResult
	if len(result.Rows) > 0 {
		kind = result.Rows[0].Kind
	}

	if kind == SchemaResult {
		sb.WriteString("Field             Type\n")
		sb.WriteString("-----------       ------\n")
		fields := []struct{ name, typ string }{
			{"name", "string"},
			{"type", "string"},
			{"depth", "int"},
			{"in_degree", "int"},
			{"out_degree", "int"},
			{"union_width", "int"},
			{"property_count", "int"},
			{"is_component", "bool"},
			{"is_inline", "bool"},
			{"is_circular", "bool"},
			{"has_ref", "bool"},
			{"hash", "string"},
			{"path", "string"},
		}
		for _, f := range fields {
			fmt.Fprintf(&sb, "%-17s %s\n", f.name, f.typ)
		}
	} else {
		sb.WriteString("Field             Type\n")
		sb.WriteString("-----------       ------\n")
		fields := []struct{ name, typ string }{
			{"name", "string"},
			{"method", "string"},
			{"path", "string"},
			{"operation_id", "string"},
			{"schema_count", "int"},
			{"component_count", "int"},
			{"tag", "string"},
			{"parameter_count", "int"},
			{"deprecated", "bool"},
			{"description", "string"},
			{"summary", "string"},
		}
		for _, f := range fields {
			fmt.Fprintf(&sb, "%-17s %s\n", f.name, f.typ)
		}
	}

	return &Result{Explain: sb.String()}, nil
}

// --- Sample ---

func execSample(stage Stage, result *Result) (*Result, error) {
	if stage.Limit >= len(result.Rows) {
		return result, nil
	}

	// Deterministic shuffle: sort by hash of row key, then take first n
	type keyed struct {
		hash string
		row  Row
	}
	items := make([]keyed, len(result.Rows))
	for i, row := range result.Rows {
		h := sha256.Sum256([]byte(rowKey(row)))
		items[i] = keyed{hash: hex.EncodeToString(h[:]), row: row}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].hash < items[j].hash
	})

	out := &Result{Fields: result.Fields}
	for i := 0; i < stage.Limit && i < len(items); i++ {
		out.Rows = append(out.Rows, items[i].row)
	}
	return out, nil
}

// --- Path ---

func execPath(stage Stage, g *graph.SchemaGraph) (*Result, error) {
	fromNode, ok := g.SchemaByName(stage.PathFrom)
	if !ok {
		return nil, fmt.Errorf("schema %q not found", stage.PathFrom)
	}
	toNode, ok := g.SchemaByName(stage.PathTo)
	if !ok {
		return nil, fmt.Errorf("schema %q not found", stage.PathTo)
	}

	path := g.ShortestPath(fromNode.ID, toNode.ID)
	out := &Result{}
	for _, id := range path {
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: int(id)})
	}
	return out, nil
}

// --- Arg parsing helpers ---

func parseTwoArgs(s string) (string, string) {
	s = strings.TrimSpace(s)
	var args []string
	for len(s) > 0 {
		if s[0] == '"' {
			// Quoted arg
			end := strings.Index(s[1:], "\"")
			if end < 0 {
				args = append(args, s[1:])
				break
			}
			args = append(args, s[1:end+1])
			s = strings.TrimSpace(s[end+2:])
		} else {
			idx := strings.IndexAny(s, " \t")
			if idx < 0 {
				args = append(args, s)
				break
			}
			args = append(args, s[:idx])
			s = strings.TrimSpace(s[idx+1:])
		}
		if len(args) == 2 {
			break
		}
	}
	if len(args) < 2 {
		if len(args) == 1 {
			return args[0], ""
		}
		return "", ""
	}
	return args[0], args[1]
}

// --- Formatting ---

// FormatTable formats a result as a simple table string.
func FormatTable(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	if len(result.Groups) > 0 {
		return formatGroups(result)
	}

	if len(result.Rows) == 0 {
		return "(empty)"
	}

	fields := result.Fields
	if len(fields) == 0 {
		if result.Rows[0].Kind == SchemaResult {
			fields = []string{"name", "type", "depth", "in_degree", "out_degree"}
		} else {
			fields = []string{"name", "method", "path", "schema_count"}
		}
	}

	// Build header
	widths := make([]int, len(fields))
	for i, f := range fields {
		widths[i] = len(f)
	}

	// Collect rows
	var tableRows [][]string
	for _, row := range result.Rows {
		var cols []string
		for i, f := range fields {
			v := valueToString(fieldValue(row, f, g))
			cols = append(cols, v)
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
		tableRows = append(tableRows, cols)
	}

	// Format
	var sb strings.Builder
	// Header
	for i, f := range fields {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(padRight(f, widths[i]))
	}
	sb.WriteString("\n")
	// Separator
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("-", w))
	}
	sb.WriteString("\n")
	// Data
	for _, row := range tableRows {
		for i, col := range row {
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(padRight(col, widths[i]))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatJSON formats a result as JSON.
func FormatJSON(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	if len(result.Groups) > 0 {
		return formatGroupsJSON(result)
	}

	if len(result.Rows) == 0 {
		return "[]"
	}

	fields := result.Fields
	if len(fields) == 0 {
		if result.Rows[0].Kind == SchemaResult {
			fields = []string{"name", "type", "depth", "in_degree", "out_degree"}
		} else {
			fields = []string{"name", "method", "path", "schema_count"}
		}
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, row := range result.Rows {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString("  {")
		for j, f := range fields {
			if j > 0 {
				sb.WriteString(", ")
			}
			v := fieldValue(row, f, g)
			fmt.Fprintf(&sb, "%q: %s", f, jsonValue(v))
		}
		sb.WriteString("}")
	}
	sb.WriteString("\n]")
	return sb.String()
}

// FormatMarkdown formats a result as a markdown table.
func FormatMarkdown(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	if len(result.Groups) > 0 {
		var sb strings.Builder
		sb.WriteString("| Key | Count |\n")
		sb.WriteString("| --- | --- |\n")
		for _, grp := range result.Groups {
			fmt.Fprintf(&sb, "| %s | %d |\n", grp.Key, grp.Count)
		}
		return sb.String()
	}

	if len(result.Rows) == 0 {
		return "(empty)"
	}

	fields := result.Fields
	if len(fields) == 0 {
		if result.Rows[0].Kind == SchemaResult {
			fields = []string{"name", "type", "depth", "in_degree", "out_degree"}
		} else {
			fields = []string{"name", "method", "path", "schema_count"}
		}
	}

	var sb strings.Builder
	// Header
	sb.WriteString("| ")
	sb.WriteString(strings.Join(fields, " | "))
	sb.WriteString(" |\n")
	// Separator
	sb.WriteString("|")
	for range fields {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")
	// Rows
	for _, row := range result.Rows {
		sb.WriteString("| ")
		for i, f := range fields {
			if i > 0 {
				sb.WriteString(" | ")
			}
			v := valueToString(fieldValue(row, f, g))
			sb.WriteString(v)
		}
		sb.WriteString(" |\n")
	}

	return sb.String()
}

// FormatToon formats a result in the TOON (Token-Oriented Object Notation) format.
// TOON uses tabular array syntax for uniform rows: header[N]{field1,field2,...}:
// followed by comma-delimited data rows. See https://github.com/toon-format/toon
func FormatToon(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return "count: " + strconv.Itoa(result.Count) + "\n"
	}

	if len(result.Groups) > 0 {
		return formatGroupsToon(result)
	}

	if len(result.Rows) == 0 {
		return "results[0]:\n"
	}

	fields := result.Fields
	if len(fields) == 0 {
		if result.Rows[0].Kind == SchemaResult {
			fields = []string{"name", "type", "depth", "in_degree", "out_degree"}
		} else {
			fields = []string{"name", "method", "path", "schema_count"}
		}
	}

	var sb strings.Builder

	// Header: results[N]{field1,field2,...}:
	fmt.Fprintf(&sb, "results[%d]{%s}:\n", len(result.Rows), strings.Join(fields, ","))

	// Data rows: comma-separated values, indented by one space
	for _, row := range result.Rows {
		sb.WriteByte(' ')
		for i, f := range fields {
			if i > 0 {
				sb.WriteByte(',')
			}
			v := fieldValue(row, f, g)
			sb.WriteString(toonValue(v))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func formatGroupsToon(result *Result) string {
	var sb strings.Builder

	// Groups as tabular array
	fmt.Fprintf(&sb, "groups[%d]{key,count,names}:\n", len(result.Groups))
	for _, grp := range result.Groups {
		names := strings.Join(grp.Names, ";")
		fmt.Fprintf(&sb, " %s,%d,%s\n", toonEscape(grp.Key), grp.Count, toonEscape(names))
	}
	return sb.String()
}

// toonValue encodes an expr.Value for TOON format.
func toonValue(v expr.Value) string {
	switch v.Kind {
	case expr.KindString:
		return toonEscape(v.Str)
	case expr.KindInt:
		return strconv.Itoa(v.Int)
	case expr.KindBool:
		return strconv.FormatBool(v.Bool)
	default:
		return "null"
	}
}

// toonEscape quotes a string if it needs escaping for TOON format.
// A string must be quoted if it: is empty, contains comma/colon/quote/backslash/
// brackets/braces/control chars, has leading/trailing whitespace, or matches
// true/false/null or a numeric pattern.
func toonEscape(s string) string {
	if s == "" {
		return `""`
	}
	if s == "true" || s == "false" || s == "null" {
		return `"` + s + `"`
	}
	// Check if it looks numeric
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return `"` + s + `"`
	}
	needsQuote := false
	for _, ch := range s {
		if ch == ',' || ch == ':' || ch == '"' || ch == '\\' ||
			ch == '[' || ch == ']' || ch == '{' || ch == '}' ||
			ch == '\n' || ch == '\r' || ch == '\t' ||
			ch < 0x20 {
			needsQuote = true
			break
		}
	}
	if s[0] == ' ' || s[len(s)-1] == ' ' {
		needsQuote = true
	}
	if !needsQuote {
		return s
	}
	// Quote with escaping
	var sb strings.Builder
	sb.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(ch)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func jsonValue(v expr.Value) string {
	switch v.Kind {
	case expr.KindString:
		return fmt.Sprintf("%q", v.Str)
	case expr.KindInt:
		return strconv.Itoa(v.Int)
	case expr.KindBool:
		return strconv.FormatBool(v.Bool)
	default:
		return "null"
	}
}

func formatGroups(result *Result) string {
	var sb strings.Builder
	for _, g := range result.Groups {
		fmt.Fprintf(&sb, "%s: count=%d", g.Key, g.Count)
		if len(g.Names) > 0 {
			names := slices.Clone(g.Names)
			if len(names) > 5 {
				names = names[:5]
				names = append(names, "...")
			}
			fmt.Fprintf(&sb, " names=[%s]", strings.Join(names, ", "))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatGroupsJSON(result *Result) string {
	var sb strings.Builder
	sb.WriteString("[\n")
	for i, g := range result.Groups {
		if i > 0 {
			sb.WriteString(",\n")
		}
		fmt.Fprintf(&sb, `  {"key": %q, "count": %d, "names": [`, g.Key, g.Count)
		for j, n := range g.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%q", n)
		}
		sb.WriteString("]}")
	}
	sb.WriteString("\n]")
	return sb.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// --- Pipeline splitting ---

func splitPipeline(input string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '"':
			inQuote = !inQuote
			current.WriteByte(ch)
		case ch == '|' && !inQuote:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func splitFirst(s string) (string, string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
