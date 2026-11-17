package arazzoexplore

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore/tui"
)

const (
	appTitle  = "Arazzo Explorer"
	modeLabel = "Workflows"
	helpHint  = "Press '?' for help"

	colorGreen      = "#10B981"
	colorBlue       = "#3B82F6"
	colorYellow     = "#F59E0B"
	colorPurple     = "#8B5CF6"
	colorGray       = "#6B7280"
	colorWhite      = "#FFFFFF"
	colorBackground = "#374151"
)

// badgeColors are the colors of the step target badges.
var badgeColors = map[string]string{
	BadgeOperation:     colorGreen,
	BadgeOperationPath: colorBlue,
	BadgeWorkflow:      colorPurple,
	BadgeSend:          colorYellow,
	BadgeReceive:       colorYellow,
	BadgeChannel:       colorYellow,
}

var (
	workflowLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorPurple))
	stepLabelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWhite))
	summaryStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGray))
)

// helpRows are the key bindings shown in the help modal.
var helpRows = [][2]string{
	{"↑/k", "Move up"},
	{"↓/j", "Move down"},
	{"gg", "Move to the top"},
	{"G", "Move to the bottom"},
	{"Ctrl-U", "Scroll up by half a screen"},
	{"Ctrl-D", "Scroll down by half a screen"},
	{"[ / ]", "Previous / next workflow"},
	{"Enter/Space", "Toggle details"},
	{"e / E", "Expand / collapse all details"},
	{"?", "Toggle help"},
	{"Esc/q", "Close help"},
	{"q/Ctrl+C", "Quit"},
}

// View renders the current state (required by bubbletea).
func (m Model) View() tea.View {
	var content string
	if m.showHelp {
		content = m.renderHelp()
	} else {
		content = m.renderHeader() + strings.Join(m.renderRows(), "\n") + m.renderFooter()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderHeader() string {
	nav := tui.ActiveButtonStyle.Render(modeLabel)
	title := tui.TitleStyle.Render(appTitle)

	line := nav
	if spacing := m.width - lipgloss.Width(nav) - lipgloss.Width(title); spacing > 4 {
		line += strings.Repeat(" ", spacing) + title
	}
	return line + "\n\n"
}

// renderRows renders the rows in the content area, padded to fill it.
func (m Model) renderRows() []string {
	height := m.contentHeight()
	lines := []string{}

	if m.scrollOffset > 0 && m.showIndicators() {
		lines = append(lines, tui.ScrollIndicatorStyle.Render("⬆ More above..."))
	}

	truncated := false
	for i := m.scrollOffset; i < len(m.items); i++ {
		rows := m.renderItem(i)
		if len(lines)+len(rows) > height {
			lines = append(lines, rows[:height-len(lines)]...)
			truncated = true
			break
		}
		lines = append(lines, rows...)
	}

	// Replace the last line with an indicator when rows or details are cut off below.
	if truncated && m.showIndicators() {
		lines[len(lines)-1] = tui.ScrollIndicatorStyle.Render("⬇ More below...")
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

// renderItem renders an item's row, followed by its details when it's expanded.
func (m Model) renderItem(index int) []string {
	item := m.items[index]
	highlighted := index == m.cursor

	style := func(s lipgloss.Style) lipgloss.Style {
		if highlighted {
			return s.Background(lipgloss.Color(colorBackground))
		}
		return s
	}
	plain := style(lipgloss.NewStyle())

	icon := "▶ "
	if m.expanded[index] {
		icon = "▼ "
	}

	var row strings.Builder
	detailIndent := "  "
	switch item.Kind {
	case KindWorkflow:
		row.WriteString(plain.Render(icon))
		row.WriteString(style(workflowLabelStyle).Render(item.Label))
	case KindStep:
		detailIndent = "    "
		badgeStyle := style(lipgloss.NewStyle().Bold(true).Width(5).Foreground(lipgloss.Color(badgeColor(item.Badge))))
		row.WriteString(plain.Render("  " + icon))
		row.WriteString(badgeStyle.Render(item.Badge))
		row.WriteString(style(stepLabelStyle).Render(item.Label))
	}
	if item.Summary != "" {
		row.WriteString(style(summaryStyle).Render("  " + item.Summary))
	}
	if pad := m.width - lipgloss.Width(row.String()); pad > 0 && highlighted {
		row.WriteString(plain.Render(strings.Repeat(" ", pad)))
	}

	truncate := lipgloss.NewStyle().MaxWidth(m.width)
	lines := []string{truncate.Render(row.String())}
	if m.expanded[index] {
		for _, detail := range item.Details {
			lines = append(lines, truncate.Render(tui.DetailStyle.Render(detailIndent+detail)))
		}
	}
	return lines
}

func badgeColor(badge string) string {
	if c, ok := badgeColors[badge]; ok {
		return c
	}
	return colorGray
}

func (m Model) renderFooter() string {
	workflows := 0
	for _, item := range m.items {
		if item.Kind == KindWorkflow {
			workflows++
		}
	}

	status := fmt.Sprintf("%s v%s · Arazzo %s · %d workflows", m.docTitle, m.docVersion, m.arazzoVersion, workflows)
	help := helpHint

	// The footer style pads one character on each side.
	width := m.width - 2
	if lipgloss.Width(help)+lipgloss.Width(status) >= width {
		help = ""
	}
	spacing := strings.Repeat(" ", max(0, width-lipgloss.Width(help)-lipgloss.Width(status)))

	return "\n\n" + tui.FooterStyle.Width(m.width).MaxWidth(m.width).Render(help+spacing+status)
}

func (m Model) renderHelp() string {
	keyWidth := 0
	for _, row := range helpRows {
		keyWidth = max(keyWidth, lipgloss.Width(row[0]))
	}

	items := make([]string, 0, len(helpRows))
	for _, row := range helpRows {
		key := row[0] + strings.Repeat(" ", keyWidth-lipgloss.Width(row[0]))
		items = append(items, tui.HelpKeyStyle.Render(key)+tui.HelpTextStyle.Render(" "+row[1]))
	}

	modal := tui.HelpModalStyle.Render(tui.HelpTitleStyle.Render("Help") + "\n\n" + strings.Join(items, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
