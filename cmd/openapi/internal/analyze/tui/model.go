package tui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
)

const (
	keySequenceThreshold  = 500 * time.Millisecond
	scrollHalfScreenLines = 21
	headerApproxLines     = 3
	footerApproxLines     = 2
	layoutBuffer          = 2
)

var schemaSortModes = []string{"complexity", "name", "fan-in", "fan-out", "tier"}

// Model is the top-level bubbletea model for the schema complexity analyzer TUI.
type Model struct {
	report *analyze.Report

	// UI state
	activeTab    Tab
	cursor       int
	scrollOffset int
	width        int
	height       int
	showHelp     bool
	expanded     map[int]bool // expanded items in list views

	// Schema list state
	schemaFilter   string         // "" = all, "red", "yellow"
	schemaSortMode int            // index into schemaSortModes
	schemaItems    []string
	schemaRanks    map[string]int // cached complexity ranks

	// Cycle list state
	cycleSelected int

	// Graph view state
	graphMode    int               // 0=DAG overview, 1=SCC gallery, 2=ego graph
	graphSCCIdx  int               // current SCC index for gallery mode
	graphEgoNode string            // node ID for ego graph
	graphEgoHops int               // hop radius (default 2)
	graphCache   map[string]string // cache rendered ASCII art
	graphItems   []string          // selectable node list for current graph mode
	graphCursor  int               // cursor within graphItems

	// Key sequence handling
	lastKey   string
	lastKeyAt time.Time

	quitting bool
}

// NewModel creates a new TUI model from an analysis report.
func NewModel(report *analyze.Report) Model {
	// Pre-compute complexity ranks
	ranked := analyze.TopSchemasByComplexity(report.Metrics, len(report.Metrics))
	ranks := make(map[string]int, len(ranked))
	for i, r := range ranked {
		ranks[r.NodeID] = i + 1
	}

	m := Model{
		report:      report,
		width:       80,
		height:      24,
		expanded:    make(map[int]bool),
		schemaRanks: ranks,

		graphEgoHops: 2,
		graphCache:   make(map[string]string),
	}
	m.rebuildSchemaItems()
	m.rebuildGraphItems()
	return m
}

func (m Model) schemaRank(nodeID string) int {
	if r, ok := m.schemaRanks[nodeID]; ok {
		return r
	}
	return 0
}

func (m *Model) rebuildGraphItems() {
	m.graphItems = nil
	m.graphCursor = 0

	switch m.graphMode {
	case GraphModeDAG:
		// List all schemas grouped by DAG layer
		dag := m.report.Cycles.DAGCondensation
		if dag == nil {
			break
		}
		for _, layer := range dag.Layers {
			for _, sccIdx := range layer {
				if sccIdx >= len(dag.Nodes) {
					continue
				}
				scc := dag.Nodes[sccIdx]
				for _, id := range scc.NodeIDs {
					m.graphItems = append(m.graphItems, id)
				}
			}
		}

	case GraphModeSCC:
		// List nodes in the current SCC
		sccs := m.report.Cycles.SCCs
		if len(sccs) == 0 {
			break
		}
		idx := m.graphSCCIdx
		if idx >= len(sccs) {
			idx = len(sccs) - 1
		}
		m.graphItems = append(m.graphItems, sccs[idx].NodeIDs...)

	case GraphModeEgo:
		if m.graphEgoNode == "" {
			break
		}
		g := m.report.Graph
		// BFS neighborhood
		visited := map[string]int{m.graphEgoNode: 0}
		queue := []string{m.graphEgoNode}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			dist := visited[current]
			if dist >= m.graphEgoHops {
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
		// Center first, then neighbors sorted
		m.graphItems = append(m.graphItems, m.graphEgoNode)
		var neighbors []string
		for id := range visited {
			if id != m.graphEgoNode {
				neighbors = append(neighbors, id)
			}
		}
		sort.Strings(neighbors)
		m.graphItems = append(m.graphItems, neighbors...)
	}
}

func (m *Model) rebuildSchemaItems() {
	m.schemaItems = nil

	var ranked []*analyze.SchemaMetrics
	sortMode := schemaSortModes[m.schemaSortMode]
	switch sortMode {
	case "name":
		ranked = analyze.TopSchemasByName(m.report.Metrics, len(m.report.Metrics))
	case "fan-in":
		ranked = analyze.TopSchemasByFanIn(m.report.Metrics, len(m.report.Metrics))
	case "fan-out":
		ranked = analyze.TopSchemasByFanOut(m.report.Metrics, len(m.report.Metrics))
	case "tier":
		ranked = analyze.TopSchemasByTier(m.report.Metrics, m.report.Codegen, len(m.report.Metrics))
	default:
		ranked = analyze.TopSchemasByComplexity(m.report.Metrics, len(m.report.Metrics))
	}

	for _, sm := range ranked {
		if m.schemaFilter != "" {
			d := m.report.Codegen.PerSchema[sm.NodeID]
			if d == nil {
				continue
			}
			switch m.schemaFilter {
			case "red":
				if d.Tier != analyze.CodegenRed {
					continue
				}
			case "yellow":
				if d.Tier != analyze.CodegenYellow && d.Tier != analyze.CodegenRed {
					continue
				}
			}
		}
		m.schemaItems = append(m.schemaItems, sm.NodeID)
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case "q", "esc", "?":
				m.showHelp = false
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "?":
			m.showHelp = true

		case "tab", "l":
			m.activeTab = (m.activeTab + 1) % Tab(len(tabNames))
			m.cursor = 0
			m.scrollOffset = 0
			m.expanded = make(map[int]bool)

		case "shift+tab", "h":
			if m.activeTab == 0 {
				m.activeTab = Tab(len(tabNames) - 1)
			} else {
				m.activeTab--
			}
			m.cursor = 0
			m.scrollOffset = 0
			m.expanded = make(map[int]bool)

		case "up", "k":
			if m.activeTab == TabGraph {
				if m.graphCursor > 0 {
					m.graphCursor--
				}
			} else if m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
			}

		case "down", "j":
			if m.activeTab == TabGraph {
				if m.graphCursor < len(m.graphItems)-1 {
					m.graphCursor++
				}
			} else {
				maxItem := m.maxCursorForTab()
				if m.cursor < maxItem {
					m.cursor++
					m.ensureCursorVisible()
				}
			}

		case "enter", " ":
			if m.activeTab == TabSchemas && m.cursor < len(m.schemaItems) {
				m.expanded[m.cursor] = !m.expanded[m.cursor]
				if m.expanded[m.cursor] {
					// Snap scroll so the expanded item starts at the top
					m.scrollOffset = m.cursor
				} else {
					m.ensureCursorVisible()
				}
				// Also set ego graph node for quick switching to graph tab
				m.graphEgoNode = m.schemaItems[m.cursor]
				m.graphCache = make(map[string]string)
			} else if m.activeTab == TabGraph && m.graphCursor < len(m.graphItems) {
				selected := m.graphItems[m.graphCursor]
				// Navigate into ego graph for selected node
				m.graphEgoNode = selected
				m.graphMode = GraphModeEgo
				m.graphCache = make(map[string]string)
				m.rebuildGraphItems()
			} else {
				m.expanded[m.cursor] = !m.expanded[m.cursor]
				if m.expanded[m.cursor] {
					m.scrollOffset = m.cursor
				} else {
					m.ensureCursorVisible()
				}
			}

		case "m":
			if m.activeTab == TabGraph {
				m.graphMode = (m.graphMode + 1) % graphModeCount
				m.graphCache = make(map[string]string)
				m.rebuildGraphItems()
			}

		case "n":
			if m.activeTab == TabGraph && m.graphMode == GraphModeSCC {
				if m.graphSCCIdx < len(m.report.Cycles.SCCs)-1 {
					m.graphSCCIdx++
					m.graphCache = make(map[string]string)
					m.rebuildGraphItems()
				}
			}

		case "p":
			if m.activeTab == TabGraph && m.graphMode == GraphModeSCC {
				if m.graphSCCIdx > 0 {
					m.graphSCCIdx--
					m.graphCache = make(map[string]string)
					m.rebuildGraphItems()
				}
			}

		case "+", "=":
			if m.activeTab == TabGraph && m.graphMode == GraphModeEgo {
				if m.graphEgoHops < 5 {
					m.graphEgoHops++
					m.graphCache = make(map[string]string)
					m.rebuildGraphItems()
				}
			}

		case "-":
			if m.activeTab == TabGraph && m.graphMode == GraphModeEgo {
				if m.graphEgoHops > 1 {
					m.graphEgoHops--
					m.graphCache = make(map[string]string)
					m.rebuildGraphItems()
				}
			}

		case "ctrl+d":
			maxItem := m.maxCursorForTab()
			newPos := m.cursor + scrollHalfScreenLines
			if newPos > maxItem {
				m.cursor = maxItem
			} else {
				m.cursor = newPos
			}
			m.ensureCursorVisible()

		case "ctrl+u":
			newPos := m.cursor - scrollHalfScreenLines
			if newPos < 0 {
				m.cursor = 0
			} else {
				m.cursor = newPos
			}
			m.ensureCursorVisible()

		case "G":
			m.cursor = m.maxCursorForTab()
			m.ensureCursorVisible()

		case "g":
			now := time.Now()
			if m.lastKey == "g" && now.Sub(m.lastKeyAt) < keySequenceThreshold {
				m.cursor = 0
				m.scrollOffset = 0
				m.lastKey = ""
				m.lastKeyAt = time.Time{}
			} else {
				m.lastKey = "g"
				m.lastKeyAt = now
			}

		case "s":
			if m.activeTab == TabSchemas {
				m.schemaSortMode = (m.schemaSortMode + 1) % len(schemaSortModes)
				m.rebuildSchemaItems()
				m.cursor = 0
				m.scrollOffset = 0
			}

		case "f":
			if m.activeTab == TabSchemas {
				switch m.schemaFilter {
				case "":
					m.schemaFilter = "yellow"
				case "yellow":
					m.schemaFilter = "red"
				case "red":
					m.schemaFilter = ""
				}
				m.rebuildSchemaItems()
				m.cursor = 0
				m.scrollOffset = 0
			}

		case "1":
			m.activeTab = TabSummary
			m.cursor = 0
			m.scrollOffset = 0
		case "2":
			m.activeTab = TabSchemas
			m.cursor = 0
			m.scrollOffset = 0
		case "3":
			m.activeTab = TabCycles
			m.cursor = 0
			m.scrollOffset = 0
		case "4":
			m.activeTab = TabGraph
			m.cursor = 0
			m.scrollOffset = 0
			m.graphCursor = 0
		case "5":
			m.activeTab = TabSuggestions
			m.cursor = 0
			m.scrollOffset = 0
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	var s strings.Builder

	header := m.renderTabBar()
	s.WriteString(header)

	var content string
	switch m.activeTab {
	case TabSummary:
		content = m.renderSummary()
	case TabSchemas:
		content = m.renderSchemaList()
	case TabCycles:
		content = m.renderCycleList()
	case TabGraph:
		content = m.renderGraphView()
	case TabSuggestions:
		content = m.renderSuggestionList()
	}

	s.WriteString(content)

	footer := m.renderFooter()
	headerLines := strings.Count(header, "\n")
	contentLines := strings.Count(content, "\n")
	footerLines := strings.Count(footer, "\n")
	remaining := m.height - headerLines - contentLines - footerLines - 1
	if remaining > 0 {
		s.WriteString(strings.Repeat("\n", remaining))
	}
	s.WriteString(footer)

	return s.String()
}

func (m Model) maxCursorForTab() int {
	switch m.activeTab {
	case TabSchemas:
		if len(m.schemaItems) == 0 {
			return 0
		}
		return len(m.schemaItems) - 1
	case TabCycles:
		if len(m.report.Cycles.Cycles) == 0 {
			return 0
		}
		return len(m.report.Cycles.Cycles) - 1
	case TabSuggestions:
		if len(m.report.Suggestions) == 0 {
			return 0
		}
		return len(m.report.Suggestions) - 1
	default:
		return 0
	}
}

func (m Model) contentHeight() int {
	return max(1, m.height-headerApproxLines-footerApproxLines-layoutBuffer)
}

func (m *Model) ensureCursorVisible() {
	contentH := m.contentHeight()

	if m.cursor == 0 {
		m.scrollOffset = 0
		return
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
		return
	}

	linesUsed := 0
	for i := m.scrollOffset; i <= m.cursor; i++ {
		linesUsed += m.itemHeight(i)
	}

	if linesUsed > contentH {
		for newOffset := m.scrollOffset + 1; newOffset <= m.cursor; newOffset++ {
			test := 0
			for i := newOffset; i <= m.cursor; i++ {
				test += m.itemHeight(i)
			}
			if test <= contentH {
				m.scrollOffset = newOffset
				break
			}
		}
	}
}

func (m Model) itemHeight(index int) int {
	if !m.expanded[index] {
		return 1
	}

	switch m.activeTab {
	case TabSchemas:
		// Estimate card height based on content
		h := 12 // base: title, tier, types, props, fan, complexity, border
		if index < len(m.schemaItems) {
			id := m.schemaItems[index]
			if d, ok := m.report.Codegen.PerSchema[id]; ok && len(d.Signals) > 0 {
				h += len(d.Signals) + 1
			}
			if edges := m.report.Graph.OutEdges[id]; len(edges) > 0 {
				h += len(edges) + 1
			}
		}
		return h
	case TabSuggestions:
		return 4 // description + schemas + blank
	case TabCycles:
		if index < len(m.report.Cycles.Cycles) {
			return m.report.Cycles.Cycles[index].Length + 3
		}
	}
	return 6
}
