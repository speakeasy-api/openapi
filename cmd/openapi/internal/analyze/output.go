package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// JSONReport is the JSON-serializable form of the analysis report.
type JSONReport struct {
	Document struct {
		Title      string `json:"title"`
		Version    string `json:"version"`
		OpenAPI    string `json:"openapi"`
	} `json:"document"`

	Summary struct {
		TotalSchemas       int     `json:"totalSchemas"`
		TotalEdges         int     `json:"totalEdges"`
		SCCCount           int     `json:"sccCount"`
		LargestSCCSize     int     `json:"largestSCCSize"`
		CycleCount         int     `json:"cycleCount"`
		SchemasInCyclesPct float64 `json:"schemasInCyclesPct"`
		RequiredOnlyCycles int     `json:"requiredOnlyCycles"`
		DAGDepth           int     `json:"dagDepth"`
		CompatibilityScore float64 `json:"compatibilityScore"`
		GreenCount         int     `json:"greenSchemas"`
		YellowCount        int     `json:"yellowSchemas"`
		RedCount           int     `json:"redSchemas"`
	} `json:"summary"`

	Schemas []JSONSchemaEntry   `json:"schemas"`
	Cycles  []JSONCycleEntry    `json:"cycles"`
	Suggestions []JSONSuggestion `json:"suggestions"`
}

// JSONSchemaEntry is the JSON form of per-schema analysis.
type JSONSchemaEntry struct {
	ID                string   `json:"id"`
	Types             []string `json:"types,omitempty"`
	PropertyCount     int      `json:"propertyCount"`
	DeepPropertyCount int      `json:"deepPropertyCount,omitempty"`
	FanIn             int      `json:"fanIn"`
	FanOut            int      `json:"fanOut"`
	NestingDepth      int      `json:"nestingDepth,omitempty"`
	CompositionDepth  int      `json:"compositionDepth,omitempty"`
	MaxUnionWidth     int      `json:"maxUnionWidth,omitempty"`
	VariantProduct    int      `json:"variantProduct,omitempty"`
	InSCC             bool     `json:"inSCC"`
	CycleCount        int      `json:"cycleCount"`
	ComplexityScore   int      `json:"complexityScore"`
	Rank              int      `json:"rank"`
	CodegenTier       string   `json:"codegenTier"`
	Signals           []string `json:"signals,omitempty"`
}

// JSONCycleEntry is the JSON form of a cycle.
type JSONCycleEntry struct {
	Path           []string `json:"path"`
	Length         int      `json:"length"`
	RequiredOnly   bool     `json:"requiredOnly"`
	BreakPointCount int    `json:"breakPointCount"`
}

// JSONSuggestion is the JSON form of a refactoring suggestion.
type JSONSuggestion struct {
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Schemas     []string `json:"affectedSchemas"`
	Impact      int      `json:"impact"`
}

// WriteJSON writes the report as JSON to the given writer.
func WriteJSON(w io.Writer, r *Report) error {
	jr := JSONReport{}
	jr.Document.Title = r.DocumentTitle
	jr.Document.Version = r.DocumentVersion
	jr.Document.OpenAPI = r.OpenAPIVersion

	jr.Summary.TotalSchemas = r.TotalSchemas
	jr.Summary.TotalEdges = r.TotalEdges
	jr.Summary.SCCCount = r.SCCCount
	jr.Summary.LargestSCCSize = r.LargestSCCSize
	jr.Summary.CycleCount = len(r.Cycles.Cycles)
	jr.Summary.SchemasInCyclesPct = r.SchemasInCyclesPct
	jr.Summary.RequiredOnlyCycles = r.RequiredOnlyCycles
	jr.Summary.DAGDepth = r.DAGDepth
	jr.Summary.CompatibilityScore = r.CompatibilityScore
	jr.Summary.GreenCount = r.Codegen.GreenCount
	jr.Summary.YellowCount = r.Codegen.YellowCount
	jr.Summary.RedCount = r.Codegen.RedCount

	// Schemas sorted by complexity
	ranked := TopSchemasByComplexity(r.Metrics, len(r.Metrics))
	for rank, sm := range ranked {
		entry := JSONSchemaEntry{
			ID:                sm.NodeID,
			Types:             sm.Types,
			PropertyCount:     sm.PropertyCount,
			DeepPropertyCount: sm.DeepPropertyCount,
			FanIn:             sm.FanIn,
			FanOut:            sm.FanOut,
			NestingDepth:      sm.NestingDepth,
			CompositionDepth:  sm.CompositionDepth,
			MaxUnionWidth:     sm.MaxUnionWidth,
			VariantProduct:    sm.VariantProduct,
			InSCC:             sm.InSCC,
			CycleCount:        sm.CycleMembership,
			ComplexityScore:   sm.ComplexityScore(),
			Rank:              rank + 1,
		}
		if d, ok := r.Codegen.PerSchema[sm.NodeID]; ok {
			entry.CodegenTier = d.Tier.String()
			for _, s := range d.Signals {
				entry.Signals = append(entry.Signals, s.ID)
			}
		}
		jr.Schemas = append(jr.Schemas, entry)
	}

	for _, c := range r.Cycles.Cycles {
		jr.Cycles = append(jr.Cycles, JSONCycleEntry{
			Path:            c.Path,
			Length:          c.Length,
			RequiredOnly:    c.HasRequiredOnlyPath,
			BreakPointCount: len(c.BreakPoints),
		})
	}

	for _, sg := range r.Suggestions {
		jr.Suggestions = append(jr.Suggestions, JSONSuggestion{
			Type:        string(sg.Type),
			Title:       sg.Title,
			Description: sg.Description,
			Schemas:     sg.AffectedSchemas,
			Impact:      sg.Impact,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}

// WriteDOT writes the schema reference graph in Graphviz DOT format.
func WriteDOT(w io.Writer, r *Report) {
	fmt.Fprintf(w, "digraph schemas {\n")
	fmt.Fprintf(w, "  rankdir=LR;\n")
	fmt.Fprintf(w, "  node [shape=box, style=filled, fontname=\"Helvetica\"];\n\n")

	// Nodes colored by tier
	ranked := TopSchemasByComplexity(r.Metrics, len(r.Metrics))
	for _, sm := range ranked {
		color := "#d4edda" // green default
		fontColor := "#155724"
		if d, ok := r.Codegen.PerSchema[sm.NodeID]; ok {
			switch d.Tier {
			case CodegenYellow:
				color = "#fff3cd"
				fontColor = "#856404"
			case CodegenRed:
				color = "#f8d7da"
				fontColor = "#721c24"
			}
		}

		label := fmt.Sprintf("%s\\nscore=%d fan-in=%d", sm.NodeID, sm.ComplexityScore(), sm.FanIn)
		fmt.Fprintf(w, "  %q [label=%q, fillcolor=%q, fontcolor=%q];\n",
			sm.NodeID, label, color, fontColor)
	}

	fmt.Fprintln(w)

	// Edges
	for _, e := range r.Graph.Edges {
		attrs := fmt.Sprintf("label=%q", string(e.Kind))
		if e.FieldName != "" {
			attrs = fmt.Sprintf("label=%q", string(e.Kind)+":"+e.FieldName)
		}
		if e.IsRequired {
			attrs += ", style=bold, color=red"
		}
		if e.IsArray {
			attrs += ", style=dashed"
		}
		fmt.Fprintf(w, "  %q -> %q [%s];\n", e.From, e.To, attrs)
	}

	fmt.Fprintf(w, "}\n")
}

// WriteText writes a human-readable text summary to the given writer.
func WriteText(w io.Writer, r *Report) {
	fmt.Fprintf(w, "Schema Complexity Report: %s v%s (OpenAPI %s)\n", r.DocumentTitle, r.DocumentVersion, r.OpenAPIVersion)
	fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 60))

	// Overview
	fmt.Fprintf(w, "OVERVIEW\n")
	fmt.Fprintf(w, "  Schemas: %d  Refs: %d\n\n", r.TotalSchemas, r.TotalEdges)

	// Cycle health
	fmt.Fprintf(w, "CYCLE HEALTH\n")
	fmt.Fprintf(w, "  SCCs: %d  Largest: %d  Cycles: %d\n", r.SCCCount, r.LargestSCCSize, len(r.Cycles.Cycles))
	fmt.Fprintf(w, "  Schemas in cycles: %.0f%%  Required-only cycles: %d\n", r.SchemasInCyclesPct, r.RequiredOnlyCycles)
	fmt.Fprintf(w, "  DAG depth: %d\n\n", r.DAGDepth)

	// Codegen
	fmt.Fprintf(w, "CODEGEN COMPATIBILITY\n")
	fmt.Fprintf(w, "  Score: %.0f%%  Green: %d  Yellow: %d  Red: %d\n\n",
		r.CompatibilityScore, r.Codegen.GreenCount, r.Codegen.YellowCount, r.Codegen.RedCount)

	// Top fan-in
	if len(r.TopFanIn) > 0 {
		fmt.Fprintf(w, "HIGHEST FAN-IN\n")
		for i, sm := range r.TopFanIn {
			if sm.FanIn == 0 {
				break
			}
			fmt.Fprintf(w, "  %d. %-30s %d refs\n", i+1, sm.NodeID, sm.FanIn)
		}
		fmt.Fprintln(w)
	}

	// Most complex
	if len(r.TopComplex) > 0 {
		fmt.Fprintf(w, "MOST COMPLEX\n")
		for i, sm := range r.TopComplex {
			score := sm.ComplexityScore()
			if score == 0 {
				break
			}
			fmt.Fprintf(w, "  %d. %-30s score=%-4d (fan-in=%d fan-out=%d props=%d depth=%d unions=%d)\n",
				i+1, sm.NodeID, score, sm.FanIn, sm.FanOut, sm.DeepPropertyCount, sm.CompositionDepth, sm.UnionSiteCount)
		}
		fmt.Fprintln(w)
	}

	// Red/yellow schemas
	var reds, yellows []string
	for id, d := range r.Codegen.PerSchema {
		switch d.Tier {
		case CodegenRed:
			reds = append(reds, id)
		case CodegenYellow:
			yellows = append(yellows, id)
		}
	}
	if len(reds) > 0 {
		fmt.Fprintf(w, "RED TIER SCHEMAS (%d)\n", len(reds))
		for _, id := range reds {
			d := r.Codegen.PerSchema[id]
			var sigs []string
			for _, s := range d.Signals {
				sigs = append(sigs, s.ID)
			}
			fmt.Fprintf(w, "  - %-30s [%s]\n", id, strings.Join(sigs, ", "))
		}
		fmt.Fprintln(w)
	}

	// Suggestions
	if len(r.Suggestions) > 0 {
		limit := 10
		if len(r.Suggestions) < limit {
			limit = len(r.Suggestions)
		}
		fmt.Fprintf(w, "SUGGESTIONS\n")
		for i := 0; i < limit; i++ {
			sg := r.Suggestions[i]
			fmt.Fprintf(w, "  %d. [%s] %s (impact: %d)\n", i+1, sg.Type, sg.Title, sg.Impact)
		}
	}
}
