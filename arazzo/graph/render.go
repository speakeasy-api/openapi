package graph

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Format is an output format for a graph.
type Format string

const (
	// FormatMermaid renders the graph as a Mermaid flowchart.
	FormatMermaid Format = "mermaid"
	// FormatDOT renders the graph as a Graphviz DOT digraph.
	FormatDOT Format = "dot"
)

const (
	successColor = "#2e7d32"
	failureColor = "#c62828"
	dependsColor = "#757575"
)

// Write renders the graph in the given format.
func (g *Graph) Write(w io.Writer, format Format) error {
	switch format {
	case FormatMermaid:
		return g.WriteMermaid(w)
	case FormatDOT:
		return g.WriteDOT(w)
	default:
		return fmt.Errorf("unknown format `%s`, expected %s or %s", format, FormatMermaid, FormatDOT)
	}
}

// WriteMermaid renders the graph as a Mermaid flowchart. Each workflow is a subgraph, success gotos are green, failure
// gotos and retries are dashed red, dependsOn edges are dashed grey and steps running a workflow are thick.
func (g *Graph) WriteMermaid(w io.Writer) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "flowchart %s\n", g.Direction)

	for _, wf := range g.Workflows {
		fmt.Fprintf(&sb, "  subgraph %s [\"%s\"]\n", wf.ID, mermaidText(wf.Title))
		fmt.Fprintf(&sb, "    %s([\"%s\"])\n", wf.Start.ID, mermaidText(wf.Start.Label))
		for _, s := range wf.Steps {
			label := mermaidText(s.Label)
			if s.Detail != "" {
				label += "<br/>" + mermaidText(s.Detail)
			}
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", s.ID, label)
		}
		sb.WriteString("  end\n")
	}

	externalIDs := make([]string, 0, len(g.External))
	for _, n := range g.External {
		fmt.Fprintf(&sb, "  %s{{\"%s\"}}\n", n.ID, mermaidText(n.Label))
		externalIDs = append(externalIDs, n.ID)
	}

	styles := map[string][]string{}
	styleOrder := []string{}
	addStyle := func(style string, index int) {
		if _, ok := styles[style]; !ok {
			styleOrder = append(styleOrder, style)
		}
		styles[style] = append(styles[style], strconv.Itoa(index))
	}

	for i, e := range g.Edges {
		arrow := "-->"
		switch e.Kind {
		case EdgeSuccessGoto:
			addStyle("stroke:"+successColor, i)
		case EdgeFailureGoto, EdgeRetry:
			arrow = "-.->"
			addStyle("stroke:"+failureColor, i)
		case EdgeDependsOn:
			arrow = "-.->"
			addStyle("stroke:"+dependsColor, i)
		case EdgeCall:
			arrow = "==>"
		case EdgeStart, EdgeNext:
		}

		if e.Label == "" {
			fmt.Fprintf(&sb, "  %s %s %s\n", e.From.ID, arrow, e.To.ID)
		} else {
			fmt.Fprintf(&sb, "  %s %s|\"%s\"| %s\n", e.From.ID, arrow, mermaidText(e.Label), e.To.ID)
		}
	}

	for _, style := range styleOrder {
		fmt.Fprintf(&sb, "  linkStyle %s %s\n", strings.Join(styles[style], ","), style)
	}
	if len(externalIDs) > 0 {
		sb.WriteString("  classDef external stroke-dasharray: 5 5\n")
		fmt.Fprintf(&sb, "  class %s external\n", strings.Join(externalIDs, ","))
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// WriteDOT renders the graph as a Graphviz DOT digraph. Each workflow is a cluster, success gotos are green, failure
// gotos and retries are dashed red, dependsOn edges are dotted grey and steps running a workflow are bold.
func (g *Graph) WriteDOT(w io.Writer) error {
	var sb strings.Builder
	sb.WriteString("digraph arazzo {\n")
	fmt.Fprintf(&sb, "  rankdir=%s;\n", g.Direction)
	sb.WriteString("  node [shape=box, style=rounded, fontname=\"Helvetica\"];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\", fontsize=10];\n")

	for _, wf := range g.Workflows {
		fmt.Fprintf(&sb, "  subgraph cluster_%s {\n", wf.ID)
		fmt.Fprintf(&sb, "    label=%s;\n", dotString(wf.Title))
		sb.WriteString("    style=rounded;\n")
		fmt.Fprintf(&sb, "    %s [label=%s, shape=oval];\n", wf.Start.ID, dotString(wf.Start.Label))
		for _, s := range wf.Steps {
			label := s.Label
			if s.Detail != "" {
				label += "\n" + s.Detail
			}
			fmt.Fprintf(&sb, "    %s [label=%s];\n", s.ID, dotString(label))
		}
		sb.WriteString("  }\n")
	}

	for _, n := range g.External {
		fmt.Fprintf(&sb, "  %s [label=%s, style=\"rounded,dashed\"];\n", n.ID, dotString(n.Label))
	}

	for _, e := range g.Edges {
		attrs := []string{}
		if e.Label != "" {
			attrs = append(attrs, "label="+dotString(e.Label))
		}
		switch e.Kind {
		case EdgeSuccessGoto:
			attrs = append(attrs, `color="`+successColor+`"`)
		case EdgeFailureGoto, EdgeRetry:
			attrs = append(attrs, `color="`+failureColor+`"`, "style=dashed")
		case EdgeDependsOn:
			attrs = append(attrs, `color="`+dependsColor+`"`, "style=dotted")
		case EdgeCall:
			attrs = append(attrs, "style=bold")
		case EdgeStart, EdgeNext:
		}

		if len(attrs) == 0 {
			fmt.Fprintf(&sb, "  %s -> %s;\n", e.From.ID, e.To.ID)
		} else {
			fmt.Fprintf(&sb, "  %s -> %s [%s];\n", e.From.ID, e.To.ID, strings.Join(attrs, ", "))
		}
	}

	sb.WriteString("}\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

var (
	mermaidReplacer = strings.NewReplacer("#", "#35;", `"`, "#quot;", "<", "#lt;", ">", "#gt;", "\r\n", " ", "\r", " ", "\n", " ")
	dotReplacer     = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\r\n", `\n`, "\r", `\n`, "\n", `\n`)
)

// mermaidText escapes text for use within a quoted Mermaid label.
func mermaidText(s string) string {
	return mermaidReplacer.Replace(s)
}

// dotString returns s as a quoted DOT string.
func dotString(s string) string {
	return `"` + dotReplacer.Replace(s) + `"`
}
