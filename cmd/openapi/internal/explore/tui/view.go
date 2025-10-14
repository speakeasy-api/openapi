package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore"
)

// renderHeader renders the application header with navigation
func (m Model) renderHeader() string {
	// Build title and nav from config
	appTitle := TitleStyle.Render(m.config.Explore.Title)
	navSection := ActiveButtonStyle.Render(m.config.Explore.ModeLabel)

	// Calculate spacing
	navWidth := lipgloss.Width(navSection)
	titleWidth := lipgloss.Width(appTitle)
	totalContentWidth := navWidth + titleWidth

	// Create the header line with proper spacing
	var headerLine string
	if m.width > totalContentWidth+4 { // 4 for some padding
		spacingWidth := m.width - totalContentWidth
		spacing := strings.Repeat(" ", spacingWidth)
		headerLine = navSection + spacing + appTitle
	} else {
		// If not enough space, just show navigation
		headerLine = navSection
	}

	// Return header with one empty line below
	return headerLine + "\n\n"
}

// renderFooter renders the application footer with help text and status/doc info
func (m Model) renderFooter() string {
	// Use help text from config
	helpText := m.config.Explore.FooterHelpText
	if m.showHelp {
		helpText = ""
	}

	// Build status/info based on selection mode
	var statusInfo string
	if m.config.Selection.Enabled {
		// In selection mode, show selected count
		selectedCount := len(m.selected)
		statusInfo = fmt.Sprintf(m.config.Selection.StatusFormat, selectedCount)
	} else {
		// In view mode, show doc info
		statusInfo = fmt.Sprintf("%s v%s", m.docTitle, m.docVersion)
	}

	// Calculate actual content width (accounting for padding in FooterStyle)
	// FooterStyle has Padding(0, 1) which adds 2 chars total
	contentWidth := m.width - 2

	// Calculate spacing needed between help text and status
	helpTextLen := len(helpText)
	statusInfoLen := len(statusInfo)
	neededWidth := helpTextLen + statusInfoLen

	var footerContent string
	if neededWidth >= contentWidth {
		// Not enough space for both - prioritize status
		helpText = ""
		spacing := strings.Repeat(" ", max(0, contentWidth-len(statusInfo)))
		footerContent = spacing + statusInfo
	} else {
		// Enough space - add spacing between
		spacing := strings.Repeat(" ", contentWidth-helpTextLen-statusInfoLen)
		footerContent = helpText + spacing + statusInfo
	}

	footerStyle := FooterStyle.
		Width(m.width).
		Align(lipgloss.Left)

	return "\n" + footerStyle.Render(footerContent)
}

// renderOperations renders the list of operations
func (m Model) renderOperations() string {
	var s strings.Builder

	contentHeight := m.calculateContentHeight()
	contentWidth := m.calculateContentWidth()

	startIdx := m.scrollOffset
	endIdx := min(m.scrollOffset+contentHeight, len(m.operations))

	// Add scroll indicator for items above
	if m.scrollOffset > 0 {
		indicator := ScrollIndicatorStyle.Render("⬆ More items above...")
		s.WriteString(indicator)
		s.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		op := m.operations[i]
		isSelected := m.selected[i]

		style := lipgloss.NewStyle()
		methodStyle := GetMethodStyle(op.Method, i == m.cursor)

		// Override colors if selected in selection mode
		if m.config.Selection.Enabled && isSelected {
			selectionColor := lipgloss.Color(m.config.Selection.SelectColor)
			style = style.Foreground(selectionColor)
			methodStyle = lipgloss.NewStyle().
				Foreground(selectionColor).
				Bold(true).
				Width(7)
		}

		if i == m.cursor {
			if m.config.Selection.Enabled && isSelected {
				// Keep selection color but add background
				style = style.Background(lipgloss.Color(colorBackground))
				methodStyle = methodStyle.Background(lipgloss.Color(colorBackground))
			} else {
				style = GetHighlightStyle()
			}
		}

		// Fold icon (always shown)
		foldIcon := "▶"
		if !op.Folded {
			foldIcon = "▼"
		}

		var line strings.Builder
		line.WriteString(style.Render(foldIcon + " "))

		// Selection icon column (only in selection mode)
		if m.config.Selection.Enabled {
			if isSelected {
				line.WriteString(style.Render(m.config.Selection.SelectIcon + " "))
			} else {
				// Empty space to keep alignment
				line.WriteString(style.Render("  "))
			}
		}

		line.WriteString(methodStyle.Render(op.Method))
		line.WriteString(style.Render(" " + op.Path))
		line.WriteString(style.Render(strings.Repeat(" ", contentWidth)))

		s.WriteString(style.Render(line.String()))
		s.WriteString("\n")

		// Show details when unfolded (regardless of selection state)
		if !op.Folded {
			details := m.formatOperationDetails(op)
			s.WriteString(DetailStyle.Render(details))
			s.WriteString("\n")
		}
	}

	// Add scroll indicator for items below
	if endIdx < len(m.operations) {
		indicator := ScrollIndicatorStyle.Render("⬇ More items below...")
		s.WriteString(indicator)
		s.WriteString("\n")
	}

	return s.String()
}

// formatOperationDetails formats the detailed information for an operation
func (m Model) formatOperationDetails(op explore.OperationInfo) string {
	var details strings.Builder

	if op.Summary != "" {
		details.WriteString(fmt.Sprintf("Summary: %s\n", op.Summary))
	}

	if op.Description != "" {
		details.WriteString(fmt.Sprintf("Description: %s\n", op.Description))
	}

	if op.OperationID != "" {
		details.WriteString(fmt.Sprintf("Operation ID: %s\n", op.OperationID))
	}

	if len(op.Tags) > 0 {
		details.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(op.Tags, ", ")))
	}

	if op.Deprecated {
		details.WriteString("⚠️  DEPRECATED\n")
	}

	// Add parameter information
	if params := op.Operation.GetParameters(); len(params) > 0 {
		details.WriteString("Parameters:\n")
		for _, param := range params {
			if param != nil && param.Object != nil {
				p := param.Object
				required := ""
				if p.Required != nil && *p.Required {
					required = " (required)"
				}
				details.WriteString(fmt.Sprintf("  - %s (%s)%s: %s\n",
					p.Name, p.In, required, p.GetDescription()))
			}
		}
	}

	// Add request body information
	if reqBody := op.Operation.GetRequestBody(); reqBody != nil && reqBody.Object != nil {
		details.WriteString("Request Body:\n")
		if reqBody.Object.Content != nil {
			// Get media types and sort them
			var mediaTypes []string
			for mediaType := range reqBody.Object.Content.All() {
				mediaTypes = append(mediaTypes, mediaType)
			}
			sort.Strings(mediaTypes)

			for _, mediaType := range mediaTypes {
				details.WriteString(fmt.Sprintf("  - %s\n", mediaType))
			}
		}
	}

	// Add response information
	if responses := op.Operation.GetResponses(); responses != nil {
		details.WriteString("Responses:\n")

		// Get response codes and sort them
		var codes []string
		for code := range responses.All() {
			codes = append(codes, code)
		}
		sortResponseCodes(codes)

		for _, code := range codes {
			if resp, ok := responses.Get(code); ok && resp != nil && resp.Object != nil {
				desc := resp.Object.GetDescription()
				if desc != "" {
					details.WriteString(fmt.Sprintf("  - %s: %s\n", code, desc))
				} else {
					details.WriteString(fmt.Sprintf("  - %s\n", code))
				}
			}
		}
	}

	return details.String()
}

// sortResponseCodes sorts HTTP response codes with stable ordering
func sortResponseCodes(codes []string) {
	sort.Slice(codes, func(i, j int) bool {
		// Try to parse as integers
		var codeI, codeJ int
		_, errI := fmt.Sscanf(codes[i], "%d", &codeI)
		_, errJ := fmt.Sscanf(codes[j], "%d", &codeJ)

		// Both are numeric - sort numerically
		if errI == nil && errJ == nil {
			return codeI < codeJ
		}

		// One numeric, one non-numeric - numeric comes first
		if errI == nil && errJ != nil {
			return true
		}
		if errI != nil && errJ == nil {
			return false
		}

		// Both non-numeric - sort alphabetically
		return codes[i] < codes[j]
	})
}

// renderHelpModal renders the help modal overlay
func (m Model) renderHelpModal() string {
	// Start with base explore mode commands
	helpData := [][]string{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"gg", "Move to the top"},
		{"G", "Move to the bottom"},
		{"Ctrl-U", "Scroll up by half a screen"},
		{"Ctrl-D", "Scroll down by half a screen"},
	}

	// Add mode-specific commands
	if m.config.Selection.Enabled {
		// Selection mode commands
		helpData = append(helpData, []string{"Space", "Select/deselect operation"})
		helpData = append(helpData, []string{"a", "Select all operations"})
		helpData = append(helpData, []string{"A", "Deselect all operations"})
		helpData = append(helpData, []string{"Enter/Space", "Toggle details / Select"})

		// Add action keys (mode-specific commands like "w" for write)
		for _, action := range m.config.Selection.ActionKeys {
			helpData = append(helpData, []string{action.Key, action.Label})
		}
	} else {
		// View mode commands
		helpData = append(helpData, []string{"Enter/Space", "Toggle details"})
	}

	// Add common commands at the end
	helpData = append(helpData, []string{"?", "Toggle help"})
	if m.config.Selection.Enabled {
		helpData = append(helpData, []string{"Esc/q", "Cancel and quit"})
	} else {
		helpData = append(helpData, []string{"Esc/q", "Close help"})
	}
	helpData = append(helpData, []string{"Ctrl+C", "Quit"})

	// Find max width for first column
	maxKeyWidth := 0
	for _, row := range helpData {
		if len(row[0]) > maxKeyWidth {
			maxKeyWidth = len(row[0])
		}
	}

	var helpItems []string
	for _, row := range helpData {
		key := HelpKeyStyle.Render(fmt.Sprintf("%-*s", maxKeyWidth, row[0]))
		desc := HelpTextStyle.Render(" " + row[1])
		helpItems = append(helpItems, key+desc)
	}

	helpContent := strings.Join(helpItems, "\n")

	title := HelpTitleStyle.Render(m.config.Explore.HelpTitle)
	modal := HelpModalStyle.Render(title + "\n\n" + helpContent)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
