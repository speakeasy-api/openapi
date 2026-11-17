package arazzoexplore

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	// keySequenceThreshold is the max time between key presses for sequences like "gg".
	keySequenceThreshold = 500 * time.Millisecond

	// headerLines is the header line and the empty line below it.
	headerLines = 2
	// footerLines is the empty line above the footer and the footer itself.
	footerLines = 2
)

// Model is the state of the Arazzo explorer.
type Model struct {
	items []Item
	// expanded tracks which items show their details.
	expanded map[int]bool

	docTitle      string
	docVersion    string
	arazzoVersion string

	cursor       int
	scrollOffset int
	width        int
	height       int
	showHelp     bool

	lastKey   string
	lastKeyAt time.Time
}

// NewModel creates an explorer for the items of a document (see Collect).
func NewModel(items []Item, docTitle, docVersion, arazzoVersion string) Model {
	return Model{
		items:         items,
		expanded:      map[int]bool{},
		docTitle:      docTitle,
		docVersion:    docVersion,
		arazzoVersion: arazzoVersion,
		width:         80,
		height:        24,
	}
}

// Init initializes the model (required by bubbletea).
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model (required by bubbletea).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()

	case tea.KeyPressMsg:
		key := msg.String()

		// Only consecutive presses form the "gg" sequence.
		if key != "g" {
			m.lastKey = ""
		}

		switch key {
		case "q", "ctrl+c":
			if !m.showHelp {
				return m, tea.Quit
			}
			m.showHelp = false
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			m.showHelp = false
			return m, nil
		}

		// The help modal swallows every other key.
		if m.showHelp || len(m.items) == 0 {
			return m, nil
		}

		switch key {
		case "up", "k":
			m.moveTo(m.cursor - 1)
		case "down", "j":
			m.moveTo(m.cursor + 1)
		case "ctrl+u":
			m.moveTo(m.cursor - m.halfPage())
		case "ctrl+d":
			m.moveTo(m.cursor + m.halfPage())
		case "G":
			m.moveTo(len(m.items) - 1)
		case "g":
			now := time.Now()
			if m.lastKey == "g" && now.Sub(m.lastKeyAt) < keySequenceThreshold {
				m.moveTo(0)
				m.lastKey = ""
				m.lastKeyAt = time.Time{}
				return m, nil
			}
			m.lastKey = "g"
			m.lastKeyAt = now
			return m, nil
		case "]":
			m.moveTo(m.nextWorkflow())
		case "[":
			m.moveTo(m.previousWorkflow())
		case "enter", "space":
			m.expanded[m.cursor] = !m.expanded[m.cursor]
			m.ensureCursorVisible()
		case "e":
			for i := range m.items {
				m.expanded[i] = true
			}
			m.ensureCursorVisible()
		case "E":
			m.expanded = map[int]bool{}
			m.ensureCursorVisible()
		}
	}

	return m, nil
}

// moveTo moves the cursor to an item, clamped to the list.
func (m *Model) moveTo(index int) {
	m.cursor = max(0, min(index, len(m.items)-1))
	m.ensureCursorVisible()
}

// nextWorkflow returns the index of the next workflow row, or the current row if there isn't one.
func (m Model) nextWorkflow() int {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].Kind == KindWorkflow {
			return i
		}
	}
	return m.cursor
}

// previousWorkflow returns the index of the workflow row before the cursor. From a step that's its own workflow.
func (m Model) previousWorkflow() int {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].Kind == KindWorkflow {
			return i
		}
	}
	return m.cursor
}

// contentHeight is the number of lines available for rows, including the scroll indicators.
func (m Model) contentHeight() int {
	return max(1, m.height-headerLines-footerLines)
}

func (m Model) halfPage() int {
	return max(1, m.contentHeight()/2)
}

// itemHeight is the number of lines an item takes up.
func (m Model) itemHeight(index int) int {
	if m.expanded[index] {
		return 1 + len(m.items[index].Details)
	}
	return 1
}

// showIndicators reports whether the content area is tall enough for the scroll indicators as well as the cursor
// row.
func (m Model) showIndicators() bool {
	return m.contentHeight() >= 3
}

// ensureCursorVisible scrolls so the cursor row is on screen, leaving room for the scroll indicators.
func (m *Model) ensureCursorVisible() {
	if m.cursor < m.scrollOffset || m.cursor == 0 {
		m.scrollOffset = m.cursor
		return
	}

	// Keep a line free for the "more below" indicator, which replaces the last line when anything is cut off.
	available := m.contentHeight()
	if m.showIndicators() {
		available--
	}

	for m.scrollOffset < m.cursor && m.linesToCursor() > available {
		m.scrollOffset++
	}
}

// linesToCursor is the number of lines from the top of the content area to the cursor row, inclusive.
func (m Model) linesToCursor() int {
	lines := 1
	if m.scrollOffset > 0 && m.showIndicators() {
		lines++ // "more above" indicator
	}
	for i := m.scrollOffset; i < m.cursor; i++ {
		lines += m.itemHeight(i)
	}
	return lines
}
