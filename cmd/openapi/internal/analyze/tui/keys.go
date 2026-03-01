package tui

// Tab identifiers
type Tab int

const (
	TabSummary Tab = iota
	TabSchemas
	TabCycles
	TabGraph
	TabSuggestions
)

var tabNames = []string{"Summary", "Schemas", "Cycles", "Graph", "Suggestions"}

func (t Tab) String() string {
	if int(t) < len(tabNames) {
		return tabNames[t]
	}
	return "Unknown"
}
