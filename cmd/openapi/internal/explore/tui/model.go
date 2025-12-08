package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore"
)

const (
	// keySequenceThreshold is the max time between key presses for sequences like "gg"
	keySequenceThreshold = 500 * time.Millisecond

	// scrollHalfScreenLines is the number of lines to scroll with Ctrl-D/U
	scrollHalfScreenLines = 21

	// Layout constants
	headerApproxLines = 2 // Single line header + one empty line
	footerApproxLines = 4
	layoutBuffer      = 2 // Extra buffer to ensure header visibility
	leftPaddingChars  = 2 // "▶" + space
)

// Config contains all configuration for the TUI
type Config struct {
	// Explore configures the explore mode appearance and text
	Explore ExploreConfig
	// Selection configures optional selection mode behavior
	Selection SelectionConfig
}

// ExploreConfig configures the explore mode (browsing) appearance and text
type ExploreConfig struct {
	// Title is the main title shown in the header
	Title string
	// ModeLabel is the label shown in the navigation section
	ModeLabel string
	// FooterHelpText is the help text shown in the footer
	FooterHelpText string
	// HelpTitle is the title for the help modal
	HelpTitle string
}

// ActionKey represents a key that triggers an action in selection mode
type ActionKey struct {
	// Key is the keyboard key (e.g., "w", "d", "ctrl+s")
	Key string
	// Label is the description for help text (e.g., "Write and save", "Discard changes")
	Label string
}

// SelectionConfig configures optional selection mode behavior
// When Enabled is true, users can select operations with Space
type SelectionConfig struct {
	// Enabled indicates if selection mode is active
	Enabled bool
	// SelectIcon is the icon shown for selected items (e.g., "✂️", "✓", "❌")
	SelectIcon string
	// SelectColor is the lipgloss color for selected items (e.g., colorGreen)
	SelectColor string
	// StatusFormat is a format string for the footer status (receives selected count as %d)
	// Example: "Selected: %d operations"
	StatusFormat string
	// ActionKeys defines the keys that trigger actions when pressed
	// Example: []ActionKey{{Key: "w", Label: "Write and save"}}
	ActionKeys []ActionKey
}

// DefaultConfig returns the default configuration for view-only mode
func DefaultConfig() Config {
	return Config{
		Explore: ExploreConfig{
			Title:          "OpenAPI Spec Explorer",
			ModeLabel:      "Operations",
			FooterHelpText: "Press '?' for help",
			HelpTitle:      "Help",
		},
		Selection: SelectionConfig{
			Enabled: false,
		},
	}
}

// Model represents the TUI application state
type Model struct {
	// Data
	operations []explore.OperationInfo
	docTitle   string
	docVersion string

	// Configuration
	config Config

	// UI state
	cursor       int
	width        int
	height       int
	scrollOffset int
	showHelp     bool

	// Selection state (only used when selectionConfig.Enabled is true)
	selected map[int]bool

	// Key sequence handling
	lastKey   string
	lastKeyAt time.Time

	// Terminal state
	quitting  bool
	actionKey string // The action key that was pressed (only set when quitting in selection mode)
}

// NewModel creates a new TUI model with default view-only configuration
func NewModel(operations []explore.OperationInfo, docTitle, docVersion string) Model {
	return NewModelWithConfig(operations, docTitle, docVersion, DefaultConfig())
}

// NewModelWithConfig creates a new TUI model with custom configuration
func NewModelWithConfig(operations []explore.OperationInfo, docTitle, docVersion string, config Config) Model {
	return Model{
		operations:   operations,
		docTitle:     docTitle,
		docVersion:   docVersion,
		config:       config,
		cursor:       0,
		width:        80,
		height:       24,
		scrollOffset: 0,
		showHelp:     false,
		selected:     make(map[int]bool),
		lastKey:      "",
		lastKeyAt:    time.Time{},
		quitting:     false,
		actionKey:    "",
	}
}

// GetSelectedOperations returns the operations that have been selected
// Only relevant when selectionConfig.Enabled is true
func (m Model) GetSelectedOperations() []explore.OperationInfo {
	var selected []explore.OperationInfo
	for idx, isSelected := range m.selected {
		if isSelected && idx < len(m.operations) {
			selected = append(selected, m.operations[idx])
		}
	}
	return selected
}

// GetActionKey returns the action key that was pressed to quit (only relevant in selection mode)
// Returns empty string if user cancelled or quit without an action
func (m Model) GetActionKey() string {
	return m.actionKey
}

// Init initializes the model (required by bubbletea)
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model (required by bubbletea)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.showHelp {
				m.showHelp = false
			} else {
				m.quitting = true
				return m, tea.Quit
			}

		case "?":
			m.showHelp = !m.showHelp

		case "esc":
			if m.showHelp {
				m.showHelp = false
			}

		case "up", "k":
			if !m.showHelp && m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
			}

		case "down", "j":
			if !m.showHelp && m.cursor < len(m.operations)-1 {
				m.cursor++
				m.ensureCursorVisible()
			}

		case "ctrl+d":
			if !m.showHelp {
				maxItems := len(m.operations) - 1
				newCursorPos := m.cursor + scrollHalfScreenLines

				if newCursorPos > maxItems {
					m.cursor = maxItems
				} else {
					m.cursor = newCursorPos
				}

				m.ensureCursorVisible()
			}

		case "ctrl+u":
			if !m.showHelp {
				halfLines := max(1, m.calculateContentHeight()/2)
				if m.cursor < halfLines {
					m.cursor = 0
				} else {
					m.cursor -= halfLines
				}

				m.ensureCursorVisible()
			}

		case "G":
			if !m.showHelp && len(m.operations) > 0 {
				m.cursor = len(m.operations) - 1
				m.ensureCursorVisible()
			}

		case "g":
			now := time.Now()
			if m.lastKey == "g" && now.Sub(m.lastKeyAt) < keySequenceThreshold {
				if !m.showHelp {
					m.cursor = 0
					m.ensureCursorVisible()
				}

				// Reset so "ggg" wouldn't be triggered
				m.lastKey = ""
				m.lastKeyAt = time.Time{}
			} else {
				m.lastKey = "g"
				m.lastKeyAt = now
			}

		case " ":
			if !m.showHelp && m.cursor < len(m.operations) {
				if m.config.Selection.Enabled {
					// In selection mode, space toggles selection
					m.selected[m.cursor] = !m.selected[m.cursor]
				} else {
					// In view mode, space toggles fold
					m.operations[m.cursor].Folded = !m.operations[m.cursor].Folded
				}
			}

		case "enter":
			// Enter always toggles details (in both modes)
			if !m.showHelp && m.cursor < len(m.operations) {
				m.operations[m.cursor].Folded = !m.operations[m.cursor].Folded
			}

		case "a":
			// Select all (only in selection mode)
			if !m.showHelp && m.config.Selection.Enabled {
				for i := range m.operations {
					m.selected[i] = true
				}
			}

		case "A":
			// Deselect all (only in selection mode)
			if !m.showHelp && m.config.Selection.Enabled {
				m.selected = make(map[int]bool)
			}

		default:
			// Check for action keys in selection mode
			if !m.showHelp && m.config.Selection.Enabled {
				for _, action := range m.config.Selection.ActionKeys {
					if msg.String() == action.Key {
						// Action key pressed - quit and return which action
						m.actionKey = action.Key
						m.quitting = true
						return m, tea.Quit
					}
				}
			}
		}
	}

	return m, nil
}

// View renders the current state (required by bubbletea)
func (m Model) View() string {
	if m.showHelp {
		return m.renderHelpModal()
	}

	var s strings.Builder

	header := m.renderHeader()
	footer := m.renderFooter()
	content := m.renderOperations()

	headerLines := strings.Count(header, "\n")
	footerLines := strings.Count(footer, "\n")
	contentLines := strings.Count(content, "\n")

	// Build the view
	s.WriteString(header)
	s.WriteString(content)

	// Add padding to fill remaining space
	usedLines := headerLines + contentLines + footerLines
	remainingLines := m.height - usedLines - 1
	if remainingLines > 0 {
		s.WriteString(strings.Repeat("\n", remainingLines))
	}

	s.WriteString(footer)

	return s.String()
}

// calculateContentHeight returns the available height for content
func (m Model) calculateContentHeight() int {
	return max(1, m.height-headerApproxLines-footerApproxLines-layoutBuffer)
}

// calculateContentWidth returns the available width for content
func (m Model) calculateContentWidth() int {
	return max(1, m.width-leftPaddingChars)
}

// getItemHeight returns the height in lines of an item at the given index
func (m Model) getItemHeight(index int) int {
	if index >= len(m.operations) {
		return 1
	}

	op := m.operations[index]

	if op.Folded {
		return 1 // Just the main line when folded
	}

	// When unfolded, count main line + detail lines (regardless of selection)
	details := m.formatOperationDetails(op)
	return 1 + strings.Count(details, "\n") + 1
}

// ensureCursorVisible adjusts scrollOffset to keep cursor visible
func (m *Model) ensureCursorVisible() {
	contentHeight := m.calculateContentHeight()

	// Special case: if cursor is at 0, scroll to the very top
	if m.cursor == 0 {
		m.scrollOffset = 0
		return
	}

	// If cursor is above current scroll position, scroll up to show it
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
		return
	}

	// Calculate how many lines are used from scrollOffset to cursor (inclusive)
	linesUsed := 0

	// Account for scroll indicator
	if m.scrollOffset > 0 {
		linesUsed++ // "More items above" indicator
	}

	// Add lines for each item
	for i := m.scrollOffset; i <= m.cursor && i < len(m.operations); i++ {
		linesUsed += m.getItemHeight(i)
	}

	// If the cursor item extends beyond available content height, scroll down
	if linesUsed > contentHeight {
		// Find the minimum scroll offset that keeps cursor visible
		for newScrollOffset := m.scrollOffset + 1; newScrollOffset <= m.cursor; newScrollOffset++ {
			testLinesUsed := 0

			// Account for "More items above" indicator
			if newScrollOffset > 0 {
				testLinesUsed++
			}

			// Calculate lines from new scroll offset to cursor
			for i := newScrollOffset; i <= m.cursor && i < len(m.operations); i++ {
				testLinesUsed += m.getItemHeight(i)
			}

			if testLinesUsed <= contentHeight {
				m.scrollOffset = newScrollOffset
				break
			}
		}
	}

	// Ensure scroll offset doesn't go negative
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}
