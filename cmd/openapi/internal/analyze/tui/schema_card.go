package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
)

// renderSchemaCard renders a bordered detail card for a schema.
func (m Model) renderSchemaCard(nodeID string) string {
	sm := m.report.Metrics[nodeID]
	d := m.report.Codegen.PerSchema[nodeID]
	node := m.report.Graph.Nodes[nodeID]
	if sm == nil || node == nil {
		return ""
	}

	// Determine card width
	cardWidth := m.width - 8
	if cardWidth > 64 {
		cardWidth = 64
	}
	if cardWidth < 30 {
		cardWidth = 30
	}
	innerWidth := cardWidth - 4 // padding inside border

	var content strings.Builder

	// Pick border color based on tier
	borderColor := colorGreen
	tierEmoji := "🟢"
	tierName := "green"
	if d != nil {
		tierName = strings.ToLower(d.Tier.String())
		switch d.Tier {
		case analyze.CodegenYellow:
			borderColor = colorYellow
			tierEmoji = "🟡"
		case analyze.CodegenRed:
			borderColor = colorRed
			tierEmoji = "🔴"
		}
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(borderColor))

	// Title line inside the card
	content.WriteString(titleStyle.Render(nodeID) + "\n")

	// Tier + score + rank
	ranked := analyze.TopSchemasByComplexity(m.report.Metrics, len(m.report.Metrics))
	rank := 0
	for i, r := range ranked {
		if r.NodeID == nodeID {
			rank = i + 1
			break
		}
	}
	content.WriteString(fmt.Sprintf("%s %s    %s %s    %s %s\n",
		StatLabel.Render("Tier:"), StatValue.Render(tierEmoji+" "+tierName),
		StatLabel.Render("Score:"), StatValue.Render(fmt.Sprintf("%d", sm.ComplexityScore())),
		StatLabel.Render("Rank:"), StatValue.Render(fmt.Sprintf("#%d", rank))))
	content.WriteString("\n")

	// Types + nullable
	types := strings.Join(node.Types, ", ")
	if types == "" {
		types = "(none)"
	}
	typeLine := fmt.Sprintf("%s %s", StatLabel.Render("Types:"), StatValue.Render(types))
	if node.IsNullable {
		typeLine += "  " + YellowBadge.Render("nullable")
	}
	content.WriteString(typeLine + "\n")

	// Properties: N required / M total (deep: D)
	propLine := fmt.Sprintf("%s %s",
		StatLabel.Render("Properties:"),
		StatValue.Render(fmt.Sprintf("%d required / %d total", sm.RequiredCount, sm.PropertyCount)))
	if sm.DeepPropertyCount != sm.PropertyCount {
		propLine += StatLabel.Render(fmt.Sprintf(" (deep: %d)", sm.DeepPropertyCount))
	}
	content.WriteString(propLine + "\n")

	// Fan-in / Fan-out
	content.WriteString(fmt.Sprintf("%s %s  %s %s\n",
		StatLabel.Render("Fan-in:"), StatValue.Render(fmt.Sprintf("%d", sm.FanIn)),
		StatLabel.Render("Fan-out:"), StatValue.Render(fmt.Sprintf("%d", sm.FanOut))))
	content.WriteString("\n")

	// Complexity section
	content.WriteString(StatValue.Render("Complexity") + "\n")
	content.WriteString(fmt.Sprintf("  %s %s  %s %s  %s %s\n",
		StatLabel.Render("Nesting:"), StatValue.Render(fmt.Sprintf("%d", sm.NestingDepth)),
		StatLabel.Render("Composition:"), StatValue.Render(fmt.Sprintf("%d", sm.CompositionDepth)),
		StatLabel.Render("Unions:"), StatValue.Render(fmt.Sprintf("%d", sm.UnionSiteCount))))
	if sm.UnionSiteCount > 0 {
		variantLine := fmt.Sprintf("  %s %s",
			StatLabel.Render("Max width:"), StatValue.Render(fmt.Sprintf("%d", sm.MaxUnionWidth)))
		if sm.VariantProduct > 1 {
			variantLine += fmt.Sprintf("  %s %s",
				StatLabel.Render("Variant product:"),
				StatWarning.Render(fmt.Sprintf("%d", sm.VariantProduct)))
		}
		if sm.HasDiscriminator {
			variantLine += "  " + GreenBadge.Render("discriminated")
		}
		content.WriteString(variantLine + "\n")
	}

	// Composition keywords
	if len(node.CompositionFields) > 0 {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			StatLabel.Render("Keywords:"),
			StatValue.Render(strings.Join(node.CompositionFields, ", "))))
	}
	content.WriteString("\n")

	// Cycle membership
	if sm.InSCC {
		cycleLine := fmt.Sprintf("%s %s",
			StatLabel.Render("Cycles:"),
			StatWarning.Render(fmt.Sprintf("member of %d cycle(s)", sm.CycleMembership)))
		content.WriteString(cycleLine + "\n\n")
	}

	// Signals with full descriptions
	if d != nil && len(d.Signals) > 0 {
		content.WriteString(StatValue.Render("Signals") + "\n")
		for _, sig := range d.Signals {
			icon := "  "
			switch sig.Severity {
			case analyze.CodegenRed:
				icon = RedBadge.Render("!")
			case analyze.CodegenYellow:
				icon = YellowBadge.Render("~")
			}
			desc := sig.Description
			if innerWidth > 6 && len(desc) > innerWidth-6 {
				desc = desc[:innerWidth-9] + "..."
			}
			content.WriteString(fmt.Sprintf("  %s %s\n", icon, StatLabel.Render(desc)))
		}
		content.WriteString("\n")
	}

	// Outgoing edges with detail
	outEdges := m.report.Graph.OutEdges[nodeID]
	if len(outEdges) > 0 {
		content.WriteString(StatValue.Render("References (out)") + "\n")
		for _, e := range outEdges {
			var parts []string
			parts = append(parts, StatValue.Render(e.To))
			kindStr := string(e.Kind)
			if e.FieldName != "" {
				kindStr += ":" + e.FieldName
			}
			parts = append(parts, StatLabel.Render("via "+kindStr))

			var flags []string
			if e.IsRequired {
				flags = append(flags, RequiredEdge.Render("req"))
			}
			if e.IsNullable {
				flags = append(flags, GreenBadge.Render("nullable"))
			}
			if e.IsArray {
				flags = append(flags, ArrayEdge.Render("array"))
			}
			if len(flags) > 0 {
				parts = append(parts, "["+strings.Join(flags, " ")+"]")
			}

			content.WriteString("  " + strings.Join(parts, " ") + "\n")
		}
		content.WriteString("\n")
	}

	// Incoming edges
	inEdges := m.report.Graph.InEdges[nodeID]
	if len(inEdges) > 0 {
		refs := make(map[string]bool)
		for _, e := range inEdges {
			refs[e.From] = true
		}
		refList := make([]string, 0, len(refs))
		for r := range refs {
			refList = append(refList, r)
		}
		sort.Strings(refList)
		refStr := strings.Join(refList, ", ")
		if innerWidth > 16 && len(refStr) > innerWidth-16 {
			refStr = refStr[:innerWidth-19] + "..."
		}
		content.WriteString(fmt.Sprintf("%s %s\n",
			StatLabel.Render("Referenced by:"), StatValue.Render(refStr)))
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(cardWidth)

	return "    " + strings.ReplaceAll(cardStyle.Render(content.String()), "\n", "\n    ") + "\n"
}
