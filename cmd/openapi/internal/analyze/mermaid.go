package analyze

import (
	"fmt"
	"sort"
	"strings"
)

// SCCToMermaid renders one SCC as a Mermaid flowchart string.
// The returned string can be displayed or piped to mermaid-rendering tools.
func SCCToMermaid(g *Graph, cycles *CycleAnalysis, sccIndex int) string {
	if sccIndex < 0 || sccIndex >= len(cycles.SCCs) {
		return ""
	}
	scc := cycles.SCCs[sccIndex]
	memberSet := make(map[string]bool, scc.Size)
	for _, id := range scc.NodeIDs {
		memberSet[id] = true
	}

	var sb strings.Builder
	sb.WriteString("graph LR\n")

	// Nodes
	for _, id := range scc.NodeIDs {
		sb.WriteString(fmt.Sprintf("  %s[%s]\n", mermaidSafeID(id), id))
	}

	// Edges within SCC
	for _, e := range g.Edges {
		if memberSet[e.From] && memberSet[e.To] {
			label := mermaidEdgeLabel(e)
			if label != "" {
				sb.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", mermaidSafeID(e.From), label, mermaidSafeID(e.To)))
			} else {
				sb.WriteString(fmt.Sprintf("  %s --> %s\n", mermaidSafeID(e.From), mermaidSafeID(e.To)))
			}
		}
	}

	return sb.String()
}

// EgoGraphToMermaid renders a BFS neighborhood subgraph around a node.
func EgoGraphToMermaid(g *Graph, nodeID string, hops int) string {
	if _, ok := g.Nodes[nodeID]; !ok {
		return ""
	}

	// BFS to collect nodes within hop radius
	visited := map[string]int{nodeID: 0}
	queue := []string{nodeID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		dist := visited[current]
		if dist >= hops {
			continue
		}
		// outgoing
		for _, e := range g.OutEdges[current] {
			if _, seen := visited[e.To]; !seen {
				visited[e.To] = dist + 1
				queue = append(queue, e.To)
			}
		}
		// incoming
		for _, e := range g.InEdges[current] {
			if _, seen := visited[e.From]; !seen {
				visited[e.From] = dist + 1
				queue = append(queue, e.From)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("graph LR\n")

	// Sort node IDs for deterministic output
	nodeIDs := make([]string, 0, len(visited))
	for id := range visited {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		if id == nodeID {
			sb.WriteString(fmt.Sprintf("  %s((%s))\n", mermaidSafeID(id), id)) // double circle for center
		} else {
			sb.WriteString(fmt.Sprintf("  %s[%s]\n", mermaidSafeID(id), id))
		}
	}

	// Edges between visited nodes
	for _, e := range g.Edges {
		if _, ok := visited[e.From]; !ok {
			continue
		}
		if _, ok := visited[e.To]; !ok {
			continue
		}
		label := mermaidEdgeLabel(e)
		if label != "" {
			sb.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", mermaidSafeID(e.From), label, mermaidSafeID(e.To)))
		} else {
			sb.WriteString(fmt.Sprintf("  %s --> %s\n", mermaidSafeID(e.From), mermaidSafeID(e.To)))
		}
	}

	return sb.String()
}

// DAGOverviewToMermaid renders the condensed DAG as a Mermaid flowchart.
// SCCs are collapsed to single nodes labeled with member names.
func DAGOverviewToMermaid(g *Graph, cycles *CycleAnalysis, maxNodes int) string {
	dag := cycles.DAGCondensation
	if dag == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("graph TD\n")

	shown := 0
	for i, scc := range dag.Nodes {
		if maxNodes > 0 && shown >= maxNodes {
			break
		}
		id := fmt.Sprintf("scc%d", i)
		if scc.IsTrivial {
			sb.WriteString(fmt.Sprintf("  %s[%s]\n", id, scc.NodeIDs[0]))
		} else {
			label := strings.Join(scc.NodeIDs, ", ")
			if len(label) > 40 {
				label = label[:37] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %s{{\"%s\"}}\n", id, label))
		}
		shown++
	}

	for _, e := range dag.Edges {
		if maxNodes > 0 && (e[0] >= maxNodes || e[1] >= maxNodes) {
			continue
		}
		sb.WriteString(fmt.Sprintf("  scc%d --> scc%d\n", e[0], e[1]))
	}

	return sb.String()
}

// RenderASCIIGraph renders a set of nodes and edges as ASCII art.
// It uses the condensed DAG layers for layout.
func RenderASCIIGraph(dag *CondensedDAG, width int) string {
	if dag == nil || len(dag.Nodes) == 0 {
		return "  (no schemas)"
	}

	var sb strings.Builder
	maxBoxWidth := 0

	// Pre-compute labels and find max width
	type nodeLabel struct {
		sccIdx int
		label  string
		isSCC  bool
	}
	var allLabels [][]nodeLabel
	for _, layer := range dag.Layers {
		var layerLabels []nodeLabel
		for _, sccIdx := range layer {
			if sccIdx >= len(dag.Nodes) {
				continue
			}
			scc := dag.Nodes[sccIdx]
			var label string
			isSCC := !scc.IsTrivial
			if scc.IsTrivial {
				label = scc.NodeIDs[0]
			} else {
				label = strings.Join(scc.NodeIDs, ", ")
				if len(label) > 35 {
					label = label[:32] + "..."
				}
				label = fmt.Sprintf("SCC: %s", label)
			}
			if len(label)+4 > maxBoxWidth {
				maxBoxWidth = len(label) + 4
			}
			layerLabels = append(layerLabels, nodeLabel{sccIdx: sccIdx, label: label, isSCC: isSCC})
		}
		allLabels = append(allLabels, layerLabels)
	}

	if maxBoxWidth > width-4 {
		maxBoxWidth = width - 4
	}
	if maxBoxWidth < 10 {
		maxBoxWidth = 10
	}

	// Build edge lookup: from SCC index -> list of to SCC indices
	edgeLookup := make(map[int][]int)
	for _, e := range dag.Edges {
		edgeLookup[e[0]] = append(edgeLookup[e[0]], e[1])
	}

	for layerIdx, layerLabels := range allLabels {
		for _, nl := range layerLabels {
			boxW := len(nl.label) + 4
			if boxW > maxBoxWidth {
				boxW = maxBoxWidth
			}

			if nl.isSCC {
				// Double-border for SCC
				sb.WriteString(fmt.Sprintf("  ╔%s╗\n", strings.Repeat("═", boxW-2)))
				content := nl.label
				if len(content) > boxW-4 {
					content = content[:boxW-7] + "..."
				}
				sb.WriteString(fmt.Sprintf("  ║ %-*s ║\n", boxW-4, content))
				sb.WriteString(fmt.Sprintf("  ╚%s╝\n", strings.Repeat("═", boxW-2)))
			} else {
				// Single border for regular node
				sb.WriteString(fmt.Sprintf("  ┌%s┐\n", strings.Repeat("─", boxW-2)))
				content := nl.label
				if len(content) > boxW-4 {
					content = content[:boxW-7] + "..."
				}
				sb.WriteString(fmt.Sprintf("  │ %-*s │\n", boxW-4, content))
				sb.WriteString(fmt.Sprintf("  └%s┘\n", strings.Repeat("─", boxW-2)))
			}

			// Draw edges to next layers
			targets := edgeLookup[nl.sccIdx]
			if len(targets) > 0 {
				sort.Ints(targets)
				var targetNames []string
				for _, t := range targets {
					if t < len(dag.Nodes) {
						targetNames = append(targetNames, sccLabel(dag.Nodes[t]))
					}
				}
				sb.WriteString(fmt.Sprintf("    ╰─→ %s\n", strings.Join(targetNames, ", ")))
			}
		}

		if layerIdx < len(allLabels)-1 {
			sb.WriteString("    │\n")
			sb.WriteString("    ▼\n")
		}
	}

	return sb.String()
}

// RenderASCIIEgoGraph renders a BFS ego graph as ASCII art.
func RenderASCIIEgoGraph(g *Graph, centerID string, hops int, width int) string {
	if _, ok := g.Nodes[centerID]; !ok {
		return fmt.Sprintf("  Schema %q not found", centerID)
	}

	// BFS to collect nodes within hop radius
	visited := map[string]int{centerID: 0}
	queue := []string{centerID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		dist := visited[current]
		if dist >= hops {
			continue
		}
		for _, e := range g.OutEdges[current] {
			if _, seen := visited[e.To]; !seen {
				visited[e.To] = dist + 1
				queue = append(queue, e.To)
			}
		}
		for _, e := range g.InEdges[current] {
			if _, seen := visited[e.From]; !seen {
				visited[e.From] = dist + 1
				queue = append(queue, e.From)
			}
		}
	}

	// Group by distance
	byDist := make(map[int][]string)
	maxDist := 0
	for id, dist := range visited {
		byDist[dist] = append(byDist[dist], id)
		if dist > maxDist {
			maxDist = dist
		}
	}

	var sb strings.Builder

	// Center node
	sb.WriteString(fmt.Sprintf("  ╔%s╗\n", strings.Repeat("═", len(centerID)+2)))
	sb.WriteString(fmt.Sprintf("  ║ %s ║\n", centerID))
	sb.WriteString(fmt.Sprintf("  ╚%s╝\n", strings.Repeat("═", len(centerID)+2)))

	// Outgoing edges from center with labels
	type edgeInfo struct {
		target string
		label  string
	}
	var outEdges []edgeInfo
	for _, e := range g.OutEdges[centerID] {
		if _, ok := visited[e.To]; ok {
			label := string(e.Kind)
			if e.FieldName != "" {
				label += ":" + e.FieldName
			}
			var flags []string
			if e.IsRequired {
				flags = append(flags, "req")
			}
			if e.IsNullable {
				flags = append(flags, "null")
			}
			if e.IsArray {
				flags = append(flags, "[]")
			}
			if len(flags) > 0 {
				label += " [" + strings.Join(flags, ",") + "]"
			}
			outEdges = append(outEdges, edgeInfo{target: e.To, label: label})
		}
	}
	if len(outEdges) > 0 {
		sb.WriteString("    │ references\n")
		sb.WriteString("    ▼\n")
		for _, ei := range outEdges {
			sb.WriteString(fmt.Sprintf("    ├─→ %-20s  %s\n", ei.target, ei.label))
		}
	}

	// Incoming edges to center with labels
	var inEdges []edgeInfo
	for _, e := range g.InEdges[centerID] {
		if _, ok := visited[e.From]; ok {
			label := string(e.Kind)
			if e.FieldName != "" {
				label += ":" + e.FieldName
			}
			var flags []string
			if e.IsRequired {
				flags = append(flags, "req")
			}
			if e.IsArray {
				flags = append(flags, "[]")
			}
			if len(flags) > 0 {
				label += " [" + strings.Join(flags, ",") + "]"
			}
			inEdges = append(inEdges, edgeInfo{target: e.From, label: label})
		}
	}
	if len(inEdges) > 0 {
		sb.WriteString("    │ referenced by\n")
		sb.WriteString("    ▲\n")
		for _, ei := range inEdges {
			sb.WriteString(fmt.Sprintf("    ├── %-20s  %s\n", ei.target, ei.label))
		}
	}

	// Distant neighbors (hops > 1)
	for dist := 2; dist <= maxDist; dist++ {
		nodes := byDist[dist]
		if len(nodes) == 0 {
			continue
		}
		sort.Strings(nodes)
		sb.WriteString(fmt.Sprintf("\n    %d hops: %s\n", dist, strings.Join(nodes, ", ")))
	}

	return sb.String()
}

// RenderASCIISCC renders an SCC with its internal edges as ASCII.
func RenderASCIISCC(g *Graph, cycles *CycleAnalysis, sccIndex int) string {
	if sccIndex < 0 || sccIndex >= len(cycles.SCCs) {
		return ""
	}
	scc := cycles.SCCs[sccIndex]
	memberSet := make(map[string]bool, scc.Size)
	for _, id := range scc.NodeIDs {
		memberSet[id] = true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  SCC #%d (%d schemas)\n", sccIndex+1, scc.Size))
	sb.WriteString(fmt.Sprintf("  %s\n\n", strings.Repeat("─", 30)))

	for _, id := range scc.NodeIDs {
		sb.WriteString(fmt.Sprintf("  [%s]\n", id))
		// Show edges to other SCC members
		for _, e := range g.OutEdges[id] {
			if memberSet[e.To] {
				label := string(e.Kind)
				if e.FieldName != "" {
					label += ":" + e.FieldName
				}
				if e.IsRequired {
					label += " (req)"
				}
				sb.WriteString(fmt.Sprintf("    └─→ %s  via %s\n", e.To, label))
			}
		}
	}

	return sb.String()
}

func mermaidSafeID(id string) string {
	// Replace characters that aren't safe in mermaid IDs
	r := strings.NewReplacer("-", "_", ".", "_", " ", "_")
	return r.Replace(id)
}

func mermaidEdgeLabel(e *Edge) string {
	var parts []string
	parts = append(parts, string(e.Kind))
	if e.FieldName != "" {
		parts = append(parts, e.FieldName)
	}
	return strings.Join(parts, ":")
}

func sccLabel(scc *SCC) string {
	if scc.IsTrivial {
		return scc.NodeIDs[0]
	}
	if len(scc.NodeIDs) <= 3 {
		return "{" + strings.Join(scc.NodeIDs, ", ") + "}"
	}
	return fmt.Sprintf("{%s +%d}", scc.NodeIDs[0], scc.Size-1)
}
