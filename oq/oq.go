// Package oq implements a pipeline query language for OpenAPI schema graphs.
//
// Queries are written as pipeline expressions like:
//
//	schemas.components | where depth > 5 | sort depth desc | take 10 | select name, depth
package oq

import (
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
	Rows    []Row
	Fields  []string // projected fields (empty = all)
	IsCount bool
	Count   int
	Groups  []GroupResult
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
)

// Stage represents a single stage in the query pipeline.
type Stage struct {
	Kind      StageKind
	Source    string   // for StageSource
	Expr      string   // for StageWhere
	Fields    []string // for StageSelect, StageGroupBy
	SortField string   // for StageSort
	SortDesc  bool     // for StageSort
	Limit     int      // for StageTake
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

	case "take":
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

	default:
		return Stage{}, fmt.Errorf("unknown stage: %q", keyword)
	}
}

// --- Executor ---

func run(stages []Stage, g *graph.SchemaGraph) (*Result, error) {
	if len(stages) == 0 {
		return &Result{}, nil
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
		grp := groups[key]
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

// --- Formatting ---

// FormatTable formats a result as a simple table string.
func FormatTable(result *Result, g *graph.SchemaGraph) string {
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
			sb.WriteString(fmt.Sprintf("%q: %s", f, jsonValue(v)))
		}
		sb.WriteString("}")
	}
	sb.WriteString("\n]")
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
		sb.WriteString(fmt.Sprintf("%s: count=%d", g.Key, g.Count))
		if len(g.Names) > 0 {
			names := slices.Clone(g.Names)
			if len(names) > 5 {
				names = names[:5]
				names = append(names, "...")
			}
			sb.WriteString(fmt.Sprintf(" names=[%s]", strings.Join(names, ", ")))
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
		sb.WriteString(fmt.Sprintf(`  {"key": %q, "count": %d, "names": [`, g.Key, g.Count))
		for j, n := range g.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", n))
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
