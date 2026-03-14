// Package graph provides a pre-computed directed graph over OpenAPI schemas and operations,
// materialized from an openapi.Index for efficient structural queries.
package graph

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/hashing"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
)

// NodeID is a unique identifier for a node in the graph.
type NodeID int

// EdgeKind represents the type of relationship between two schema nodes.
type EdgeKind int

const (
	EdgeProperty         EdgeKind = iota // properties/X
	EdgeItems                            // items
	EdgeAllOf                            // allOf[i]
	EdgeOneOf                            // oneOf[i]
	EdgeAnyOf                            // anyOf[i]
	EdgeAdditionalProps                  // additionalProperties
	EdgeNot                              // not
	EdgeIf                               // if
	EdgeThen                             // then
	EdgeElse                             // else
	EdgeContains                         // contains
	EdgePrefixItems                      // prefixItems[i]
	EdgeDependentSchema                  // dependentSchemas/X
	EdgePatternProperty                  // patternProperties/X
	EdgePropertyNames                    // propertyNames
	EdgeUnevaluatedItems                 // unevaluatedItems
	EdgeUnevaluatedProps                 // unevaluatedProperties
	EdgeRef                              // resolved $ref
)

// Edge represents a directed edge between two schema nodes.
type Edge struct {
	From  NodeID
	To    NodeID
	Kind  EdgeKind
	Label string // property name, pattern key, or index
}

// SchemaNode represents a schema in the graph.
type SchemaNode struct {
	ID            NodeID
	Name          string // component name or JSON pointer
	Path          string // JSON pointer in document
	Schema        *oas3.JSONSchemaReferenceable
	Location      openapi.Locations
	IsComponent   bool
	IsInline      bool
	IsExternal    bool
	IsBoolean     bool
	IsCircular    bool
	HasRef        bool
	Type          string // primary schema type
	Depth         int
	InDegree      int
	OutDegree     int
	UnionWidth    int
	PropertyCount int
	Hash          string
}

// OperationNode represents an operation in the graph.
type OperationNode struct {
	ID             NodeID
	Name           string // operationId or "METHOD /path"
	Method         string
	Path           string
	OperationID    string
	Operation      *openapi.Operation
	Location       openapi.Locations
	SchemaCount    int
	ComponentCount int
}

// SchemaGraph is a pre-computed directed graph over OpenAPI schemas and operations.
type SchemaGraph struct {
	Schemas    []SchemaNode
	Operations []OperationNode

	outEdges map[NodeID][]Edge
	inEdges  map[NodeID][]Edge

	// Lookup maps
	ptrToNode  map[*oas3.JSONSchemaReferenceable]NodeID
	nameToNode map[string]NodeID

	// Operation-schema relationships
	opSchemas map[NodeID]map[NodeID]bool // operation -> set of schema NodeIDs
	schemaOps map[NodeID]map[NodeID]bool // schema -> set of operation NodeIDs
}

// Build constructs a SchemaGraph from an openapi.Index.
func Build(_ context.Context, idx *openapi.Index) *SchemaGraph {
	g := &SchemaGraph{
		outEdges:   make(map[NodeID][]Edge),
		inEdges:    make(map[NodeID][]Edge),
		ptrToNode:  make(map[*oas3.JSONSchemaReferenceable]NodeID),
		nameToNode: make(map[string]NodeID),
		opSchemas:  make(map[NodeID]map[NodeID]bool),
		schemaOps:  make(map[NodeID]map[NodeID]bool),
	}

	// Phase 1: Register nodes
	g.registerNodes(idx)

	// Phase 2: Build edges
	g.buildEdges()

	// Phase 3: Operation edges
	g.buildOperationEdges(idx)

	// Phase 4: Compute metrics
	g.computeMetrics()

	return g
}

// OutEdges returns the outgoing edges from the given node.
func (g *SchemaGraph) OutEdges(id NodeID) []Edge {
	return g.outEdges[id]
}

// InEdges returns the incoming edges to the given node.
func (g *SchemaGraph) InEdges(id NodeID) []Edge {
	return g.inEdges[id]
}

// SchemaByName returns the schema node with the given component name, if any.
func (g *SchemaGraph) SchemaByName(name string) (SchemaNode, bool) {
	if id, ok := g.nameToNode[name]; ok && int(id) < len(g.Schemas) {
		return g.Schemas[id], true
	}
	return SchemaNode{}, false
}

// SchemaByPtr returns the NodeID for a schema identified by its pointer.
func (g *SchemaGraph) SchemaByPtr(ptr *oas3.JSONSchemaReferenceable) (NodeID, bool) {
	id, ok := g.ptrToNode[ptr]
	return id, ok
}

// OperationSchemas returns the schema NodeIDs reachable from the given operation.
// Results are sorted by NodeID for deterministic output.
func (g *SchemaGraph) OperationSchemas(opID NodeID) []NodeID {
	set := g.opSchemas[opID]
	ids := make([]NodeID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// SchemaOperations returns the operation NodeIDs that reference the given schema.
// Results are sorted by NodeID for deterministic output.
func (g *SchemaGraph) SchemaOperations(schemaID NodeID) []NodeID {
	set := g.schemaOps[schemaID]
	ids := make([]NodeID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Phase 1: Register all schema nodes from the index.
func (g *SchemaGraph) registerNodes(idx *openapi.Index) {
	addSchema := func(node *openapi.IndexNode[*oas3.JSONSchemaReferenceable], isComponent, isInline, isExternal, isBoolean bool) {
		if node == nil || node.Node == nil {
			return
		}
		// Avoid duplicates
		if _, exists := g.ptrToNode[node.Node]; exists {
			return
		}

		id := NodeID(len(g.Schemas))
		jp := string(node.Location.ToJSONPointer())

		name := jp
		if isComponent {
			// Extract component name from the JSON pointer: /components/schemas/Name
			parts := strings.Split(jp, "/")
			if len(parts) >= 4 {
				name = parts[len(parts)-1]
			}
		}

		hasRef := false
		schemaType := ""
		if schema := node.Node.GetSchema(); schema != nil {
			hasRef = schema.Ref != nil
			types := schema.GetType()
			if len(types) > 0 {
				schemaType = string(types[0])
			}
		}

		sn := SchemaNode{
			ID:          id,
			Name:        name,
			Path:        jp,
			Schema:      node.Node,
			Location:    node.Location,
			IsComponent: isComponent,
			IsInline:    isInline,
			IsExternal:  isExternal,
			IsBoolean:   isBoolean,
			HasRef:      hasRef,
			Type:        schemaType,
		}

		g.Schemas = append(g.Schemas, sn)
		g.ptrToNode[node.Node] = id
		if isComponent {
			g.nameToNode[name] = id
		}
	}

	for _, n := range idx.ComponentSchemas {
		addSchema(n, true, false, false, false)
	}
	for _, n := range idx.InlineSchemas {
		addSchema(n, false, true, false, false)
	}
	for _, n := range idx.ExternalSchemas {
		addSchema(n, false, false, true, false)
	}
	for _, n := range idx.BooleanSchemas {
		addSchema(n, false, false, false, true)
	}

	// Also register schema references (nodes that are $refs to other schemas)
	for _, n := range idx.SchemaReferences {
		addSchema(n, false, false, false, false)
	}
}

// Phase 2: Build edges by inspecting child-bearing fields of each schema.
func (g *SchemaGraph) buildEdges() {
	for i := range g.Schemas {
		sn := &g.Schemas[i]
		schema := sn.Schema.GetSchema()
		if schema == nil {
			continue
		}

		// If this is a $ref node, add an edge to the resolved target
		if schema.Ref != nil {
			if targetID, ok := g.resolveRef(string(*schema.Ref)); ok {
				g.addEdge(sn.ID, targetID, EdgeRef, string(*schema.Ref))
			}
		}

		// Properties
		if schema.Properties != nil {
			for key, child := range schema.Properties.All() {
				if childID, ok := g.resolveChild(child); ok {
					g.addEdge(sn.ID, childID, EdgeProperty, key)
				}
			}
		}

		// Items
		if schema.Items != nil {
			if childID, ok := g.resolveChild(schema.Items); ok {
				g.addEdge(sn.ID, childID, EdgeItems, "items")
			}
		}

		// AllOf
		for j, child := range schema.AllOf {
			if childID, ok := g.resolveChild(child); ok {
				g.addEdge(sn.ID, childID, EdgeAllOf, "allOf/"+intStr(j))
			}
		}

		// OneOf
		for j, child := range schema.OneOf {
			if childID, ok := g.resolveChild(child); ok {
				g.addEdge(sn.ID, childID, EdgeOneOf, "oneOf/"+intStr(j))
			}
		}

		// AnyOf
		for j, child := range schema.AnyOf {
			if childID, ok := g.resolveChild(child); ok {
				g.addEdge(sn.ID, childID, EdgeAnyOf, "anyOf/"+intStr(j))
			}
		}

		// AdditionalProperties
		if schema.AdditionalProperties != nil {
			if childID, ok := g.resolveChild(schema.AdditionalProperties); ok {
				g.addEdge(sn.ID, childID, EdgeAdditionalProps, "additionalProperties")
			}
		}

		// Not
		if schema.Not != nil {
			if childID, ok := g.resolveChild(schema.Not); ok {
				g.addEdge(sn.ID, childID, EdgeNot, "not")
			}
		}

		// If / Then / Else
		if schema.If != nil {
			if childID, ok := g.resolveChild(schema.If); ok {
				g.addEdge(sn.ID, childID, EdgeIf, "if")
			}
		}
		if schema.Then != nil {
			if childID, ok := g.resolveChild(schema.Then); ok {
				g.addEdge(sn.ID, childID, EdgeThen, "then")
			}
		}
		if schema.Else != nil {
			if childID, ok := g.resolveChild(schema.Else); ok {
				g.addEdge(sn.ID, childID, EdgeElse, "else")
			}
		}

		// Contains
		if schema.Contains != nil {
			if childID, ok := g.resolveChild(schema.Contains); ok {
				g.addEdge(sn.ID, childID, EdgeContains, "contains")
			}
		}

		// PrefixItems
		for j, child := range schema.PrefixItems {
			if childID, ok := g.resolveChild(child); ok {
				g.addEdge(sn.ID, childID, EdgePrefixItems, "prefixItems/"+intStr(j))
			}
		}

		// DependentSchemas
		if schema.DependentSchemas != nil {
			for key, child := range schema.DependentSchemas.All() {
				if childID, ok := g.resolveChild(child); ok {
					g.addEdge(sn.ID, childID, EdgeDependentSchema, key)
				}
			}
		}

		// PatternProperties
		if schema.PatternProperties != nil {
			for key, child := range schema.PatternProperties.All() {
				if childID, ok := g.resolveChild(child); ok {
					g.addEdge(sn.ID, childID, EdgePatternProperty, key)
				}
			}
		}

		// PropertyNames
		if schema.PropertyNames != nil {
			if childID, ok := g.resolveChild(schema.PropertyNames); ok {
				g.addEdge(sn.ID, childID, EdgePropertyNames, "propertyNames")
			}
		}

		// UnevaluatedItems
		if schema.UnevaluatedItems != nil {
			if childID, ok := g.resolveChild(schema.UnevaluatedItems); ok {
				g.addEdge(sn.ID, childID, EdgeUnevaluatedItems, "unevaluatedItems")
			}
		}

		// UnevaluatedProperties
		if schema.UnevaluatedProperties != nil {
			if childID, ok := g.resolveChild(schema.UnevaluatedProperties); ok {
				g.addEdge(sn.ID, childID, EdgeUnevaluatedProps, "unevaluatedProperties")
			}
		}
	}
}

// resolveChild finds the node ID for a child schema pointer.
// If the pointer is directly registered, returns it.
// If not, checks if it's a $ref and resolves via the component name lookup.
func (g *SchemaGraph) resolveChild(child *oas3.JSONSchemaReferenceable) (NodeID, bool) {
	if child == nil {
		return 0, false
	}
	// Direct pointer match
	if id, ok := g.ptrToNode[child]; ok {
		return id, true
	}
	// Try to resolve via $ref
	if s := child.GetSchema(); s != nil && s.Ref != nil {
		return g.resolveRef(string(*s.Ref))
	}
	return 0, false
}

// resolveRef resolves a $ref string (e.g., "#/components/schemas/Owner") to a node ID.
func (g *SchemaGraph) resolveRef(ref string) (NodeID, bool) {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		name := ref[len(prefix):]
		if id, ok := g.nameToNode[name]; ok {
			return id, true
		}
	}
	return 0, false
}

func (g *SchemaGraph) addEdge(from, to NodeID, kind EdgeKind, label string) {
	e := Edge{From: from, To: to, Kind: kind, Label: label}
	g.outEdges[from] = append(g.outEdges[from], e)
	g.inEdges[to] = append(g.inEdges[to], e)
}

// Phase 3: Build operation nodes and operation-schema relationships.
func (g *SchemaGraph) buildOperationEdges(idx *openapi.Index) {
	for _, opNode := range idx.Operations {
		if opNode == nil || opNode.Node == nil {
			continue
		}

		method, path := openapi.ExtractMethodAndPath(opNode.Location)
		opID := opNode.Node.GetOperationID()

		name := opID
		if name == "" {
			name = strings.ToUpper(method) + " " + path
		}

		opNodeID := NodeID(len(g.Operations))
		on := OperationNode{
			ID:          opNodeID,
			Name:        name,
			Method:      method,
			Path:        path,
			OperationID: opID,
			Operation:   opNode.Node,
			Location:    opNode.Location,
		}

		// Find schemas reachable from this operation by walking its structure
		directSchemas := g.findOperationSchemas(opNode.Node)

		// Build transitive closure from direct schemas
		reachable := make(map[NodeID]bool)
		for _, sid := range directSchemas {
			g.reachableBFS(sid, reachable)
		}

		g.opSchemas[opNodeID] = reachable

		componentCount := 0
		for sid := range reachable {
			if int(sid) < len(g.Schemas) && g.Schemas[sid].IsComponent {
				componentCount++
			}
			// Build reverse mapping
			if g.schemaOps[sid] == nil {
				g.schemaOps[sid] = make(map[NodeID]bool)
			}
			g.schemaOps[sid][opNodeID] = true
		}

		on.SchemaCount = len(reachable)
		on.ComponentCount = componentCount

		g.Operations = append(g.Operations, on)
	}
}

// findOperationSchemas finds schema NodeIDs directly referenced by an operation's
// parameters, request body, and responses.
func (g *SchemaGraph) findOperationSchemas(op *openapi.Operation) []NodeID {
	var result []NodeID
	seen := make(map[NodeID]bool)

	addIfKnown := func(js *oas3.JSONSchemaReferenceable) {
		if js == nil {
			return
		}
		if id, ok := g.ptrToNode[js]; ok && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	// Walk parameters
	for _, param := range op.Parameters {
		if param == nil {
			continue
		}
		p := param.GetObject()
		if p == nil {
			continue
		}
		if p.Schema != nil {
			addIfKnown(p.Schema)
		}
	}

	// Walk request body
	if op.RequestBody != nil {
		rb := op.RequestBody.GetObject()
		if rb != nil && rb.Content != nil {
			for _, mt := range rb.Content.All() {
				if mt != nil && mt.Schema != nil {
					addIfKnown(mt.Schema)
				}
			}
		}
	}

	// Walk responses
	for _, resp := range op.Responses.All() {
		if resp == nil {
			continue
		}
		r := resp.GetObject()
		if r == nil || r.Content == nil {
			continue
		}
		for _, mt := range r.Content.All() {
			if mt != nil && mt.Schema != nil {
				addIfKnown(mt.Schema)
			}
		}
	}
	// Also check default response
	if op.Responses.Default != nil {
		r := op.Responses.Default.GetObject()
		if r != nil && r.Content != nil {
			for _, mt := range r.Content.All() {
				if mt != nil && mt.Schema != nil {
					addIfKnown(mt.Schema)
				}
			}
		}
	}

	return result
}

// reachableBFS performs BFS from a schema node and adds all reachable nodes to the set.
func (g *SchemaGraph) reachableBFS(start NodeID, visited map[NodeID]bool) {
	if visited[start] {
		return
	}
	queue := []NodeID{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range g.outEdges[current] {
			if !visited[edge.To] {
				visited[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
}

// Phase 4: Compute metrics for each schema node.
func (g *SchemaGraph) computeMetrics() {
	// Detect circular nodes with a single shared DFS (O(V+E))
	circularNodes := make(map[NodeID]bool)
	visited := make(map[NodeID]bool)
	inStack := make(map[NodeID]bool)
	for i := range g.Schemas {
		nid := NodeID(i)
		if !visited[nid] {
			g.detectCycle(nid, visited, inStack, circularNodes)
		}
	}

	for i := range g.Schemas {
		sn := &g.Schemas[i]
		id := NodeID(i)

		sn.OutDegree = len(g.outEdges[id])
		sn.InDegree = len(g.inEdges[id])
		sn.IsCircular = circularNodes[id]

		schema := sn.Schema.GetSchema()
		if schema != nil {
			sn.UnionWidth = len(schema.AllOf) + len(schema.OneOf) + len(schema.AnyOf)
			if schema.Properties != nil {
				sn.PropertyCount = schema.Properties.Len()
			}
			sn.Hash = hashing.Hash(schema)
		}

		// Compute depth via DFS with cycle detection
		depthVisited := make(map[NodeID]bool)
		sn.Depth = g.computeDepth(id, depthVisited)
	}
}

func (g *SchemaGraph) computeDepth(id NodeID, visited map[NodeID]bool) int {
	if visited[id] {
		return 0 // cycle
	}
	visited[id] = true

	maxChild := 0
	for _, edge := range g.outEdges[id] {
		d := g.computeDepth(edge.To, visited)
		if d+1 > maxChild {
			maxChild = d + 1
		}
	}
	visited[id] = false
	return maxChild
}

// detectCycle performs a DFS from id, marking nodes that participate in cycles.
// It returns the NodeID of the cycle entry point that still needs to be "closed"
// by an ancestor frame, or -1 if no open cycle passes through this node.
func (g *SchemaGraph) detectCycle(id NodeID, visited, inStack map[NodeID]bool, circular map[NodeID]bool) NodeID {
	if inStack[id] {
		circular[id] = true
		return id // back-edge found; id is the cycle entry point
	}
	if visited[id] {
		return -1
	}
	visited[id] = true
	inStack[id] = true

	var outerEntry NodeID = -1
	for _, edge := range g.outEdges[id] {
		entry := g.detectCycle(edge.To, visited, inStack, circular)
		if entry != -1 {
			circular[id] = true
			// If the cycle entry is this node, the cycle is closed — don't propagate.
			// Otherwise, remember the outermost open entry to propagate upward.
			if entry != id {
				outerEntry = entry
			}
		}
	}

	inStack[id] = false
	return outerEntry
}

// Reachable returns all schema NodeIDs transitively reachable from the given node via out-edges.
func (g *SchemaGraph) Reachable(id NodeID) []NodeID {
	visited := make(map[NodeID]bool)
	g.reachableBFS(id, visited)
	delete(visited, id) // exclude self
	result := make([]NodeID, 0, len(visited))
	for nid := range visited {
		result = append(result, nid)
	}
	return result
}

// Ancestors returns all schema NodeIDs that can transitively reach the given node via in-edges.
func (g *SchemaGraph) Ancestors(id NodeID) []NodeID {
	visited := make(map[NodeID]bool)
	visited[id] = true
	queue := []NodeID{id}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range g.inEdges[current] {
			if !visited[edge.From] {
				visited[edge.From] = true
				queue = append(queue, edge.From)
			}
		}
	}

	delete(visited, id) // exclude self
	result := make([]NodeID, 0, len(visited))
	for nid := range visited {
		result = append(result, nid)
	}
	return result
}

// ShortestPath returns the shortest path from `from` to `to` using out-edges (BFS).
// Returns nil if no path exists. The returned slice includes both endpoints.
func (g *SchemaGraph) ShortestPath(from, to NodeID) []NodeID {
	if from == to {
		return []NodeID{from}
	}

	parent := make(map[NodeID]NodeID)
	visited := make(map[NodeID]bool)
	visited[from] = true
	queue := []NodeID{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range g.outEdges[current] {
			if visited[edge.To] {
				continue
			}
			visited[edge.To] = true
			parent[edge.To] = current

			if edge.To == to {
				// Reconstruct path
				var path []NodeID
				for n := to; n != from; n = parent[n] {
					path = append(path, n)
				}
				path = append(path, from)
				// Reverse
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path
			}

			queue = append(queue, edge.To)
		}
	}

	return nil
}

// SchemaOpCount returns the number of operations that reference the given schema.
func (g *SchemaGraph) SchemaOpCount(id NodeID) int {
	return len(g.schemaOps[id])
}

// Neighbors returns schema NodeIDs within maxDepth hops of the given node,
// following both out-edges and in-edges (bidirectional BFS).
// The result excludes the seed node itself.
func (g *SchemaGraph) Neighbors(id NodeID, maxDepth int) []NodeID {
	visited := map[NodeID]bool{id: true}
	current := []NodeID{id}

	for depth := 0; depth < maxDepth && len(current) > 0; depth++ {
		var next []NodeID
		for _, nid := range current {
			for _, edge := range g.outEdges[nid] {
				if !visited[edge.To] {
					visited[edge.To] = true
					next = append(next, edge.To)
				}
			}
			for _, edge := range g.inEdges[nid] {
				if !visited[edge.From] {
					visited[edge.From] = true
					next = append(next, edge.From)
				}
			}
		}
		current = next
	}

	delete(visited, id)
	result := make([]NodeID, 0, len(visited))
	for nid := range visited {
		result = append(result, nid)
	}
	return result
}

// StronglyConnectedComponents returns the SCCs of the schema graph using
// Tarjan's algorithm. Only returns components with more than one node
// (i.e., actual cycles, not singleton nodes).
func (g *SchemaGraph) StronglyConnectedComponents() [][]NodeID {
	idx := 0
	var stack []NodeID
	onStack := make(map[NodeID]bool)
	indices := make(map[NodeID]int)
	lowlinks := make(map[NodeID]int)
	defined := make(map[NodeID]bool)
	var sccs [][]NodeID

	var strongConnect func(v NodeID)
	strongConnect = func(v NodeID) {
		indices[v] = idx
		lowlinks[v] = idx
		defined[v] = true
		idx++
		stack = append(stack, v)
		onStack[v] = true

		for _, edge := range g.outEdges[v] {
			w := edge.To
			if !defined[w] {
				strongConnect(w)
				if lowlinks[w] < lowlinks[v] {
					lowlinks[v] = lowlinks[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlinks[v] {
					lowlinks[v] = indices[w]
				}
			}
		}

		if lowlinks[v] == indices[v] {
			var scc []NodeID
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			if len(scc) > 1 {
				sccs = append(sccs, scc)
			}
		}
	}

	for i := range g.Schemas {
		nid := NodeID(i)
		if !defined[nid] {
			strongConnect(nid)
		}
	}

	return sccs
}

// ConnectedComponent computes the full connected component reachable from the
// given seed schema and operation nodes. It treats schema edges as undirected
// (follows both out-edges and in-edges) and crosses schema↔operation links.
// Returns the sets of reachable schema and operation NodeIDs (including seeds).
func (g *SchemaGraph) ConnectedComponent(schemaSeeds, opSeeds []NodeID) (schemas []NodeID, ops []NodeID) {
	visitedSchemas := make(map[NodeID]bool)
	visitedOps := make(map[NodeID]bool)

	// Queues for BFS across both node types
	schemaQueue := make([]NodeID, 0, len(schemaSeeds))
	opQueue := make([]NodeID, 0, len(opSeeds))

	for _, id := range schemaSeeds {
		if !visitedSchemas[id] {
			visitedSchemas[id] = true
			schemaQueue = append(schemaQueue, id)
		}
	}
	for _, id := range opSeeds {
		if !visitedOps[id] {
			visitedOps[id] = true
			opQueue = append(opQueue, id)
		}
	}

	for len(schemaQueue) > 0 || len(opQueue) > 0 {
		// Process schema nodes
		for len(schemaQueue) > 0 {
			current := schemaQueue[0]
			schemaQueue = schemaQueue[1:]

			// Follow out-edges (undirected: treat as bidirectional)
			for _, edge := range g.outEdges[current] {
				if !visitedSchemas[edge.To] {
					visitedSchemas[edge.To] = true
					schemaQueue = append(schemaQueue, edge.To)
				}
			}
			// Follow in-edges
			for _, edge := range g.inEdges[current] {
				if !visitedSchemas[edge.From] {
					visitedSchemas[edge.From] = true
					schemaQueue = append(schemaQueue, edge.From)
				}
			}
			// Cross to operations
			for opID := range g.schemaOps[current] {
				if !visitedOps[opID] {
					visitedOps[opID] = true
					opQueue = append(opQueue, opID)
				}
			}
		}

		// Process operation nodes
		for len(opQueue) > 0 {
			current := opQueue[0]
			opQueue = opQueue[1:]

			// Cross to schemas
			for sid := range g.opSchemas[current] {
				if !visitedSchemas[sid] {
					visitedSchemas[sid] = true
					schemaQueue = append(schemaQueue, sid)
				}
			}
		}
	}

	schemas = make([]NodeID, 0, len(visitedSchemas))
	for id := range visitedSchemas {
		schemas = append(schemas, id)
	}
	ops = make([]NodeID, 0, len(visitedOps))
	for id := range visitedOps {
		ops = append(ops, id)
	}
	return schemas, ops
}

func intStr(i int) string {
	return strconv.Itoa(i)
}
