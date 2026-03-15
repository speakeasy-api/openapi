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
	oas3 "github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq/expr"
)

// deriveResult creates a new Result that inherits metadata (Fields, FormatHint, EmitYAML) from a parent.
func deriveResult(parent *Result) *Result {
	return &Result{
		Fields:     parent.Fields,
		FormatHint: parent.FormatHint,
		EmitYAML:   parent.EmitYAML,
	}
}

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

	// Thread env through stages for let bindings
	env := map[string]expr.Value{}

	// Execute remaining stages
	for _, stage := range stages[1:] {
		result, env, err = execStageWithEnv(stage, result, g, env)
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
	case "operations":
		for i := range g.Operations {
			result.Rows = append(result.Rows, Row{Kind: OperationResult, OpIdx: i})
		}
	case "components.schemas":
		for i, s := range g.Schemas {
			if s.IsComponent {
				result.Rows = append(result.Rows, Row{Kind: SchemaResult, SchemaIdx: i})
			}
		}
	case "components.parameters":
		result.Rows = append(result.Rows, componentRows(g, componentParameters)...)
	case "components.responses":
		result.Rows = append(result.Rows, componentRows(g, componentResponses)...)
	case "components.request-bodies":
		result.Rows = append(result.Rows, componentRows(g, componentRequestBodies)...)
	case "components.headers":
		result.Rows = append(result.Rows, componentRows(g, componentHeaders)...)
	case "components.security-schemes":
		result.Rows = append(result.Rows, componentRows(g, componentSecuritySchemes)...)
	default:
		return nil, fmt.Errorf("unknown source: %q", stage.Source)
	}
	return result, nil
}

func execStageWithEnv(stage Stage, result *Result, g *graph.SchemaGraph, env map[string]expr.Value) (*Result, map[string]expr.Value, error) {
	switch stage.Kind {
	case StageLet:
		return execLet(stage, result, g, env)
	case StageWhere:
		r, err := execWhere(stage, result, g, env)
		return r, env, err
	default:
		r, err := execStage(stage, result, g)
		return r, env, err
	}
}

func execStage(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	switch stage.Kind {
	case StageLast:
		return execLast(stage, result)
	case StageSelect:
		result.Fields = stage.Fields
		return result, nil
	case StageSort:
		return execSort(stage, result, g)
	case StageTake:
		return execTake(stage, result)
	case StageUnique:
		return execUnique(result, g)
	case StageGroupBy:
		return execGroupBy(stage, result, g)
	case StageCount:
		return &Result{IsCount: true, Count: len(result.Rows)}, nil
	case StageRefsOut:
		if stage.Limit < 0 {
			return execTraversal(result, g, traverseReachable)
		}
		return execTraversal(result, g, traverseRefsOut)
	case StageRefsIn:
		if stage.Limit < 0 {
			return execTraversal(result, g, traverseAncestors)
		}
		return execTraversal(result, g, traverseRefsIn)
	case StageProperties:
		if stage.Limit < 0 {
			return execPropertiesFixpoint(result, g)
		}
		return execTraversal(result, g, traverseProperties)
	case StageItems:
		return execTraversal(result, g, traverseItems)
	case StageToOperations:
		return execSchemasToOps(result, g)
	case StageToSchemas:
		return execOpsToSchemas(result, g)
	case StageFields:
		return execFields(result)
	case StageSample:
		return execSample(stage, result)
	case StagePath:
		return execPath(stage, g)
	case StageHighest:
		// Expand to sort desc + take
		sorted, err := execSort(Stage{Kind: StageSort, SortField: stage.SortField, SortDesc: true}, result, g)
		if err != nil {
			return nil, err
		}
		return execTake(Stage{Kind: StageTake, Limit: stage.Limit}, sorted)
	case StageLowest:
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
	case StageCrossTag:
		return execTagBoundary(result, g)
	case StageSharedRefs:
		return execSharedRefs(stage, result, g)
	case StageOrigin:
		return execOrigin(result, g)
	case StageToYAML:
		result.EmitYAML = true
		return result, nil
	case StageParameters:
		return execParameters(result, g)
	case StageResponses:
		return execResponses(result, g)
	case StageRequestBody:
		return execRequestBody(result, g)
	case StageContentTypes:
		return execContentTypes(result, g)
	case StageHeaders:
		return execHeaders(result, g)
	case StageToSchema:
		return execSchema(result, g)
	case StageOperation:
		return execOperation(result, g)
	case StageSecurity:
		return execSecurity(result, g)
	case StageMembers:
		return execMembers(result, g)
	default:
		return nil, fmt.Errorf("unimplemented stage kind: %d", stage.Kind)
	}
}

func execWhere(stage Stage, result *Result, g *graph.SchemaGraph, env map[string]expr.Value) (*Result, error) {
	predicate, err := expr.Parse(stage.Expr)
	if err != nil {
		return nil, fmt.Errorf("where expression error: %w", err)
	}

	filtered := deriveResult(result)
	for _, row := range result.Rows {
		r := rowAdapter{row: row, g: g, env: env}
		val := predicate.Eval(r)
		if val.Kind == expr.KindBool && val.Bool {
			filtered.Rows = append(filtered.Rows, row)
		}
	}
	return filtered, nil
}

func execLast(stage Stage, result *Result) (*Result, error) {
	rows := result.Rows
	if stage.Limit < len(rows) {
		rows = rows[len(rows)-stage.Limit:]
	}
	out := deriveResult(result)
	out.Rows = slices.Clone(rows)
	return out, nil
}

func execLet(stage Stage, result *Result, g *graph.SchemaGraph, env map[string]expr.Value) (*Result, map[string]expr.Value, error) {
	predicate, err := expr.Parse(stage.Expr)
	if err != nil {
		return nil, env, fmt.Errorf("let expression error: %w", err)
	}

	// Evaluate against first row
	newEnv := make(map[string]expr.Value, len(env)+1)
	for k, v := range env {
		newEnv[k] = v
	}

	if len(result.Rows) > 0 {
		r := rowAdapter{row: result.Rows[0], g: g, env: env}
		val := predicate.Eval(r)
		newEnv[stage.VarName] = val
	} else {
		newEnv[stage.VarName] = expr.NullVal()
	}

	return result, newEnv, nil
}

func execSort(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	sorted := deriveResult(result)
	sorted.Rows = slices.Clone(result.Rows)
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
	out := deriveResult(result)
	out.Rows = slices.Clone(rows)
	return out, nil
}

func execUnique(result *Result, g *graph.SchemaGraph) (*Result, error) {
	seen := make(map[string]bool)
	filtered := deriveResult(result)
	for _, row := range result.Rows {
		var key string
		if len(result.Fields) > 0 {
			// When pick is active, deduplicate by projected field values
			var sb strings.Builder
			for i, f := range result.Fields {
				if i > 0 {
					sb.WriteByte('\x00')
				}
				sb.WriteString(valueToString(fieldValue(row, f, g)))
			}
			key = sb.String()
		} else {
			key = rowKey(row)
		}
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
	// Optional second field specifies which field to collect as group names (default: "name")
	nameField := "name"
	if len(stage.Fields) >= 2 {
		nameField = stage.Fields[1]
	}

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
		nameV := fieldValue(row, nameField, g)
		grp.names = append(grp.names, valueToString(nameV))
	}

	grouped := deriveResult(result)
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
		grouped.Rows = append(grouped.Rows, Row{
			Kind:       GroupRowResult,
			GroupKey:   key,
			GroupCount: grp.count,
			GroupNames: grp.names,
		})
	}
	return grouped, nil
}

// --- Traversal ---

type traversalFunc func(row Row, g *graph.SchemaGraph) []Row

func execTraversal(result *Result, g *graph.SchemaGraph, fn traversalFunc) (*Result, error) {
	out := deriveResult(result)
	seen := make(map[string]bool)
	for _, row := range result.Rows {
		seedName := schemaName(row.SchemaIdx, g)
		for _, newRow := range fn(row, g) {
			if newRow.Target == "" {
				newRow.Target = seedName
			}
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
	if row.Via == "" {
		return base
	}
	return base + "|" + row.From + "|" + row.Via + "|" + row.Key
}

// traverseOutEdges returns outgoing edge rows, optionally filtered by edge kind.
// If no kinds are given, all outgoing edges are included.
func traverseOutEdges(row Row, g *graph.SchemaGraph, kinds ...graph.EdgeKind) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	fromName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if len(kinds) > 0 && !edgeKindMatch(edge.Kind, kinds) {
			continue
		}
		result = append(result, Row{
			Kind:      SchemaResult,
			SchemaIdx: int(edge.To),
			Via:       edgeKindString(edge.Kind),
			Key:       edge.Label,
			From:      fromName,
		})
	}
	return result
}

func edgeKindMatch(k graph.EdgeKind, kinds []graph.EdgeKind) bool {
	for _, want := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

func traverseRefsOut(row Row, g *graph.SchemaGraph) []Row {
	return traverseOutEdges(row, g)
}

func traverseRefsIn(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	var result []Row
	for _, edge := range g.InEdges(graph.NodeID(row.SchemaIdx)) {
		idx := int(edge.From)
		// Resolve through inline wrapper nodes to reach the owning component schema.
		// Also capture the structural edge (the edge from the owner to the first
		// intermediate node) so via/key reflect how the owner includes this reference
		// (e.g., via=property key=owner) rather than the raw $ref edge.
		idx, structEdge := resolveToOwner(idx, g)
		via := edgeKindString(edge.Kind)
		key := edge.Label
		if structEdge != nil {
			via = edgeKindString(structEdge.Kind)
			key = structEdge.Label
		}
		result = append(result, Row{
			Kind:      SchemaResult,
			SchemaIdx: idx,
			Via:       via,
			Key:       key,
			From:      schemaName(idx, g),
		})
	}
	return result
}

// resolveToOwner walks up through inline schema nodes via in-edges
// until it reaches a component schema or a node with no in-edges.
// Returns the resolved owner index and the structural edge from the owner
// to the first intermediate node (or nil if no resolution was needed).
func resolveToOwner(idx int, g *graph.SchemaGraph) (int, *graph.Edge) {
	visited := make(map[int]bool)
	var lastEdge *graph.Edge
	for {
		if idx < 0 || idx >= len(g.Schemas) || visited[idx] {
			return idx, lastEdge
		}
		s := &g.Schemas[idx]
		if s.IsComponent {
			return idx, lastEdge
		}
		visited[idx] = true
		inEdges := g.InEdges(graph.NodeID(idx))
		if len(inEdges) == 0 {
			return idx, lastEdge
		}
		e := inEdges[0]
		lastEdge = &e
		idx = int(e.From)
	}
}

func nodeIDsToRows(ids []graph.NodeID) []Row {
	result := make([]Row, len(ids))
	for i, id := range ids {
		result[i] = Row{Kind: SchemaResult, SchemaIdx: int(id)}
	}
	return result
}

func traverseReachable(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	return nodeIDsToRows(g.Reachable(graph.NodeID(row.SchemaIdx)))
}

func traverseAncestors(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	return nodeIDsToRows(g.Ancestors(graph.NodeID(row.SchemaIdx)))
}

func traverseProperties(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	// Resolve $ref on input: if this schema is a $ref wrapper, follow to target.
	row.SchemaIdx = resolveRefTarget(row.SchemaIdx, g)
	fromName := schemaName(row.SchemaIdx, g)

	return collectSchemaProperties(row.SchemaIdx, fromName, g)
}

// collectSchemaProperties gets all properties from a schema, handling:
// - Direct property edges
// - allOf: transparent flattening (merged into parent fromName)
// - oneOf/anyOf: expansion with qualified from paths (e.g., "Event/oneOf/SystemEvent")
func collectSchemaProperties(idx int, fromName string, g *graph.SchemaGraph) []Row {
	var result []Row
	seen := make(map[string]bool) // dedup by (fromPrefix + key)

	collectPropertiesDirect(idx, fromName, g, seen, &result)
	collectAllOfProperties(idx, fromName, g, seen, &result)
	collectUnionProperties(idx, fromName, g, seen, &result)

	return result
}

// collectPropertiesDirect collects direct EdgeProperty edges.
func collectPropertiesDirect(idx int, fromName string, g *graph.SchemaGraph, seen map[string]bool, result *[]Row) {
	for _, edge := range g.OutEdges(graph.NodeID(idx)) {
		if edge.Kind == graph.EdgeProperty {
			dedupKey := fromName + "." + edge.Label
			if !seen[dedupKey] {
				seen[dedupKey] = true
				*result = append(*result, Row{
					Kind:      SchemaResult,
					SchemaIdx: int(edge.To),
					Via:       edgeKindString(edge.Kind),
					Key:       edge.Label,
					From:      fromName,
				})
			}
		}
	}
}

// collectAllOfProperties recursively collects properties from allOf members.
// allOf is transparent — properties are attributed to the parent fromName.
func collectAllOfProperties(idx int, fromName string, g *graph.SchemaGraph, seen map[string]bool, result *[]Row) {
	for _, edge := range g.OutEdges(graph.NodeID(idx)) {
		if edge.Kind != graph.EdgeAllOf {
			continue
		}
		memberIdx := resolveRefTarget(int(edge.To), g)
		collectPropertiesDirect(memberIdx, fromName, g, seen, result)
		// Recurse into nested allOf
		if g.Schemas[memberIdx].AllOfCount > 0 {
			collectAllOfProperties(memberIdx, fromName, g, seen, result)
		}
	}
}

// collectUnionProperties expands through oneOf/anyOf members,
// qualifying the from path (e.g., "Event/oneOf/SystemEvent").
func collectUnionProperties(idx int, _ string, g *graph.SchemaGraph, seen map[string]bool, result *[]Row) {
	parentName := schemaName(idx, g)
	for _, edge := range g.OutEdges(graph.NodeID(idx)) {
		if edge.Kind != graph.EdgeOneOf && edge.Kind != graph.EdgeAnyOf {
			continue
		}
		memberIdx := resolveRefTarget(int(edge.To), g)
		memberName := schemaName(memberIdx, g)
		qualifiedFrom := parentName + "/" + edgeKindString(edge.Kind) + "/" + memberName

		collectPropertiesDirect(memberIdx, qualifiedFrom, g, seen, result)
		collectAllOfProperties(memberIdx, qualifiedFrom, g, seen, result)
		// Recurse into nested unions
		collectUnionProperties(memberIdx, qualifiedFrom, g, seen, result)
	}
}

// execPropertiesFixpoint recursively expands properties until no new rows appear.
// Each call to traverseProperties handles $ref, allOf, and oneOf/anyOf uniformly.
func execPropertiesFixpoint(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	seen := make(map[int]bool)

	current := result.Rows
	for depth := 1; len(current) > 0; depth++ {
		var nextLevel []Row
		for _, row := range current {
			for _, prop := range traverseProperties(row, g) {
				if !seen[prop.SchemaIdx] {
					seen[prop.SchemaIdx] = true
					prop.BFSDepth = depth
					nextLevel = append(nextLevel, prop)
				}
			}
		}
		out.Rows = append(out.Rows, nextLevel...)
		current = nextLevel
	}

	return out, nil
}

func traverseUnionMembers(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	// Resolve $ref on input: if this schema is a $ref wrapper, follow to target.
	row.SchemaIdx = resolveRefTarget(row.SchemaIdx, g)
	fromName := schemaName(row.SchemaIdx, g)
	var result []Row
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind == graph.EdgeAllOf || edge.Kind == graph.EdgeOneOf || edge.Kind == graph.EdgeAnyOf {
			// Follow through $ref nodes transparently
			target := resolveRefTarget(int(edge.To), g)
			result = append(result, Row{
				Kind:      SchemaResult,
				SchemaIdx: target,
				Via:       edgeKindString(edge.Kind),
				Key:       edge.Label,
				From:      fromName,
			})
		}
	}
	return result
}

func traverseItems(row Row, g *graph.SchemaGraph) []Row {
	if row.Kind != SchemaResult {
		return nil
	}
	// Resolve $ref on input: if this schema is a $ref wrapper, follow to target.
	row.SchemaIdx = resolveRefTarget(row.SchemaIdx, g)
	// Direct items edges
	result := traverseOutEdges(row, g, graph.EdgeItems)
	if len(result) > 0 {
		return result
	}
	// If no direct items, check allOf members (composition may contain the array items)
	fromName := schemaName(row.SchemaIdx, g)
	for _, edge := range g.OutEdges(graph.NodeID(row.SchemaIdx)) {
		if edge.Kind != graph.EdgeAllOf {
			continue
		}
		memberIdx := resolveRefTarget(int(edge.To), g)
		for _, itemEdge := range g.OutEdges(graph.NodeID(memberIdx)) {
			if itemEdge.Kind == graph.EdgeItems {
				result = append(result, Row{
					Kind:      SchemaResult,
					SchemaIdx: int(itemEdge.To),
					Via:       edgeKindString(itemEdge.Kind),
					Key:       itemEdge.Label,
					From:      fromName,
				})
			}
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

// resolveThinWrapper follows through non-component inline schemas that are thin
// wrappers around a single child. Handles:
//   - {type: array, items: {$ref: ...}} → returns the items target
//   - {allOf: [{$ref: ...}]} → returns the single allOf target
//   - {oneOf: [{$ref: ...}]} → returns the single oneOf target
//
// If the schema is a component, has multiple children, or has its own properties,
// it's returned as-is.
func resolveThinWrapper(idx int, g *graph.SchemaGraph) int {
	if idx < 0 || idx >= len(g.Schemas) {
		return idx
	}
	s := &g.Schemas[idx]
	// Only unwrap non-component inline schemas
	if s.IsComponent {
		return idx
	}
	// Check for single-child pattern: exactly one outgoing edge
	edges := g.OutEdges(graph.NodeID(idx))
	if len(edges) != 1 {
		return idx
	}
	edge := edges[0]
	// Only follow items, allOf, oneOf, anyOf edges
	switch edge.Kind {
	case graph.EdgeItems, graph.EdgeAllOf, graph.EdgeOneOf, graph.EdgeAnyOf:
		target := resolveRefTarget(int(edge.To), g)
		return target
	default:
		return idx
	}
}

func execSchemasToOps(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
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
	out := deriveResult(result)
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
		default:
			// Non-schema/operation rows don't participate in connectivity analysis
		}
	}

	schemas, ops := g.ConnectedComponent(schemaSeeds, opSeeds)

	out := deriveResult(result)
	for _, id := range schemas {
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: int(id)})
	}
	for _, id := range ops {
		out.Rows = append(out.Rows, Row{Kind: OperationResult, OpIdx: int(id)})
	}
	return out, nil
}

func execBlastRadius(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
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

	// Add schema rows, skipping $ref wrapper nodes (pure plumbing)
	for _, idx := range schemaIndices {
		if idx < len(g.Schemas) && g.Schemas[idx].HasRef && !g.Schemas[idx].IsComponent {
			continue
		}
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
	out := deriveResult(result)
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
	out := deriveResult(result)
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
	out := deriveResult(result)
	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		// A leaf schema has no outgoing $ref edges to other component schemas.
		// Property children (inline type: string, type: integer, etc.) don't count.
		if !hasComponentRef(row.SchemaIdx, g) {
			out.Rows = append(out.Rows, row)
		}
	}
	return out, nil
}

// hasComponentRef returns true if the schema at idx has any outgoing edge
// that leads (possibly through a $ref) to a component schema.
func hasComponentRef(idx int, g *graph.SchemaGraph) bool {
	for _, edge := range g.OutEdges(graph.NodeID(idx)) {
		targetIdx := int(edge.To)
		// Follow $ref to see if it resolves to a component
		resolved := resolveRefTarget(targetIdx, g)
		if resolved < len(g.Schemas) && g.Schemas[resolved].IsComponent {
			return true
		}
	}
	return false
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

	out := deriveResult(result)
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
		key := "cycle-" + strconv.Itoa(i+1)
		out.Groups = append(out.Groups, GroupResult{
			Key:   key,
			Count: len(scc),
			Names: names,
		})
		out.Rows = append(out.Rows, Row{
			Kind:       GroupRowResult,
			GroupKey:   key,
			GroupCount: len(scc),
			GroupNames: names,
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
	out := deriveResult(result)
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
			key := "cluster-" + strconv.Itoa(clusterNum)
			out.Groups = append(out.Groups, GroupResult{
				Key:   key,
				Count: len(component),
				Names: names,
			})
			out.Rows = append(out.Rows, Row{
				Kind:       GroupRowResult,
				GroupKey:   key,
				GroupCount: len(component),
				GroupNames: names,
			})
		}
	}

	return out, nil
}

func execTagBoundary(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		// Default to component schemas only — inline schemas produce noise
		if !g.Schemas[row.SchemaIdx].IsComponent {
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

func execSharedRefs(stage Stage, result *Result, g *graph.SchemaGraph) (*Result, error) {
	var ops []graph.NodeID
	for _, row := range result.Rows {
		if row.Kind == OperationResult {
			ops = append(ops, graph.NodeID(row.OpIdx))
		}
	}

	if len(ops) == 0 {
		return deriveResult(result), nil
	}

	// Count how many operations reference each schema
	schemaCounts := make(map[graph.NodeID]int)
	for _, opID := range ops {
		for _, sid := range g.OperationSchemas(opID) {
			schemaCounts[sid]++
		}
	}

	// Determine threshold: shared-refs(N) = at least N ops, shared-refs = ALL ops
	minCount := len(ops) // default: shared by ALL
	if stage.Limit > 0 {
		minCount = stage.Limit
	}

	// Collect schemas meeting the threshold
	sortedIDs := make([]int, 0, len(schemaCounts))
	for sid, count := range schemaCounts {
		if count >= minCount {
			sortedIDs = append(sortedIDs, int(sid))
		}
	}
	sort.Ints(sortedIDs)

	out := deriveResult(result)
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
		return "Filter: where(" + stage.Expr + ")"
	case StageSelect:
		return "Project: select " + strings.Join(stage.Fields, ", ")
	case StageSort:
		dir := "asc"
		if stage.SortDesc {
			dir = "desc"
		}
		return "Sort: sort-by(" + stage.SortField + ", " + dir + ")"
	case StageTake:
		return "Limit: take(" + strconv.Itoa(stage.Limit) + ")"
	case StageLast:
		return "Limit: last(" + strconv.Itoa(stage.Limit) + ")"
	case StageLet:
		return "Bind: let " + stage.VarName + " = " + stage.Expr
	case StageUnique:
		return "Unique: deduplicate rows"
	case StageGroupBy:
		return "Group: group-by(" + strings.Join(stage.Fields, ", ") + ")"
	case StageCount:
		return "Count: count rows"
	case StageRefsOut:
		if stage.Limit < 0 {
			return "Traverse: refs-out(*) transitive closure"
		}
		return "Traverse: refs-out " + strconv.Itoa(stage.Limit) + " hop(s)"
	case StageRefsIn:
		if stage.Limit < 0 {
			return "Traverse: refs-in(*) transitive closure"
		}
		return "Traverse: refs-in " + strconv.Itoa(stage.Limit) + " hop(s)"
	case StageProperties:
		if stage.Limit < 0 {
			return "Traverse: properties(*) recursive"
		}
		return "Traverse: property children"
	case StageItems:
		return "Traverse: array items"
	case StageToOperations:
		return "Navigate: schemas to operations"
	case StageToSchemas:
		return "Navigate: operations to schemas"
	case StageFields:
		return "Terminal: list available fields"
	case StageSample:
		return "Sample: random " + strconv.Itoa(stage.Limit) + " rows"
	case StagePath:
		return "Path: shortest path from " + stage.PathFrom + " to " + stage.PathTo
	case StageHighest:
		return "Highest: " + strconv.Itoa(stage.Limit) + " by " + stage.SortField + " descending"
	case StageLowest:
		return "Lowest: " + strconv.Itoa(stage.Limit) + " by " + stage.SortField + " ascending"
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
		return "Filter: schemas with no $ref to component schemas (leaf nodes)"
	case StageCycles:
		return "Analyze: strongly connected components (actual cycles)"
	case StageClusters:
		return "Analyze: weakly connected component clusters"
	case StageCrossTag:
		return "Filter: schemas used by operations across multiple tags"
	case StageSharedRefs:
		if stage.Limit > 0 {
			return "Analyze: schemas shared by at least " + strconv.Itoa(stage.Limit) + " operations"
		}
		return "Analyze: schemas shared by all operations in result"
	case StageOrigin:
		return "Traverse: structural origin schema (walk up JSON pointer)"
	case StageToYAML:
		return "Output: raw YAML nodes from underlying spec objects"
	case StageParameters:
		return "Navigate: operation parameters"
	case StageResponses:
		return "Navigate: operation responses"
	case StageRequestBody:
		return "Navigate: operation request body"
	case StageContentTypes:
		return "Navigate: content types from responses or request body"
	case StageHeaders:
		return "Navigate: response headers"
	case StageToSchema:
		return "Navigate: extract schema from parameter, content-type, or header (to-schema)"
	case StageOperation:
		return "Navigate: back to source operation"
	case StageSecurity:
		return "Navigate: operation security requirements"
	case StageMembers:
		return "Expand: union members (allOf/oneOf/anyOf) or group row members"
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

	if kind == GroupRowResult {
		sb.WriteString("Field                          Type\n")
		sb.WriteString("-----------------------------  ------\n")
		fields := []struct{ name, typ string }{
			{"key", "string"},
			{"count", "int"},
			{"names", "string"},
		}
		for _, f := range fields {
			fmt.Fprintf(&sb, "%-30s %s\n", f.name, f.typ)
		}
		return &Result{Explain: sb.String()}, nil
	}

	sb.WriteString("Field                          Type\n")
	sb.WriteString("-----------------------------  ------\n")

	var fields []struct{ name, typ string }

	switch kind {
	case SchemaResult:
		fields = []struct{ name, typ string }{
			// Graph-level (pre-computed)
			{"name", "string"},
			{"type", "string"},
			{"depth", "int"},
			{"inDegree", "int"},
			{"outDegree", "int"},
			{"unionWidth", "int"},
			{"allOfCount", "int"},
			{"oneOfCount", "int"},
			{"anyOfCount", "int"},
			{"propertyCount", "int"},
			{"properties", "array"},
			{"isComponent", "bool"},
			{"isInline", "bool"},
			{"isCircular", "bool"},
			{"hasRef", "bool"},
			{"hash", "string"},
			{"path", "string"},
			{"opCount", "int"},
			{"tagCount", "int"},
			{"via", "string"},
			{"key", "string"},
			{"from", "string"},
			{"target", "string"},
			{"bfsDepth", "int"},
			// Schema content
			{"description", "string"},
			{"hasDescription", "bool"},
			{"title", "string"},
			{"hasTitle", "bool"},
			{"format", "string"},
			{"pattern", "string"},
			{"nullable", "bool"},
			{"readOnly", "bool"},
			{"writeOnly", "bool"},
			{"deprecated", "bool"},
			{"uniqueItems", "bool"},
			{"hasDiscriminator", "bool"},
			{"discriminatorProperty", "string"},
			{"discriminatorMappingCount", "int"},
			{"requiredCount", "int"},
			{"enumCount", "int"},
			{"hasDefault", "bool"},
			{"hasExample", "bool"},
			{"minimum", "int?"},
			{"maximum", "int?"},
			{"minLength", "int?"},
			{"maxLength", "int?"},
			{"minItems", "int?"},
			{"maxItems", "int?"},
			{"minProperties", "int?"},
			{"maxProperties", "int?"},
			{"extensionCount", "int"},
			{"contentEncoding", "string"},
			{"contentMediaType", "string"},
		}
	case OperationResult:
		fields = []struct{ name, typ string }{
			{"name", "string"},
			{"method", "string"},
			{"path", "string"},
			{"operationId", "string"},
			{"schemaCount", "int"},
			{"componentCount", "int"},
			{"tag", "string"},
			{"tags", "string"},
			{"parameterCount", "int"},
			{"deprecated", "bool"},
			{"description", "string"},
			{"summary", "string"},
			{"responseCount", "int"},
			{"hasErrorResponse", "bool"},
			{"hasRequestBody", "bool"},
			{"securityCount", "int"},
			{"via", "string"},
			{"key", "string"},
			{"from", "string"},
		}
	case ParameterResult:
		fields = []struct{ name, typ string }{
			{"name", "string"},
			{"in", "string"},
			{"required", "bool"},
			{"deprecated", "bool"},
			{"description", "string"},
			{"style", "string"},
			{"explode", "bool"},
			{"hasSchema", "bool"},
			{"allowEmptyValue", "bool"},
			{"allowReserved", "bool"},
			{"operation", "string"},
		}
	case ResponseResult:
		fields = []struct{ name, typ string }{
			{"statusCode", "string"},
			{"name", "string"},
			{"description", "string"},
			{"contentTypeCount", "int"},
			{"headerCount", "int"},
			{"linkCount", "int"},
			{"hasContent", "bool"},
			{"operation", "string"},
		}
	case RequestBodyResult:
		fields = []struct{ name, typ string }{
			{"name", "string"},
			{"description", "string"},
			{"required", "bool"},
			{"contentTypeCount", "int"},
			{"operation", "string"},
		}
	case ContentTypeResult:
		fields = []struct{ name, typ string }{
			{"mediaType", "string"},
			{"name", "string"},
			{"hasSchema", "bool"},
			{"hasEncoding", "bool"},
			{"hasExample", "bool"},
			{"statusCode", "string"},
			{"operation", "string"},
		}
	case HeaderResult:
		fields = []struct{ name, typ string }{
			{"name", "string"},
			{"description", "string"},
			{"required", "bool"},
			{"deprecated", "bool"},
			{"hasSchema", "bool"},
			{"statusCode", "string"},
			{"operation", "string"},
		}
	case SecuritySchemeResult:
		fields = []struct{ name, typ string }{
			{"name", "string"},
			{"type", "string"},
			{"in", "string"},
			{"scheme", "string"},
			{"bearerFormat", "string"},
			{"description", "string"},
			{"hasFlows", "bool"},
			{"deprecated", "bool"},
		}
	case SecurityRequirementResult:
		fields = []struct{ name, typ string }{
			{"schemeName", "string"},
			{"schemeType", "string"},
			{"scopes", "array"},
			{"scopeCount", "int"},
			{"operation", "string"},
		}
	default:
		// GroupRowResult handled above; unknown kinds produce empty fields list
	}

	for _, f := range fields {
		fmt.Fprintf(&sb, "%-30s %s\n", f.name, f.typ)
	}

	return &Result{Explain: sb.String()}, nil
}

// --- Origin ---

// execOrigin navigates back to the source schema of 1-hop edge annotations.
// After properties, union-members, items, refs-out, or refs-in, each row has
// a From field naming the source node. This stage looks up those source schemas
// by name, replacing the result set with the origin schemas.
func execOrigin(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	seen := make(map[int]bool)

	for _, row := range result.Rows {
		if row.Kind != SchemaResult {
			continue
		}
		// Follow in-edges to find the structural parent(s) of this schema.
		for _, edge := range g.InEdges(graph.NodeID(row.SchemaIdx)) {
			idx := int(edge.From)
			if !seen[idx] {
				seen[idx] = true
				out.Rows = append(out.Rows, Row{
					Kind:      SchemaResult,
					SchemaIdx: idx,
				})
			}
		}
	}
	return out, nil
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

	out := deriveResult(result)
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

	// Try forward direction first, then reverse if no path found
	path := g.ShortestPath(fromNode.ID, toNode.ID)
	if len(path) == 0 {
		path = g.ShortestPath(toNode.ID, fromNode.ID)
	}

	out := &Result{}
	for _, id := range path {
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: int(id)})
	}
	return out, nil
}

// --- Navigation stages ---

func execParameters(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	for _, row := range result.Rows {
		if row.Kind != OperationResult {
			continue
		}
		op := &g.Operations[row.OpIdx]
		if op.Operation == nil {
			continue
		}
		for _, paramRef := range op.Operation.Parameters {
			if paramRef == nil {
				continue
			}
			p := paramRef.GetObject()
			if p == nil {
				continue
			}
			out.Rows = append(out.Rows, Row{
				Kind:         ParameterResult,
				Parameter:    p,
				ComponentKey: p.Name,
				SourceOpIdx:  row.OpIdx,
				OpIdx:        row.OpIdx,
			})
		}
	}
	return out, nil
}

func execResponses(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	for _, row := range result.Rows {
		if row.Kind != OperationResult {
			continue
		}
		op := &g.Operations[row.OpIdx]
		if op.Operation == nil || op.Operation.Responses.Map == nil {
			continue
		}
		for code, respRef := range op.Operation.Responses.All() {
			if respRef == nil {
				continue
			}
			r := respRef.GetObject()
			if r == nil {
				continue
			}
			out.Rows = append(out.Rows, Row{
				Kind:        ResponseResult,
				Response:    r,
				StatusCode:  code,
				SourceOpIdx: row.OpIdx,
				OpIdx:       row.OpIdx,
			})
		}
		// Default response
		if op.Operation.Responses.Default != nil {
			r := op.Operation.Responses.Default.GetObject()
			if r != nil {
				out.Rows = append(out.Rows, Row{
					Kind:        ResponseResult,
					Response:    r,
					StatusCode:  "default",
					SourceOpIdx: row.OpIdx,
					OpIdx:       row.OpIdx,
				})
			}
		}
	}
	return out, nil
}

func execRequestBody(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	for _, row := range result.Rows {
		if row.Kind != OperationResult {
			continue
		}
		op := &g.Operations[row.OpIdx]
		if op.Operation == nil || op.Operation.RequestBody == nil {
			continue
		}
		rb := op.Operation.RequestBody.GetObject()
		if rb == nil {
			continue
		}
		out.Rows = append(out.Rows, Row{
			Kind:        RequestBodyResult,
			RequestBody: rb,
			SourceOpIdx: row.OpIdx,
			OpIdx:       row.OpIdx,
		})
	}
	return out, nil
}

func execContentTypes(result *Result, _ *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	for _, row := range result.Rows {
		switch row.Kind {
		case ResponseResult:
			if row.Response == nil || row.Response.Content == nil {
				continue
			}
			for mediaType, mt := range row.Response.Content.All() {
				if mt == nil {
					continue
				}
				out.Rows = append(out.Rows, Row{
					Kind:          ContentTypeResult,
					ContentType:   mt,
					MediaTypeName: mediaType,
					StatusCode:    row.StatusCode,
					SourceOpIdx:   row.SourceOpIdx,
					OpIdx:         row.OpIdx,
				})
			}
		case RequestBodyResult:
			if row.RequestBody == nil || row.RequestBody.Content == nil {
				continue
			}
			for mediaType, mt := range row.RequestBody.Content.All() {
				if mt == nil {
					continue
				}
				out.Rows = append(out.Rows, Row{
					Kind:          ContentTypeResult,
					ContentType:   mt,
					MediaTypeName: mediaType,
					SourceOpIdx:   row.SourceOpIdx,
					OpIdx:         row.OpIdx,
				})
			}
		default:
			// content-types only works on response and request-body rows
		}
	}
	return out, nil
}

func execHeaders(result *Result, _ *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	for _, row := range result.Rows {
		if row.Kind != ResponseResult || row.Response == nil || row.Response.Headers == nil {
			continue
		}
		for name, hdrRef := range row.Response.Headers.All() {
			if hdrRef == nil {
				continue
			}
			h := hdrRef.GetObject()
			if h == nil {
				continue
			}
			out.Rows = append(out.Rows, Row{
				Kind:        HeaderResult,
				Header:      h,
				HeaderName:  name,
				StatusCode:  row.StatusCode,
				SourceOpIdx: row.SourceOpIdx,
				OpIdx:       row.OpIdx,
			})
		}
	}
	return out, nil
}

func execSchema(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	seen := make(map[int]bool)

	resolveAndAdd := func(js *oas3.JSONSchemaReferenceable) {
		if js == nil {
			return
		}
		id, ok := g.SchemaByPtr(js)
		if !ok {
			return
		}
		idx := int(id)
		// Follow $ref edges to get the actual component schema
		idx = resolveRefTarget(idx, g)
		// Follow thin wrappers: if the schema is a non-component inline node
		// whose only purpose is to wrap a single child (items, single allOf/oneOf),
		// resolve to the meaningful target.
		idx = resolveThinWrapper(idx, g)
		if seen[idx] {
			return
		}
		seen[idx] = true
		out.Rows = append(out.Rows, Row{Kind: SchemaResult, SchemaIdx: idx})
	}

	for _, row := range result.Rows {
		switch row.Kind {
		case ParameterResult:
			if row.Parameter != nil && row.Parameter.Schema != nil {
				resolveAndAdd(row.Parameter.Schema)
			}
		case ContentTypeResult:
			if row.ContentType != nil && row.ContentType.Schema != nil {
				resolveAndAdd(row.ContentType.Schema)
			}
		case HeaderResult:
			if row.Header != nil && row.Header.Schema != nil {
				resolveAndAdd(row.Header.Schema)
			}
		default:
			// schema only works on parameter, content-type, and header rows
		}
	}
	return out, nil
}

func execOperation(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	seen := make(map[int]bool)
	for _, row := range result.Rows {
		var opIdx int
		switch row.Kind {
		case OperationResult:
			opIdx = row.OpIdx
		case ParameterResult, ResponseResult, RequestBodyResult, ContentTypeResult, HeaderResult,
			SecurityRequirementResult:
			opIdx = row.SourceOpIdx
		default:
			continue
		}
		if opIdx < 0 || opIdx >= len(g.Operations) || seen[opIdx] {
			continue
		}
		seen[opIdx] = true
		out.Rows = append(out.Rows, Row{Kind: OperationResult, OpIdx: opIdx})
	}
	return out, nil
}

// --- Component sources ---

type componentKind int

const (
	componentParameters componentKind = iota
	componentResponses
	componentRequestBodies
	componentHeaders
	componentSecuritySchemes
)

func componentRows(g *graph.SchemaGraph, kind componentKind) []Row {
	if g.Index == nil || g.Index.Doc == nil {
		return nil
	}
	components := g.Index.Doc.GetComponents()
	if components == nil {
		return nil
	}

	var rows []Row
	switch kind {
	case componentParameters:
		if components.Parameters == nil {
			return nil
		}
		for name, ref := range components.Parameters.All() {
			if ref == nil {
				continue
			}
			p := ref.GetObject()
			if p == nil {
				continue
			}
			rows = append(rows, Row{
				Kind:         ParameterResult,
				Parameter:    p,
				ComponentKey: name,
				SourceOpIdx:  -1,
			})
		}
	case componentResponses:
		if components.Responses == nil {
			return nil
		}
		for name, ref := range components.Responses.All() {
			if ref == nil {
				continue
			}
			r := ref.GetObject()
			if r == nil {
				continue
			}
			rows = append(rows, Row{
				Kind:         ResponseResult,
				Response:     r,
				ComponentKey: name,
				SourceOpIdx:  -1,
			})
		}
	case componentRequestBodies:
		if components.RequestBodies == nil {
			return nil
		}
		for name, ref := range components.RequestBodies.All() {
			if ref == nil {
				continue
			}
			rb := ref.GetObject()
			if rb == nil {
				continue
			}
			rows = append(rows, Row{
				Kind:         RequestBodyResult,
				RequestBody:  rb,
				ComponentKey: name,
				SourceOpIdx:  -1,
			})
		}
	case componentHeaders:
		if components.Headers == nil {
			return nil
		}
		for name, ref := range components.Headers.All() {
			if ref == nil {
				continue
			}
			h := ref.GetObject()
			if h == nil {
				continue
			}
			rows = append(rows, Row{
				Kind:        HeaderResult,
				Header:      h,
				HeaderName:  name,
				SourceOpIdx: -1,
			})
		}
	case componentSecuritySchemes:
		if components.SecuritySchemes == nil {
			return nil
		}
		for name, ref := range components.SecuritySchemes.All() {
			if ref == nil {
				continue
			}
			ss := ref.GetObject()
			if ss == nil {
				continue
			}
			rows = append(rows, Row{
				Kind:           SecuritySchemeResult,
				SecurityScheme: ss,
				SchemeName:     name,
				SourceOpIdx:    -1,
			})
		}
	}
	return rows
}

// --- Members ---

// execMembers expands group rows (from cycles, clusters, group_by) into their
// member schema rows. Each group's GroupNames are resolved to schema nodes.
func execMembers(result *Result, g *graph.SchemaGraph) (*Result, error) {
	if len(result.Rows) == 0 {
		return deriveResult(result), nil
	}

	// Dispatch based on input row type:
	// - GroupRowResult → expand group names into schema rows
	// - SchemaResult → expand union members (allOf/oneOf/anyOf children)
	if result.Rows[0].Kind == GroupRowResult {
		return execGroupMembers(result, g)
	}
	return execTraversal(result, g, traverseUnionMembers)
}

func execGroupMembers(result *Result, g *graph.SchemaGraph) (*Result, error) {
	nameIdx := make(map[string]int, len(g.Schemas))
	for i := range g.Schemas {
		nameIdx[schemaName(i, g)] = i
	}

	out := deriveResult(result)
	seen := make(map[int]bool)

	for _, row := range result.Rows {
		if row.Kind != GroupRowResult {
			continue
		}
		for _, name := range row.GroupNames {
			idx, ok := nameIdx[name]
			if !ok {
				continue
			}
			if seen[idx] {
				continue
			}
			seen[idx] = true
			out.Rows = append(out.Rows, Row{
				Kind:      SchemaResult,
				SchemaIdx: idx,
			})
		}
	}
	return out, nil
}

// --- Security ---

func execSecurity(result *Result, g *graph.SchemaGraph) (*Result, error) {
	out := deriveResult(result)
	// Build scheme name → SecurityScheme lookup
	schemeMap := buildSecuritySchemeMap(g)

	for _, row := range result.Rows {
		if row.Kind != OperationResult {
			continue
		}
		op := &g.Operations[row.OpIdx]
		if op.Operation == nil {
			continue
		}
		// Use per-operation security if explicitly set, otherwise inherit global
		secReqs := op.Operation.Security
		if secReqs == nil && g.Index != nil && g.Index.Doc != nil {
			secReqs = g.Index.Doc.GetSecurity()
		}
		for _, req := range secReqs {
			if req == nil {
				continue
			}
			for schemeName, scopes := range req.All() {
				r := Row{
					Kind:        SecurityRequirementResult,
					SchemeName:  schemeName,
					Scopes:      scopes,
					SourceOpIdx: row.OpIdx,
					OpIdx:       row.OpIdx,
				}
				if ss, ok := schemeMap[schemeName]; ok {
					r.SecurityScheme = ss
				}
				out.Rows = append(out.Rows, r)
			}
		}
	}
	return out, nil
}

func buildSecuritySchemeMap(g *graph.SchemaGraph) map[string]*openapi.SecurityScheme {
	m := make(map[string]*openapi.SecurityScheme)
	if g.Index == nil || g.Index.Doc == nil {
		return m
	}
	components := g.Index.Doc.GetComponents()
	if components == nil || components.SecuritySchemes == nil {
		return m
	}
	for name, ref := range components.SecuritySchemes.All() {
		if ref == nil {
			continue
		}
		ss := ref.GetObject()
		if ss != nil {
			m[name] = ss
		}
	}
	return m
}
