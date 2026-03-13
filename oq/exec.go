package oq

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/oq/expr"
)

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
	case StageConnected:
		return execConnected(result, g)
	case StageBlastRadius:
		return execBlastRadius(result, g)
	case StageNeighbors:
		return execNeighbors(stage, result, g)
	case StageOrphans:
		return execOrphans(result, g)
	case StageLeaves:
		return execLeaves(result, g)
	case StageCycles:
		return execCycles(result, g)
	case StageClusters:
		return execClusters(result, g)
	case StageTagBoundary:
		return execTagBoundary(result, g)
	case StageSharedRefs:
		return execSharedRefs(result, g)
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
	sorted := &Result{
		Fields:     result.Fields,
		FormatHint: result.FormatHint,
		Rows:       slices.Clone(result.Rows),
	}
	sort.SliceStable(sorted.Rows, func(i, j int) bool {
		vi := fieldValue(sorted.Rows[i], stage.SortField, g)
		vj := fieldValue(sorted.Rows[j], stage.SortField, g)

		cmp := compareValues(vi, vj)
		if stage.SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
	return sorted, nil
}

func execTake(stage Stage, result *Result) (*Result, error) {
	rows := result.Rows
	if stage.Limit < len(rows) {
		rows = rows[:stage.Limit]
	}
	return &Result{
		Fields:     result.Fields,
		FormatHint: result.FormatHint,
		Rows:       slices.Clone(rows),
	}, nil
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
			key := edgeRowKey(newRow)
			if !seen[key] {
				seen[key] = true
				out.Rows = append(out.Rows, newRow)
			}
		}
	}
	return out, nil
}

func edgeRowKey(row Row) string {
	base := rowKey(row)
	if row.EdgeKind == "" {
		return base
	}
	return base + "|" + row.EdgeFrom + "|" + row.EdgeKind + "|" + row.EdgeLabel
}

func traverseRefsOut(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	fromName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		result = append(result, Row{
			Kind:      SchemaResult,
			SchemaIdx: int(edge.To),
			EdgeKind:  edgeKindString(edge.Kind),
			EdgeLabel: edge.Label,
			EdgeFrom:  fromName,
		})
	}
	return result
}

func traverseRefsIn(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	toName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.InEdges(graph.NodeID(row.SchemaIdx)) {
		result = append(result, Row{
			Kind:      SchemaResult,
			SchemaIdx: int(edge.From),
			EdgeKind:  edgeKindString(edge.Kind),
			EdgeLabel: edge.Label,
			EdgeFrom:  toName,
		})
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
	fromName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeProperty {
			result = append(result, Row{
				Kind:      SchemaResult,
				SchemaIdx: int(edge.To),
				EdgeKind:  edgeKindString(edge.Kind),
				EdgeLabel: edge.Label,
				EdgeFrom:  fromName,
			})
		}
	}
	return result
}

func traverseUnionMembers(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	fromName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeAllOf || edge.Kind == graph.EdgeOneOf || edge.Kind == graph.EdgeAnyOf {
			// Follow through $ref nodes transparently
			target := resolveRefTarget(int(edge.To), g)
			result = append(result, Row{
				Kind:      SchemaResult,
				SchemaIdx: target,
				EdgeKind:  edgeKindString(edge.Kind),
				EdgeLabel: edge.Label,
				EdgeFrom:  fromName,
			})
		}
	}
	return result
}

func traverseItems(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	fromName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeItems {
			result = append(result, Row{
				Kind:      SchemaResult,
				SchemaIdx: int(edge.To),
				EdgeKind:  edgeKindString(edge.Kind),
				EdgeLabel: edge.Label,
				EdgeFrom:  fromName,
			})
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

func execConnected(result *Result, g *graph.SchemaGraph) (*Result, error) {
	var schemaSeeds, opSeeds []graph.NodeID
	for _, row := range result.Rows {
		switch row.Kind {
		case SchemaResult:
			schemaSeeds = append(schemaSeeds, graph.NodeID(row.SchemaIdx))
		case OperationResult:
			opSeeds = append(opSeeds, graph.NodeID(row.OpIdx))
		}
	}

	schemas, ops := g.ConnectedComponent(schemaSeeds, opSeeds)

	out := &Result{Fields: result.Fields}
	for _, id := range schemas {
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: int(id)})
	}
	for _, id := range ops {
		out.Rows = append(out.Rows, Row{Kind: OperationResult, OpIdx: int(id)})
	}
	return out, nil
}

func execBlastRadius(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	seenSchemas := make(map[int]bool)
	seenOps := make(map[int]bool)

	// Collect seed schemas
	var seeds []graph.NodeID
	for _, row := range result.Rows {
		if row.Kind == SchemaResult {
			seeds = append(seeds, graph.NodeID(row.SchemaIdx))
			seenSchemas[row.SchemaIdx] = true
		}
	}

	// Find all ancestors (schemas that depend on the seeds)
	for _, seed := range seeds {
		for _, aid := range g.Ancestors(seed) {
			seenSchemas[int(aid)] = true
		}
	}

	// Collect and sort schema indices for deterministic output
	schemaIndices := make([]int, 0, len(seenSchemas))
	for idx := range seenSchemas {
		schemaIndices = append(schemaIndices, idx)
	}
	sort.Ints(schemaIndices)

	// Add schema rows
	for _, idx := range schemaIndices {
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: idx})
	}

	// Find all operations that reference any affected schema
	for _, idx := range schemaIndices {
		for _, opID := range g.SchemaOperations(graph.NodeID(idx)) {
			if !seenOps[int(opID)] {
				seenOps[int(opID)] = true
				out.Rows = append(out.Rows, Row{Kind: OperationResult, OpIdx: int(opID)})
			}
		}
	}

	return out, nil
}

func execNeighbors(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	seen := make(map[int]bool)

	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		// Include seed
		if !seen[row.SchemaIdx] {
			seen[row.SchemaIdx] = true
			out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: row.SchemaIdx})
		}
		for _, id := range g.Neighbors(graph.NodeID(row.SchemaIdx), stage.Limit) {
			if !seen[int(id)] {
				seen[int(id)] = true
				out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: int(id)})
			}
		}
	}

	return out, nil
}

func execOrphans(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		s := &g.Schemas[row.SchemaIdx]
		if s.InDegree == 0 && g.SchemaOpCount(graph.NodeID(row.SchemaIdx)) == 0 {
			out.Rows = append(out.Rows, row)
		}
	}
	return out, nil
}

func execLeaves(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		if g.Schemas[row.SchemaIdx].OutDegree == 0 {
			out.Rows = append(out.Rows, row)
		}
	}
	return out, nil
}

func execCycles(result *Result, g *graph.SchemaGraph) (*Result, error) {
	sccs := g.StronglyConnectedComponents()

	// Filter SCCs to only include nodes present in the current result
	resultNodes := make(map[int]bool)
	for _, row := range result.Rows {
		if row.Kind == SchemaResult {
			resultNodes[row.SchemaIdx] = true
		}
	}

	out := &Result{Fields: result.Fields}
	for i, scc := range sccs {
		hasMatch := false
		for _, id := range scc {
			if resultNodes[int(id)] {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			continue
		}
		var names []string
		for _, id := range scc {
			if int(id) < len(g.Schemas) {
				names = append(names, g.Schemas[id].Name)
			}
		}
		out.Groups = append(out.Groups, GroupResult{
			Key:   "cycle-" + strconv.Itoa(i+1),
			Count: len(scc),
			Names: names,
		})
	}

	return out, nil
}

func execClusters(result *Result, g *graph.SchemaGraph) (*Result, error) {
	resultNodes := make(map[int]bool)
	for _, row := range result.Rows {
		if row.Kind == SchemaResult {
			resultNodes[row.SchemaIdx] = true
		}
	}

	// Sort node indices for deterministic iteration
	sortedNodes := make([]int, 0, len(resultNodes))
	for idx := range resultNodes {
		sortedNodes = append(sortedNodes, idx)
	}
	sort.Ints(sortedNodes)

	// BFS to find connected components. Follow ALL graph edges (including
	// through intermediary nodes like $ref wrappers) but only collect
	// nodes that are in the result set.
	assigned := make(map[int]bool) // result nodes already assigned to a cluster
	out := &Result{Fields: result.Fields}
	clusterNum := 0

	for _, idx := range sortedNodes {
		if assigned[idx] {
			continue
		}
		clusterNum++
		var component []int

		// BFS through the full graph
		visited := make(map[int]bool)
		queue := []int{idx}
		visited[idx] = true

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			if resultNodes[cur] && !assigned[cur] {
				assigned[cur] = true
				component = append(component, cur)
			}

			for _, edge := range g.OutEdges(graph.NodeID(cur)) {
				to := int(edge.To)
				if !visited[to] {
					visited[to] = true
					queue = append(queue, to)
				}
			}
			for _, edge := range g.InEdges(graph.NodeID(cur)) {
				from := int(edge.From)
				if !visited[from] {
					visited[from] = true
					queue = append(queue, from)
				}
			}
		}

		var names []string
		for _, id := range component {
			if id < len(g.Schemas) {
				names = append(names, g.Schemas[id].Name)
			}
		}
		if len(component) > 0 {
			out.Groups = append(out.Groups, GroupResult{
				Key:   "cluster-" + strconv.Itoa(clusterNum),
				Count: len(component),
				Names: names,
			})
		}
	}

	return out, nil
}

func execTagBoundary(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := &Result{Fields: result.Fields}
	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		if schemaTagCount(row.SchemaIdx, g) > 1 {
			out.Rows = append(out.Rows, row)
		}
	}
	return out, nil
}

func schemaTagCount(schemaIdx int, g *graph.SchemaGraph) int {
	tags := make(map[string]bool)
	for _, opID := range g.SchemaOperations(graph.NodeID(schemaIdx)) {
		if int(opID) < len(g.Operations) {
			op := &g.Operations[opID]
			if op.Operation != nil {
				for _, tag := range op.Operation.Tags {
					tags[tag] = true
				}
			}
		}
	}
	return len(tags)
}

func execSharedRefs(result *Result, g *graph.SchemaGraph) (*Result, error) {
	var ops []graph.NodeID
	for _, row := range result.Rows {
		if row.Kind == OperationResult {
			ops = append(ops, graph.NodeID(row.OpIdx))
		}
	}

	if len(ops) == 0 {
		return &Result{Fields: result.Fields}, nil
	}

	// Start with first operation's schemas
	intersection := make(map[graph.NodeID]bool)
	for _, sid := range g.OperationSchemas(ops[0]) {
		intersection[sid] = true
	}

	// Intersect with each subsequent operation
	for _, opID := range ops[1:] {
		opSchemas := make(map[graph.NodeID]bool)
		for _, sid := range g.OperationSchemas(opID) {
			opSchemas[sid] = true
		}
		for sid := range intersection {
			if !opSchemas[sid] {
				delete(intersection, sid)
			}
		}
	}

	// Sort for deterministic output
	sortedIDs := make([]int, 0, len(intersection))
	for sid := range intersection {
		sortedIDs = append(sortedIDs, int(sid))
	}
	sort.Ints(sortedIDs)

	out := &Result{Fields: result.Fields}
	for _, sid := range sortedIDs {
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: sid})
	}
	return out, nil
}

// --- Edge annotation helpers ---

func schemaName(idx int, g *graph.SchemaGraph) string {
	if idx >= 0 && idx < len(g.Schemas) {
		return g.Schemas[idx].Name
	}
	return ""
}

func edgeKindString(k graph.EdgeKind) string {
	switch k {
	case graph.EdgeProperty:
		return "property"
	case graph.EdgeItems:
		return "items"
	case graph.EdgeAllOf:
		return "allOf"
	case graph.EdgeOneOf:
		return "oneOf"
	case graph.EdgeAnyOf:
		return "anyOf"
	case graph.EdgeAdditionalProps:
		return "additionalProperties"
	case graph.EdgeNot:
		return "not"
	case graph.EdgeIf:
		return "if"
	case graph.EdgeThen:
		return "then"
	case graph.EdgeElse:
		return "else"
	case graph.EdgeContains:
		return "contains"
	case graph.EdgePrefixItems:
		return "prefixItems"
	case graph.EdgeDependentSchema:
		return "dependentSchema"
	case graph.EdgePatternProperty:
		return "patternProperty"
	case graph.EdgePropertyNames:
		return "propertyNames"
	case graph.EdgeUnevaluatedItems:
		return "unevaluatedItems"
	case graph.EdgeUnevaluatedProps:
		return "unevaluatedProperties"
	case graph.EdgeRef:
		return "ref"
	default:
		return "unknown"
	}
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
	case StageConnected:
		return "Traverse: full connected component (schemas + operations)"
	case StageBlastRadius:
		return "Traverse: blast radius (ancestors + affected operations)"
	case StageNeighbors:
		return "Traverse: bidirectional neighbors within " + strconv.Itoa(stage.Limit) + " hops"
	case StageOrphans:
		return "Filter: schemas with no incoming refs and no operation usage"
	case StageLeaves:
		return "Filter: schemas with no outgoing refs (leaf nodes)"
	case StageCycles:
		return "Analyze: strongly connected components (actual cycles)"
	case StageClusters:
		return "Analyze: weakly connected component clusters"
	case StageTagBoundary:
		return "Filter: schemas used by operations across multiple tags"
	case StageSharedRefs:
		return "Analyze: schemas shared by all operations in result"
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
			{"op_count", "int"},
			{"tag_count", "int"},
			{"edge_kind", "string"},
			{"edge_label", "string"},
			{"edge_from", "string"},
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
			{"edge_kind", "string"},
			{"edge_label", "string"},
			{"edge_from", "string"},
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

	// Deterministic shuffle using Fisher-Yates with a fixed seed derived from row count.
	rows := append([]Row{}, result.Rows...)
	rng := rand.New(rand.NewPCG(uint64(len(rows)), 0)) //nolint:gosec // deterministic seed is intentional
	rng.Shuffle(len(rows), func(i, j int) {
		rows[i], rows[j] = rows[j], rows[i]
	})

	out := &Result{Fields: result.Fields}
	out.Rows = rows[:stage.Limit]
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
