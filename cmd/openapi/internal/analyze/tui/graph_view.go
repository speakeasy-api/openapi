package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
)

// Graph mode constants.
const (
	GraphModeDAG   = 0
	GraphModeSCC   = 1
	GraphModeEgo   = 2
	graphModeCount = 3
)

var graphModeNames = []string{"DAG overview", "SCC gallery", "Ego graph"}

// renderGraphView dispatches to the appropriate graph rendering mode.
func (m Model) renderGraphView() string {
	var sb strings.Builder

	// Mode indicator bar
	sb.WriteString(m.renderGraphModeBar())
	sb.WriteString("\n")

	switch m.graphMode {
	case GraphModeDAG:
		sb.WriteString(m.renderGraphDAG())
	case GraphModeSCC:
		sb.WriteString(m.renderGraphSCCGallery())
	case GraphModeEgo:
		sb.WriteString(m.renderGraphEgo())
	}

	// Selectable node list
	sb.WriteString(m.renderGraphNodeList())

	return sb.String()
}

func (m Model) renderGraphModeBar() string {
	var tabs []string
	for i, name := range graphModeNames {
		label := name
		switch i {
		case GraphModeSCC:
			sccCount := len(m.report.Cycles.SCCs)
			if sccCount > 0 {
				label = fmt.Sprintf("%s (%d/%d)", name, m.graphSCCIdx+1, sccCount)
			} else {
				label = fmt.Sprintf("%s (none)", name)
			}
		case GraphModeEgo:
			if m.graphEgoNode != "" {
				label = fmt.Sprintf("%s: %s (%d hops)", name, m.graphEgoNode, m.graphEgoHops)
			}
		}

		if i == m.graphMode {
			tabs = append(tabs, GraphModeActive.Render(label))
		} else {
			tabs = append(tabs, GraphModeInactive.Render(label))
		}
	}

	return "  " + strings.Join(tabs, "  ") + "\n"
}

func (m Model) renderGraphDAG() string {
	dag := m.report.Cycles.DAGCondensation
	if dag == nil || len(dag.Nodes) == 0 {
		return StatLabel.Render("  No schemas to display.") + "\n"
	}

	var sb strings.Builder
	sb.WriteString(StatLabel.Render(fmt.Sprintf("  %d nodes, %d edges, %d layers",
		len(dag.Nodes), len(dag.Edges), dag.Depth)) + "\n\n")

	sb.WriteString(analyze.RenderASCIIGraph(dag, m.width))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderGraphSCCGallery() string {
	sccs := m.report.Cycles.SCCs
	if len(sccs) == 0 {
		return GreenBadge.Render("  No SCCs found — graph is acyclic!") + "\n"
	}

	idx := m.graphSCCIdx
	if idx >= len(sccs) {
		idx = len(sccs) - 1
	}

	var sb strings.Builder

	// SCC ASCII rendering
	ascii := analyze.RenderASCIISCC(m.report.Graph, m.report.Cycles, idx)
	sb.WriteString(ascii)

	// Cycles in this SCC
	scc := sccs[idx]
	memberSet := make(map[string]bool, scc.Size)
	for _, id := range scc.NodeIDs {
		memberSet[id] = true
	}

	sb.WriteString("\n")
	sb.WriteString(StatLabel.Render("  Cycles through this SCC:") + "\n")
	cycleCount := 0
	for _, c := range m.report.Cycles.Cycles {
		if len(c.Path) > 0 && memberSet[c.Path[0]] {
			cycleCount++
			severity := GreenBadge.Render("optional")
			if c.HasRequiredOnlyPath {
				severity = RedBadge.Render("required-only")
			}
			path := strings.Join(c.Path, " -> ") + " -> " + c.Path[0]
			if len(path) > m.width-20 {
				path = path[:m.width-23] + "..."
			}
			sb.WriteString(fmt.Sprintf("    %s  %s\n", severity, path))
		}
	}
	if cycleCount == 0 {
		sb.WriteString(StatLabel.Render("    (self-loop only)") + "\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderGraphEgo() string {
	if m.graphEgoNode == "" {
		return StatLabel.Render("  Select a schema and press Enter to view its ego graph.\n  Or navigate nodes below with j/k and Enter.") + "\n"
	}

	var sb strings.Builder
	ascii := analyze.RenderASCIIEgoGraph(m.report.Graph, m.graphEgoNode, m.graphEgoHops, m.width)
	sb.WriteString(ascii)
	sb.WriteString("\n")

	return sb.String()
}

// renderGraphNodeList renders a selectable list of schemas below the graph art.
func (m Model) renderGraphNodeList() string {
	if len(m.graphItems) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("  " + strings.Repeat("─", min(m.width-4, 60)) + "\n")

	label := "Schemas"
	switch m.graphMode {
	case GraphModeSCC:
		label = "SCC members"
	case GraphModeEgo:
		label = "Neighborhood"
	}
	sb.WriteString(StatLabel.Render(fmt.Sprintf("  %s (%d)  j/k:navigate  enter:focus", label, len(m.graphItems))) + "\n")

	// Show a window of items around the cursor
	maxVisible := m.contentHeight() / 2
	if maxVisible < 5 {
		maxVisible = 5
	}
	start := m.graphCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.graphItems) {
		end = len(m.graphItems)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	if start > 0 {
		sb.WriteString(ScrollIndicatorStyle.Render("    ... more above") + "\n")
	}

	for i := start; i < end; i++ {
		id := m.graphItems[i]
		sm := m.report.Metrics[id]
		d := m.report.Codegen.PerSchema[id]

		prefix := "  "
		style := NormalRow
		if i == m.graphCursor {
			prefix = "> "
			style = SelectedRow
		}

		// Tier badge
		tier := ""
		if d != nil {
			switch d.Tier {
			case analyze.CodegenGreen:
				tier = GreenBadge.Render("G")
			case analyze.CodegenYellow:
				tier = YellowBadge.Render("Y")
			case analyze.CodegenRed:
				tier = RedBadge.Render("R")
			}
		}

		// Score and fan info
		info := ""
		if sm != nil {
			info = fmt.Sprintf("score=%-3d in=%d out=%d", sm.ComplexityScore(), sm.FanIn, sm.FanOut)
		}

		// Highlight center node in ego mode
		marker := ""
		if m.graphMode == GraphModeEgo && id == m.graphEgoNode {
			marker = StatHighlight.Render(" *")
		}

		line := fmt.Sprintf("%s  %s %-24s %s%s", prefix, tier, truncate(id, 24), info, marker)
		sb.WriteString(style.Render(line) + "\n")
	}

	if end < len(m.graphItems) {
		sb.WriteString(ScrollIndicatorStyle.Render("    ... more below") + "\n")
	}

	return sb.String()
}

// Graph mode indicator styles.
var (
	GraphModeActive = lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color(colorBlue)).
			Foreground(lipgloss.Color(colorWhite)).
			Bold(true)

	GraphModeInactive = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color(colorGray))
)
