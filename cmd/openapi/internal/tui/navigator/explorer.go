package navigator

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Explorer represents the bubbletea model for the navigator
type Explorer struct {
	tree         *Tree
	styles       *Styles
	selectedItem int
	width        int
	height       int
	showHelp     bool
	quitting     bool
}

// NewExplorer creates a new navigator explorer
func NewExplorer(root TreeNode) *Explorer {
	return &Explorer{
		tree:         NewTree(root),
		styles:       NewStyles(),
		selectedItem: 0,
		width:        80,
		height:       24,
		showHelp:     false,
		quitting:     false,
	}
}

// Init initializes the explorer
func (m *Explorer) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the explorer
func (m *Explorer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m *Explorer) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "up":
		if m.selectedItem > 0 {
			m.selectedItem--
		}
		return m, nil

	case "down":
		items := m.tree.GetCurrentLevel()
		if m.selectedItem < len(items)-1 {
			m.selectedItem++
		}
		return m, nil

	case "enter", "right":
		return m.navigateInto()

	case "left", "esc":
		return m.navigateBack()

	case "home":
		m.selectedItem = 0
		return m, nil

	case "end":
		items := m.tree.GetCurrentLevel()
		if len(items) > 0 {
			m.selectedItem = len(items) - 1
		}
		return m, nil
	}

	return m, nil
}

// navigateInto moves into the selected item if it's expandable
func (m *Explorer) navigateInto() (tea.Model, tea.Cmd) {
	items := m.tree.GetCurrentLevel()
	if m.selectedItem >= len(items) {
		return m, nil
	}

	selectedNode := items[m.selectedItem]
	if !selectedNode.IsExpandable() {
		return m, nil // Don't navigate into leaf nodes
	}

	// Check if the selected node has children that are also expandable
	// If all children are leaf nodes, don't navigate deeper
	children := selectedNode.GetChildren()
	hasExpandableChildren := false
	for _, child := range children {
		if child.IsExpandable() {
			hasExpandableChildren = true
			break
		}
	}

	// Only navigate if there are expandable children
	// This prevents navigation into nodes where all children are leaf nodes
	if hasExpandableChildren {
		if m.tree.NavigateToChild(m.selectedItem) {
			m.selectedItem = 0 // Reset selection to first item in new level
		}
	}

	return m, nil
}

// navigateBack moves back to the parent level
func (m *Explorer) navigateBack() (tea.Model, tea.Cmd) {
	if m.tree.NavigateToParent() {
		m.selectedItem = 0 // Reset selection when going back
	}
	return m, nil
}

// View renders the explorer
func (m *Explorer) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var sections []string

	// Header with breadcrumb
	sections = append(sections, m.renderHeader())

	// Main content
	sections = append(sections, m.renderContent())

	// Footer with help
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header with breadcrumb navigation
func (m *Explorer) renderHeader() string {
	breadcrumb := m.tree.GetBreadcrumb()
	if breadcrumb == "" {
		breadcrumb = "OpenAPI Document"
	}

	header := m.styles.HeaderStyle.Render(fmt.Sprintf("📍 %s", breadcrumb))
	return lipgloss.NewStyle().Width(m.width).Render(header)
}

// renderContent renders the main navigation content
func (m *Explorer) renderContent() string {
	if m.showHelp {
		return m.renderHelp()
	}

	items := m.tree.GetCurrentLevelWithPreview()
	if len(items) == 0 {
		return m.styles.ContentStyle.Render("No items to display")
	}

	// Calculate dynamic column width based on content
	maxTitleWidth := m.calculateMaxTitleWidth(items)

	var lines []string
	contentHeight := m.height - 4 // Reserve space for header and footer

	for i, item := range items {
		if len(lines) >= contentHeight-2 { // Leave some space
			break
		}

		// Render main item with colored title
		itemText := m.renderItemWithColors(item, maxTitleWidth)
		if i == m.selectedItem {
			// Add padding manually to the text content, not the style
			paddedText := " " + itemText + " "
			lines = append(lines, m.styles.SelectedStyle.Render(paddedText))
		} else {
			lines = append(lines, m.styles.ItemStyle.Render(itemText))
		}

		// Render preview if selected and has children (YAML-like style)
		if i == m.selectedItem && len(item.Preview) > 0 {
			for _, preview := range item.Preview {
				if len(lines) >= contentHeight-1 {
					break
				}
				// YAML-like indentation without trailing colon
				previewText := fmt.Sprintf("  %s", preview.Name)
				lines = append(lines, m.styles.PreviewStyle.Render(previewText))
			}
		}
	}

	content := strings.Join(lines, "\n")
	return m.styles.ContentStyle.Render(content)
}

// renderHelp renders the help screen
func (m *Explorer) renderHelp() string {
	help := []string{
		"Navigation Help:",
		"",
		"  ↑          Move up",
		"  ↓          Move down",
		"  →/Enter    Enter selected item",
		"  ←/Esc      Go back to parent",
		"  Home       Go to first item",
		"  End        Go to last item",
		"  ?          Toggle this help",
		"  q/Ctrl+C   Quit",
		"",
		"Press ? again to close this help.",
	}

	return m.styles.ContentStyle.Render(strings.Join(help, "\n"))
}

// renderFooter renders the footer with command hints
func (m *Explorer) renderFooter() string {
	var commands []string

	if m.showHelp {
		commands = append(commands, "? close help")
	} else {
		commands = append(commands, "↑↓ navigate")

		items := m.tree.GetCurrentLevel()
		if m.selectedItem < len(items) {
			selectedNode := items[m.selectedItem]
			if selectedNode.IsExpandable() {
				// Check if this node has expandable children
				children := selectedNode.GetChildren()
				hasExpandableChildren := false
				for _, child := range children {
					if child.IsExpandable() {
						hasExpandableChildren = true
						break
					}
				}
				if hasExpandableChildren {
					commands = append(commands, "→ enter")
				}
			}
		}

		if len(m.tree.Path) > 1 {
			commands = append(commands, "← back")
		}

		commands = append(commands, "? help")
	}

	commands = append(commands, "q quit")

	footer := strings.Join(commands, " • ")
	return m.styles.FooterStyle.Width(m.width).Render(footer)
}

// GetCurrentNode returns the currently selected node
func (m *Explorer) GetCurrentNode() TreeNode {
	return m.tree.Current
}

// GetSelectedNode returns the currently selected child node
func (m *Explorer) GetSelectedNode() TreeNode {
	items := m.tree.GetCurrentLevel()
	if m.selectedItem >= 0 && m.selectedItem < len(items) {
		return items[m.selectedItem]
	}
	return nil
}

// SetSize updates the model's dimensions
func (m *Explorer) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// calculateMaxTitleWidth calculates the maximum title width for dynamic column sizing
func (m *Explorer) calculateMaxTitleWidth(items []DisplayItem) int {
	maxWidth := 0
	for _, item := range items {
		titleLen := len(item.Node.GetDisplayTitle())
		if titleLen > maxWidth {
			maxWidth = titleLen
		}
	}
	// Add a small buffer
	return maxWidth + 2
}

// renderItemWithColors renders an item with colored title and proper column formatting
func (m *Explorer) renderItemWithColors(item DisplayItem, maxTitleWidth int) string {
	node := item.Node
	title := node.GetDisplayTitle()
	description := node.GetDescription()

	// Handle operations differently to match the aesthetic
	if node.GetNodeType() == NodeTypeOperation {
		return m.renderOperationItem(node, title, description)
	}

	// For non-operations, use the tree-style with expand indicators
	// Check if this node can actually be navigated into
	canNavigate := item.HasChildren && m.hasExpandableChildren(item.Node)
	icon := m.getExpandIcon(canNavigate)

	// Apply color styling to the title based on node type
	titleStyle := m.styles.NodeTypeStyle(node.GetNodeType())
	coloredTitle := titleStyle.Render(title)

	// Only show description if it's not empty
	if description != "" && strings.TrimSpace(description) != "" {
		// Use dynamic column width
		actualTitleWidth := len(title)

		if actualTitleWidth < maxTitleWidth {
			padding := maxTitleWidth - actualTitleWidth
			paddingStr := strings.Repeat(" ", padding)
			return fmt.Sprintf("%s %s%s%s", icon, coloredTitle, paddingStr, description)
		} else {
			return fmt.Sprintf("%s %s  %s", icon, coloredTitle, description)
		}
	}

	return fmt.Sprintf("%s %s", icon, coloredTitle)
}

// renderOperationItem renders HTTP operations in the style shown in the screenshot
func (m *Explorer) renderOperationItem(node TreeNode, title, description string) string {
	// Extract HTTP method from the title (assuming format like "GET List all pets")
	parts := strings.SplitN(title, " ", 2)
	if len(parts) >= 2 {
		method := parts[0]

		// Color the HTTP method and make it uppercase
		methodStyle := m.styles.HTTPMethodStyle(method)
		coloredMethod := methodStyle.Render(strings.ToUpper(method))

		// Format as: "• GET    /path/to/endpoint"
		if description != "" {
			return fmt.Sprintf("• %s    %s", coloredMethod, description)
		}
		return fmt.Sprintf("• %s", coloredMethod)
	}

	// Fallback for operations that don't match expected format
	titleStyle := m.styles.NodeTypeStyle(node.GetNodeType())
	coloredTitle := titleStyle.Render(title)
	return fmt.Sprintf("• %s", coloredTitle)
}

// getExpandIcon returns the appropriate expand/collapse icon
func (m *Explorer) getExpandIcon(canNavigate bool) string {
	if canNavigate {
		return "▸" // Right-pointing triangle for expandable items
	}
	return "•" // Bullet point for non-expandable items
}

// hasExpandableChildren checks if a node has children that can be expanded further
func (m *Explorer) hasExpandableChildren(node TreeNode) bool {
	children := node.GetChildren()
	for _, child := range children {
		if child.IsExpandable() {
			return true
		}
	}
	return false
}
