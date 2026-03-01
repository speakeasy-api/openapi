package analyze

import "sort"

// SuggestionType categorizes the kind of refactoring suggestion.
type SuggestionType string

const (
	SuggestionCutEdge            SuggestionType = "cut-edge"
	SuggestionAddDiscriminator   SuggestionType = "add-discriminator"
	SuggestionSplitSCC           SuggestionType = "split-scc"
	SuggestionReducePropertyCount SuggestionType = "reduce-property-count"
)

// Suggestion is an actionable refactoring recommendation.
type Suggestion struct {
	// Type categorizes the suggestion.
	Type SuggestionType
	// Title is a short human-readable title.
	Title string
	// Description explains what to do and why.
	Description string
	// AffectedSchemas lists the schema IDs involved.
	AffectedSchemas []string
	// Impact estimates how many issues this would resolve (e.g., cycles broken).
	Impact int
	// Edge is the specific edge to cut (for cut-edge suggestions).
	Edge *Edge
}

// GenerateSuggestions produces actionable refactoring suggestions based on the analysis.
func GenerateSuggestions(g *Graph, cycles *CycleAnalysis, metrics map[string]*SchemaMetrics, codegen *CodegenReport) []*Suggestion {
	var suggestions []*Suggestion

	suggestions = append(suggestions, suggestCycleBreaks(g, cycles)...)
	suggestions = append(suggestions, suggestMissingDiscriminators(g, codegen)...)
	suggestions = append(suggestions, suggestSCCSplits(g, cycles)...)
	suggestions = append(suggestions, suggestPropertyReduction(metrics)...)

	// Sort by impact (highest first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Impact > suggestions[j].Impact
	})

	return suggestions
}

// suggestCycleBreaks finds the minimum set of edges whose removal would break the most cycles.
// Uses a greedy approximation: pick the edge that appears in the most cycles, remove it, repeat.
func suggestCycleBreaks(g *Graph, cycles *CycleAnalysis) []*Suggestion {
	if len(cycles.Cycles) == 0 {
		return nil
	}

	var suggestions []*Suggestion

	// Count how many cycles each edge participates in
	type edgeKey struct{ from, to string }
	edgeCycleCounts := make(map[edgeKey]int)
	edgeMap := make(map[edgeKey]*Edge)

	for _, cycle := range cycles.Cycles {
		for _, e := range cycle.Edges {
			key := edgeKey{e.From, e.To}
			edgeCycleCounts[key]++
			edgeMap[key] = e
		}
	}

	// Greedily pick edges that break the most cycles
	remaining := make(map[int]bool)
	for i := range cycles.Cycles {
		remaining[i] = true
	}

	for len(remaining) > 0 {
		// Find the edge in remaining cycles with the highest count
		best := edgeKey{}
		bestCount := 0

		counts := make(map[edgeKey]int)
		for i := range remaining {
			for _, e := range cycles.Cycles[i].Edges {
				key := edgeKey{e.From, e.To}
				counts[key]++
				if counts[key] > bestCount {
					bestCount = counts[key]
					best = key
				}
			}
		}

		if bestCount == 0 {
			break
		}

		edge := edgeMap[best]
		qualifier := "optional/nullable"
		if edge.IsRequired && !edge.IsNullable && !edge.IsArray {
			qualifier = "optional or nullable"
		}

		suggestions = append(suggestions, &Suggestion{
			Type:            SuggestionCutEdge,
			Title:           "Make " + edge.From + " → " + edge.To + " " + qualifier,
			Description:     describeEdgeCut(edge, bestCount),
			AffectedSchemas: []string{edge.From, edge.To},
			Impact:          bestCount,
			Edge:            edge,
		})

		// Remove cycles that contained this edge
		for i := range remaining {
			for _, e := range cycles.Cycles[i].Edges {
				if e.From == best.from && e.To == best.to {
					delete(remaining, i)
					break
				}
			}
		}
	}

	return suggestions
}

func describeEdgeCut(e *Edge, cyclesBroken int) string {
	desc := "Making the "
	switch e.Kind {
	case EdgeProperty:
		desc += "property '" + e.FieldName + "'"
	case EdgeItems:
		if e.FieldName != "" {
			desc += "items of '" + e.FieldName + "'"
		} else {
			desc += "items"
		}
	default:
		desc += string(e.Kind)
	}
	desc += " reference from " + e.From + " to " + e.To
	if e.IsRequired {
		desc += " optional (currently required)"
	} else {
		desc += " nullable"
	}
	desc += " would break "
	if cyclesBroken == 1 {
		desc += "1 cycle"
	} else {
		desc += itoa(cyclesBroken) + " cycles"
	}
	return desc
}

func suggestMissingDiscriminators(g *Graph, codegen *CodegenReport) []*Suggestion {
	var suggestions []*Suggestion

	for id, d := range codegen.PerSchema {
		for _, s := range d.Signals {
			if s.ID == "oneOf-no-discriminator" {
				suggestions = append(suggestions, &Suggestion{
					Type:            SuggestionAddDiscriminator,
					Title:           "Add discriminator to " + id,
					Description:     "Schema " + id + " uses oneOf without a discriminator. Adding a discriminator property enables code generators to produce efficient deserialization without trial-and-error.",
					AffectedSchemas: []string{id},
					Impact:          1,
				})
			}
		}
	}

	return suggestions
}

func suggestSCCSplits(g *Graph, cycles *CycleAnalysis) []*Suggestion {
	var suggestions []*Suggestion

	for _, scc := range cycles.SCCs {
		if scc.Size <= 2 {
			continue // Small SCCs are already apparent from cycle suggestions
		}

		// Find the edge whose removal would split this SCC
		// (the edge between the two nodes with the fewest other connections within the SCC)
		sccSet := make(map[string]bool, scc.Size)
		for _, id := range scc.NodeIDs {
			sccSet[id] = true
		}

		suggestions = append(suggestions, &Suggestion{
			Type:            SuggestionSplitSCC,
			Title:           "Consider splitting tightly-coupled group",
			Description:     "A group of " + itoa(scc.Size) + " schemas are all mutually reachable. This tight coupling may indicate an opportunity to extract a simpler interface or break the group into independent layers.",
			AffectedSchemas: scc.NodeIDs,
			Impact:          scc.Size,
		})
	}

	return suggestions
}

func suggestPropertyReduction(metrics map[string]*SchemaMetrics) []*Suggestion {
	var suggestions []*Suggestion

	for id, m := range metrics {
		if m.PropertyCount > 30 {
			suggestions = append(suggestions, &Suggestion{
				Type:            SuggestionReducePropertyCount,
				Title:           "Split " + id + " into smaller schemas",
				Description:     "Schema " + id + " has " + itoa(m.PropertyCount) + " properties. Consider grouping related properties into sub-schemas for better organization and reusability.",
				AffectedSchemas: []string{id},
				Impact:          1,
			})
		}
	}

	return suggestions
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
