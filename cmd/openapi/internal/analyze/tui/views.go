package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
)

func (m Model) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, ActiveTab.Render(name))
		} else {
			tabs = append(tabs, InactiveTab.Render(name))
		}
	}

	title := TitleStyle.Render("Schema Analyzer")
	docInfo := SubtitleStyle.Render(fmt.Sprintf(" %s v%s", m.report.DocumentTitle, m.report.DocumentVersion))

	return title + docInfo + "\n" + strings.Join(tabs, " ") + "\n\n"
}

func (m Model) renderSummary() string {
	r := m.report
	var s strings.Builder

	// Overview card
	s.WriteString(CardTitleStyle.Render("Overview") + "\n")
	s.WriteString(fmt.Sprintf("  %s %s   %s %s   %s %s\n",
		StatLabel.Render("Schemas:"), StatValue.Render(fmt.Sprintf("%d", r.TotalSchemas)),
		StatLabel.Render("Refs:"), StatValue.Render(fmt.Sprintf("%d", r.TotalEdges)),
		StatLabel.Render("OpenAPI:"), StatValue.Render(r.OpenAPIVersion),
	))
	s.WriteString("\n")

	// Cycle health
	s.WriteString(CardTitleStyle.Render("Cycle Health") + "\n")
	sccLabel := fmt.Sprintf("%d", r.SCCCount)
	if r.SCCCount == 0 {
		sccLabel = GreenBadge.Render("0 (none)")
	} else {
		sccLabel = StatWarning.Render(sccLabel)
	}
	s.WriteString(fmt.Sprintf("  %s %s   %s %s   %s %s\n",
		StatLabel.Render("SCCs:"), sccLabel,
		StatLabel.Render("Largest:"), formatSCCSize(r.LargestSCCSize),
		StatLabel.Render("Cycles:"), formatCycleCount(len(r.Cycles.Cycles)),
	))
	cyclesPctLabel := fmt.Sprintf("%.0f%%", r.SchemasInCyclesPct)
	if r.SchemasInCyclesPct == 0 {
		cyclesPctLabel = GreenBadge.Render("0%")
	} else if r.SchemasInCyclesPct > 30 {
		cyclesPctLabel = RedBadge.Render(cyclesPctLabel)
	} else {
		cyclesPctLabel = YellowBadge.Render(cyclesPctLabel)
	}
	reqCyclesLabel := fmt.Sprintf("%d", r.RequiredOnlyCycles)
	if r.RequiredOnlyCycles > 0 {
		reqCyclesLabel = RedBadge.Render(reqCyclesLabel)
	} else {
		reqCyclesLabel = GreenBadge.Render("0")
	}
	s.WriteString(fmt.Sprintf("  %s %s   %s %s   %s %d\n",
		StatLabel.Render("In cycles:"), cyclesPctLabel,
		StatLabel.Render("Required-only:"), reqCyclesLabel,
		StatLabel.Render("DAG depth:"), r.DAGDepth,
	))
	s.WriteString("\n")

	// Codegen compatibility
	s.WriteString(CardTitleStyle.Render("Codegen Compatibility") + "\n")
	scoreStr := fmt.Sprintf("%.0f%%", r.CompatibilityScore)
	if r.CompatibilityScore >= 80 {
		scoreStr = GreenBadge.Render(scoreStr)
	} else if r.CompatibilityScore >= 50 {
		scoreStr = YellowBadge.Render(scoreStr)
	} else {
		scoreStr = RedBadge.Render(scoreStr)
	}
	s.WriteString(fmt.Sprintf("  %s %s   %s %s  %s %s  %s %s\n",
		StatLabel.Render("Score:"), scoreStr,
		StatLabel.Render("Green:"), GreenBadge.Render(fmt.Sprintf("%d", r.Codegen.GreenCount)),
		StatLabel.Render("Yellow:"), YellowBadge.Render(fmt.Sprintf("%d", r.Codegen.YellowCount)),
		StatLabel.Render("Red:"), RedBadge.Render(fmt.Sprintf("%d", r.Codegen.RedCount)),
	))
	s.WriteString("\n")

	// Compatibility bar
	s.WriteString("  " + renderBar(r.Codegen.GreenCount, r.Codegen.YellowCount, r.Codegen.RedCount, m.width-6) + "\n\n")

	// Top schemas by fan-in
	if len(r.TopFanIn) > 0 {
		s.WriteString(CardTitleStyle.Render("Highest Fan-In (most referenced)") + "\n")
		for i, sm := range r.TopFanIn {
			if sm.FanIn == 0 {
				break
			}
			s.WriteString(fmt.Sprintf("  %d. %-30s %s\n", i+1, sm.NodeID, StatHighlight.Render(fmt.Sprintf("%d refs", sm.FanIn))))
		}
		s.WriteString("\n")
	}

	// Top schemas by complexity
	if len(r.TopComplex) > 0 {
		s.WriteString(CardTitleStyle.Render("Most Complex Schemas") + "\n")
		for i, sm := range r.TopComplex {
			score := sm.ComplexityScore()
			if score == 0 {
				break
			}
			tier := tierBadge(m.report.Codegen.PerSchema[sm.NodeID])
			detail := fmt.Sprintf("fan-in:%d fan-out:%d props:%d", sm.FanIn, sm.FanOut, sm.DeepPropertyCount)
			if sm.VariantProduct > 1 {
				detail += fmt.Sprintf(" variants:%d", sm.VariantProduct)
			}
			s.WriteString(fmt.Sprintf("  %d. %-28s %s  %s  %s\n",
				i+1, sm.NodeID, tier, StatHighlight.Render(fmt.Sprintf("score=%d", score)), detail))
		}
		s.WriteString("\n")
	}

	// Top suggestions
	if len(r.Suggestions) > 0 {
		limit := 5
		if len(r.Suggestions) < limit {
			limit = len(r.Suggestions)
		}
		s.WriteString(CardTitleStyle.Render("Top Suggestions") + "\n")
		for i := 0; i < limit; i++ {
			sg := r.Suggestions[i]
			s.WriteString(SuggestionStyle.Render(fmt.Sprintf("  %d. %s (impact: %d)", i+1, sg.Title, sg.Impact)) + "\n")
		}
	}

	return s.String()
}

func (m Model) renderSchemaList() string {
	var s strings.Builder

	// Filter indicator
	filterLabel := "all"
	switch m.schemaFilter {
	case "yellow":
		filterLabel = YellowBadge.Render("yellow+red")
	case "red":
		filterLabel = RedBadge.Render("red only")
	}
	sortLabel := schemaSortModes[m.schemaSortMode]
	s.WriteString(StatLabel.Render(fmt.Sprintf("  Filter: %s  Sort: %s  (%d schemas)", filterLabel, sortLabel, len(m.schemaItems))) + "\n\n")

	// Header — pad before styling so ANSI escapes don't break alignment
	s.WriteString(fmt.Sprintf("  %s %s %s %s %s %s\n",
		StatLabel.Render(fmt.Sprintf("%-30s", "Schema")),
		StatLabel.Render(fmt.Sprintf("%5s", "Score")),
		StatLabel.Render(fmt.Sprintf("%6s", "Fan-In")),
		StatLabel.Render(fmt.Sprintf("%7s", "Fan-Out")),
		StatLabel.Render(fmt.Sprintf("%5s", "Props")),
		StatLabel.Render("Tier"),
	))
	s.WriteString("  " + strings.Repeat("-", min(m.width-4, 70)) + "\n")

	contentH := m.contentHeight() - 4 // header rows
	linesRendered := 0

	for i := m.scrollOffset; i < len(m.schemaItems) && linesRendered < contentH; i++ {
		id := m.schemaItems[i]
		sm := m.report.Metrics[id]
		d := m.report.Codegen.PerSchema[id]

		prefix := "  "
		rowStyle := NormalRow
		if i == m.cursor {
			prefix = "> "
			rowStyle = SelectedRow
		}

		tier := tierBadge(d)
		line := fmt.Sprintf("%s%-30s %5d %6d %7d %5d %s",
			prefix, truncate(id, 30), sm.ComplexityScore(), sm.FanIn, sm.FanOut, sm.PropertyCount, tier)
		s.WriteString(rowStyle.Render(line) + "\n")
		linesRendered++

		// Expanded detail — bordered card
		if m.expanded[i] {
			card := m.renderSchemaCard(id)
			s.WriteString(card)
			linesRendered += strings.Count(card, "\n")
		}
	}

	if m.scrollOffset > 0 {
		s.WriteString(ScrollIndicatorStyle.Render("  ... more above") + "\n")
	}
	if m.scrollOffset+contentH < len(m.schemaItems) {
		s.WriteString(ScrollIndicatorStyle.Render("  ... more below") + "\n")
	}

	return s.String()
}


func (m Model) renderCycleList() string {
	var s strings.Builder

	cycles := m.report.Cycles.Cycles
	if len(cycles) == 0 {
		s.WriteString(GreenBadge.Render("  No cycles detected!") + "\n")
		return s.String()
	}

	s.WriteString(StatLabel.Render(fmt.Sprintf("  %d cycles found  (%d required-only)", len(cycles), m.report.RequiredOnlyCycles)) + "\n\n")

	contentH := m.contentHeight() - 2
	linesRendered := 0

	for i := m.scrollOffset; i < len(cycles) && linesRendered < contentH; i++ {
		c := cycles[i]

		prefix := "  "
		rowStyle := NormalRow
		if i == m.cursor {
			prefix = "> "
			rowStyle = SelectedRow
		}

		severity := GreenBadge.Render("optional")
		if c.HasRequiredOnlyPath {
			severity = RedBadge.Render("required-only")
		} else if len(c.BreakPoints) == 0 {
			severity = YellowBadge.Render("no-break-point")
		}

		line := fmt.Sprintf("%sCycle %d  len:%d  %s  %s",
			prefix, i+1, c.Length, severity, formatCyclePath(c))
		s.WriteString(rowStyle.Render(line) + "\n")
		linesRendered++

		if m.expanded[i] {
			s.WriteString(m.renderCycleDetail(c))
			linesRendered += c.Length + 3
		}
	}

	return s.String()
}

func (m Model) renderCycleDetail(c *analyze.Cycle) string {
	var s strings.Builder

	s.WriteString(DetailStyle.Render("    Path:") + "\n")
	for j, nodeID := range c.Path {
		var edge *analyze.Edge
		if j < len(c.Edges) {
			edge = c.Edges[j]
		}

		nodeStr := "    " + nodeID
		if edge != nil {
			edgeLabel := formatEdgeLabel(edge)
			isBreak := false
			for _, bp := range c.BreakPoints {
				if bp == edge {
					isBreak = true
					break
				}
			}
			if isBreak {
				nodeStr += " " + GreenBadge.Render("--"+edgeLabel+"--> ") + GreenBadge.Render("[cut here]")
			} else {
				nodeStr += " " + RequiredEdge.Render("--"+edgeLabel+"-->")
			}
		} else {
			// Last node connects back to first
			nodeStr += " " + StatLabel.Render("(back to "+c.Path[0]+")")
		}

		s.WriteString(DetailStyle.Render(nodeStr) + "\n")
	}

	if len(c.BreakPoints) > 0 {
		s.WriteString(SuggestionStyle.Render(fmt.Sprintf("    Suggestion: %d break point(s) available", len(c.BreakPoints))) + "\n")
	} else if c.HasRequiredOnlyPath {
		s.WriteString(RedBadge.Render("    Warning: No natural break points — all edges required") + "\n")
	}
	s.WriteString("\n")

	return s.String()
}


func (m Model) renderSuggestionList() string {
	var s strings.Builder

	suggestions := m.report.Suggestions
	if len(suggestions) == 0 {
		s.WriteString(GreenBadge.Render("  No suggestions — the schema graph looks good!") + "\n")
		return s.String()
	}

	s.WriteString(StatLabel.Render(fmt.Sprintf("  %d suggestions (sorted by impact)", len(suggestions))) + "\n\n")

	contentH := m.contentHeight() - 2
	linesRendered := 0

	for i := m.scrollOffset; i < len(suggestions) && linesRendered < contentH; i++ {
		sg := suggestions[i]

		prefix := "  "
		rowStyle := NormalRow
		if i == m.cursor {
			prefix = "> "
			rowStyle = SelectedRow
		}

		typeLabel := StatLabel.Render("[" + string(sg.Type) + "]")
		line := fmt.Sprintf("%s%s %s  %s",
			prefix, typeLabel, sg.Title, StatHighlight.Render(fmt.Sprintf("impact=%d", sg.Impact)))
		s.WriteString(rowStyle.Render(line) + "\n")
		linesRendered++

		if m.expanded[i] {
			// Description
			s.WriteString(DetailStyle.Render("    "+sg.Description) + "\n")
			linesRendered++

			// Affected schemas
			if len(sg.AffectedSchemas) > 0 {
				schemas := strings.Join(sg.AffectedSchemas, ", ")
				s.WriteString(StatLabel.Render("    Schemas: ") + StatValue.Render(schemas) + "\n")
				linesRendered++
			}
			s.WriteString("\n")
			linesRendered++
		}
	}

	if m.scrollOffset > 0 {
		s.WriteString(ScrollIndicatorStyle.Render("  ... more above") + "\n")
	}
	if m.scrollOffset+contentH < len(suggestions) {
		s.WriteString(ScrollIndicatorStyle.Render("  ... more below") + "\n")
	}

	return s.String()
}

func (m Model) renderFooter() string {
	var parts []string
	parts = append(parts, "q:quit")
	parts = append(parts, "tab:next view")
	parts = append(parts, "1-5:jump to view")
	parts = append(parts, "?:help")

	switch m.activeTab {
	case TabSchemas:
		parts = append(parts, "f:filter tier")
		parts = append(parts, "s:sort")
		parts = append(parts, "enter:expand")
	case TabCycles:
		parts = append(parts, "enter:expand")
	case TabSuggestions:
		parts = append(parts, "enter:expand")
	case TabGraph:
		parts = append(parts, "j/k:navigate")
		parts = append(parts, "enter:focus")
		parts = append(parts, "m:mode")
		if m.graphMode == GraphModeSCC {
			parts = append(parts, "n/p:SCC")
		}
		if m.graphMode == GraphModeEgo {
			parts = append(parts, "+/-:hops")
		}
	}

	return FooterStyle.Width(m.width).Render(strings.Join(parts, "  ")) + "\n"
}

func (m Model) renderHelp() string {
	var s strings.Builder

	s.WriteString(HelpTitleStyle.Render("Schema Analyzer Help") + "\n\n")

	helpItems := []struct{ key, desc string }{
		{"q / Ctrl-C", "Quit"},
		{"Tab / l", "Next tab"},
		{"Shift-Tab / h", "Previous tab"},
		{"1-5", "Jump to tab"},
		{"j / Down", "Move down"},
		{"k / Up", "Move up"},
		{"gg", "Jump to top"},
		{"G", "Jump to bottom"},
		{"Ctrl-D", "Scroll down half page"},
		{"Ctrl-U", "Scroll up half page"},
		{"Enter / Space", "Expand/collapse or focus node"},
		{"f", "Cycle tier filter (schemas tab)"},
		{"s", "Cycle sort mode (schemas tab)"},
		{"m", "Cycle graph mode (graph tab)"},
		{"n / p", "Next/prev SCC (SCC gallery)"},
		{"+ / -", "Increase/decrease ego hops"},
		{"Enter", "Focus node → ego graph (graph)"},
		{"?", "Toggle help"},
	}

	for _, item := range helpItems {
		s.WriteString(fmt.Sprintf("  %s  %s\n",
			HelpKeyStyle.Render(fmt.Sprintf("%-14s", item.key)),
			HelpTextStyle.Render(item.desc)))
	}

	return HelpModalStyle.Render(s.String())
}

// --- Helpers ---

func tierBadge(d *analyze.CodegenDifficulty) string {
	if d == nil {
		return StatLabel.Render("-")
	}
	switch d.Tier {
	case analyze.CodegenGreen:
		return GreenBadge.Render("GREEN")
	case analyze.CodegenYellow:
		return YellowBadge.Render("YELLOW")
	case analyze.CodegenRed:
		return RedBadge.Render("RED")
	}
	return "-"
}

func formatEdgeLabel(e *analyze.Edge) string {
	label := string(e.Kind)
	if e.FieldName != "" {
		label += ":" + e.FieldName
	}
	if e.IsRequired {
		label += " [req]"
	}
	if e.IsArray {
		label += " []"
	}
	return label
}

func formatCyclePath(c *analyze.Cycle) string {
	if len(c.Path) == 0 {
		return ""
	}
	path := strings.Join(c.Path, " -> ")
	path += " -> " + c.Path[0]
	if len(path) > 60 {
		path = path[:57] + "..."
	}
	return path
}

func formatSCCSize(size int) string {
	if size == 0 {
		return GreenBadge.Render("0")
	}
	if size > 5 {
		return RedBadge.Render(fmt.Sprintf("%d", size))
	}
	return YellowBadge.Render(fmt.Sprintf("%d", size))
}

func formatCycleCount(count int) string {
	if count == 0 {
		return GreenBadge.Render("0")
	}
	return StatWarning.Render(fmt.Sprintf("%d", count))
}

func renderBar(green, yellow, red, width int) string {
	total := green + yellow + red
	if total == 0 || width <= 0 {
		return ""
	}

	gw := green * width / total
	yw := yellow * width / total
	rw := width - gw - yw

	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen)).Render(strings.Repeat("█", gw)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow)).Render(strings.Repeat("█", yw)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed)).Render(strings.Repeat("█", rw))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
