package analyze

import "sort"

// SCC represents a strongly connected component — a set of schemas
// that are all mutually reachable through references.
type SCC struct {
	// NodeIDs is the set of schema IDs in this component.
	NodeIDs []string
	// Size is len(NodeIDs).
	Size int
	// IsTrivial is true if the SCC has only one node and no self-loop.
	IsTrivial bool
}

// Cycle represents a specific cycle path through the schema graph.
type Cycle struct {
	// Path is the ordered list of schema IDs forming the cycle (last connects back to first).
	Path []string
	// Edges is the list of edges forming the cycle, parallel to Path.
	Edges []*Edge
	// Length is len(Path).
	Length int
	// HasRequiredOnlyPath is true if every edge in the cycle is required (no optional break point).
	HasRequiredOnlyPath bool
	// BreakPoints are edges that could be made optional/nullable to break the cycle.
	BreakPoints []*Edge
}

// CycleAnalysis holds the results of cycle and SCC analysis on a schema graph.
type CycleAnalysis struct {
	// SCCs is all non-trivial strongly connected components (size > 1 or self-loop).
	SCCs []*SCC
	// LargestSCCSize is the size of the largest SCC.
	LargestSCCSize int
	// Cycles is the enumerated list of distinct cycles.
	Cycles []*Cycle
	// NodesInCycles is the set of node IDs that participate in at least one cycle.
	NodesInCycles map[string]bool
	// DAGCondensation is the condensed DAG after collapsing SCCs.
	DAGCondensation *CondensedDAG
}

// CondensedDAG represents the graph after collapsing each SCC into a single node.
type CondensedDAG struct {
	// Nodes maps SCC index to the SCC.
	Nodes []*SCC
	// NodeToSCC maps schema ID to SCC index.
	NodeToSCC map[string]int
	// Edges are the edges between SCCs (deduplicated).
	Edges [][2]int // [from SCC index, to SCC index]
	// Depth is the longest path in the DAG (number of layers).
	Depth int
	// Layers groups SCCs by their topological layer (0 = no dependencies).
	Layers [][]int
}

// AnalyzeCycles performs SCC detection, cycle enumeration, and DAG condensation.
func AnalyzeCycles(g *Graph) *CycleAnalysis {
	result := &CycleAnalysis{
		NodesInCycles: make(map[string]bool),
	}

	// Step 1: Find SCCs using Tarjan's algorithm
	sccs := tarjanSCC(g)
	for _, scc := range sccs {
		if !scc.IsTrivial {
			result.SCCs = append(result.SCCs, scc)
			if scc.Size > result.LargestSCCSize {
				result.LargestSCCSize = scc.Size
			}
			for _, id := range scc.NodeIDs {
				result.NodesInCycles[id] = true
			}
		}
	}

	// Step 2: Enumerate cycles (bounded DFS within each SCC)
	result.Cycles = enumerateCycles(g, result.SCCs)

	// Step 3: Build condensed DAG
	result.DAGCondensation = buildCondensedDAG(g, sccs)

	return result
}

// tarjanSCC implements Tarjan's algorithm for finding strongly connected components.
func tarjanSCC(g *Graph) []*SCC {
	var (
		index   int
		stack   []string
		onStack = make(map[string]bool)
		indices = make(map[string]int)
		lowlink = make(map[string]int)
		result  []*SCC
	)

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, e := range g.OutEdges[v] {
			if _, visited := indices[e.To]; !visited {
				strongConnect(e.To)
				if lowlink[e.To] < lowlink[v] {
					lowlink[v] = lowlink[e.To]
				}
			} else if onStack[e.To] {
				if indices[e.To] < lowlink[v] {
					lowlink[v] = indices[e.To]
				}
			}
		}

		if lowlink[v] == indices[v] {
			scc := &SCC{}
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc.NodeIDs = append(scc.NodeIDs, w)
				if w == v {
					break
				}
			}
			scc.Size = len(scc.NodeIDs)
			scc.IsTrivial = scc.Size == 1 && !hasSelfLoop(g, scc.NodeIDs[0])
			sort.Strings(scc.NodeIDs) // deterministic ordering
			result = append(result, scc)
		}
	}

	// Visit all nodes (sorted for determinism)
	nodeIDs := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		if _, visited := indices[id]; !visited {
			strongConnect(id)
		}
	}

	return result
}

func hasSelfLoop(g *Graph, nodeID string) bool {
	for _, e := range g.OutEdges[nodeID] {
		if e.To == nodeID {
			return true
		}
	}
	return false
}

// enumerateCycles uses bounded DFS within each SCC to find distinct cycles.
// Limited to maxCyclesPerSCC to avoid combinatorial explosion.
func enumerateCycles(g *Graph, sccs []*SCC) []*Cycle {
	const maxCyclesPerSCC = 50

	var allCycles []*Cycle
	for _, scc := range sccs {
		sccSet := make(map[string]bool, scc.Size)
		for _, id := range scc.NodeIDs {
			sccSet[id] = true
		}

		cycles := findCyclesInSCC(g, scc.NodeIDs[0], sccSet, maxCyclesPerSCC)
		allCycles = append(allCycles, cycles...)
	}

	return allCycles
}

func findCyclesInSCC(g *Graph, startNode string, sccSet map[string]bool, maxCycles int) []*Cycle {
	var cycles []*Cycle
	visited := make(map[string]bool)
	path := []string{}
	pathEdges := []*Edge{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		if len(cycles) >= maxCycles {
			return true
		}

		visited[node] = true
		path = append(path, node)

		for _, e := range g.OutEdges[node] {
			if !sccSet[e.To] {
				continue
			}

			if e.To == startNode && len(path) > 1 {
				// Found a cycle back to start
				cyclePath := make([]string, len(path))
				copy(cyclePath, path)
				cycleEdges := make([]*Edge, len(pathEdges))
				copy(cycleEdges, pathEdges)
				cycleEdges = append(cycleEdges, e)

				cycle := &Cycle{
					Path:   cyclePath,
					Edges:  cycleEdges,
					Length: len(cyclePath),
				}
				classifyCycle(cycle)
				cycles = append(cycles, cycle)

				if len(cycles) >= maxCycles {
					return true
				}
				continue
			}

			if !visited[e.To] {
				pathEdges = append(pathEdges, e)
				if dfs(e.To) {
					return true
				}
				pathEdges = pathEdges[:len(pathEdges)-1]
			}
		}

		path = path[:len(path)-1]
		visited[node] = false
		return false
	}

	dfs(startNode)
	return cycles
}

func classifyCycle(c *Cycle) {
	allRequired := true
	for _, e := range c.Edges {
		if !e.IsRequired {
			allRequired = false
			c.BreakPoints = append(c.BreakPoints, e)
		} else if e.IsNullable || e.IsArray {
			c.BreakPoints = append(c.BreakPoints, e)
		}
	}
	c.HasRequiredOnlyPath = allRequired
}

// buildCondensedDAG collapses SCCs into single nodes and computes the DAG structure.
func buildCondensedDAG(g *Graph, sccs []*SCC) *CondensedDAG {
	dag := &CondensedDAG{
		Nodes:     sccs,
		NodeToSCC: make(map[string]int),
	}

	for i, scc := range sccs {
		for _, id := range scc.NodeIDs {
			dag.NodeToSCC[id] = i
		}
	}

	// Build edges between SCCs
	edgeSet := make(map[[2]int]bool)
	for _, e := range g.Edges {
		fromSCC, ok1 := dag.NodeToSCC[e.From]
		toSCC, ok2 := dag.NodeToSCC[e.To]
		if !ok1 || !ok2 || fromSCC == toSCC {
			continue
		}
		key := [2]int{fromSCC, toSCC}
		if !edgeSet[key] {
			edgeSet[key] = true
			dag.Edges = append(dag.Edges, key)
		}
	}

	// Compute topological layers
	dag.Layers = topologicalLayers(len(sccs), dag.Edges)
	dag.Depth = len(dag.Layers)

	return dag
}

// topologicalLayers assigns each node to a layer based on longest incoming path.
func topologicalLayers(nodeCount int, edges [][2]int) [][]int {
	inDegree := make([]int, nodeCount)
	adj := make([][]int, nodeCount)
	for i := range adj {
		adj[i] = []int{}
	}

	for _, e := range edges {
		adj[e[0]] = append(adj[e[0]], e[1])
		inDegree[e[1]]++
	}

	// Kahn's algorithm with layer tracking
	var queue []int
	layer := make([]int, nodeCount)
	for i := 0; i < nodeCount; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
			layer[i] = 0
		}
	}

	maxLayer := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, next := range adj[node] {
			if layer[node]+1 > layer[next] {
				layer[next] = layer[node] + 1
			}
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
				if layer[next] > maxLayer {
					maxLayer = layer[next]
				}
			}
		}
	}

	layers := make([][]int, maxLayer+1)
	for i := 0; i < nodeCount; i++ {
		layers[layer[i]] = append(layers[layer[i]], i)
	}

	return layers
}
