package oq

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/oq/expr"
	"gopkg.in/yaml.v3"
)

// FormatTable formats a result as a simple table string.
func FormatTable(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	syncGroupsFromRows(result)

	// Use group-specific formatting only when no explicit field projection
	if len(result.Groups) > 0 && len(result.Fields) == 0 {
		return formatGroups(result)
	}

	if len(result.Rows) == 0 {
		return "(empty)"
	}

	fields := result.Fields
	if len(fields) == 0 {
		fields = defaultFieldsForKind(result.Rows[0].Kind)
	}

	// Build header
	widths := make([]int, len(fields))
	for i, f := range fields {
		widths[i] = len(f)
	}

	// Collect rows
	var tableRows [][]string
	for _, row := range result.Rows {
		var cols []string
		for i, f := range fields {
			v := valueToString(fieldValue(row, f, g))
			cols = append(cols, v)
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
		tableRows = append(tableRows, cols)
	}

	// Format
	var sb strings.Builder
	// Header
	for i, f := range fields {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(padRight(f, widths[i]))
	}
	sb.WriteString("\n")
	// Separator
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("-", w))
	}
	sb.WriteString("\n")
	// Data
	for _, row := range tableRows {
		for i, col := range row {
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(padRight(col, widths[i]))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatJSON formats a result as JSON.
func FormatJSON(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	syncGroupsFromRows(result)

	if len(result.Groups) > 0 && len(result.Fields) == 0 {
		return formatGroupsJSON(result)
	}

	if len(result.Rows) == 0 {
		return "[]"
	}

	fields := result.Fields
	if len(fields) == 0 {
		fields = defaultFieldsForKind(result.Rows[0].Kind)
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, row := range result.Rows {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString("  {")
		for j, f := range fields {
			if j > 0 {
				sb.WriteString(", ")
			}
			v := fieldValue(row, f, g)
			fmt.Fprintf(&sb, "%q: %s", f, jsonValue(v))
		}
		sb.WriteString("}")
	}
	sb.WriteString("\n]")
	return sb.String()
}

// FormatMarkdown formats a result as a markdown table.
func FormatMarkdown(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	syncGroupsFromRows(result)

	if len(result.Groups) > 0 && len(result.Fields) == 0 {
		var sb strings.Builder
		sb.WriteString("| Key | Count |\n")
		sb.WriteString("| --- | --- |\n")
		for _, grp := range result.Groups {
			fmt.Fprintf(&sb, "| %s | %d |\n", grp.Key, grp.Count)
		}
		return sb.String()
	}

	if len(result.Rows) == 0 {
		return "(empty)"
	}

	fields := result.Fields
	if len(fields) == 0 {
		fields = defaultFieldsForKind(result.Rows[0].Kind)
	}

	var sb strings.Builder
	// Header
	sb.WriteString("| ")
	sb.WriteString(strings.Join(fields, " | "))
	sb.WriteString(" |\n")
	// Separator
	sb.WriteString("|")
	for range fields {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")
	// Rows
	for _, row := range result.Rows {
		sb.WriteString("| ")
		for i, f := range fields {
			if i > 0 {
				sb.WriteString(" | ")
			}
			v := valueToString(fieldValue(row, f, g))
			sb.WriteString(v)
		}
		sb.WriteString(" |\n")
	}

	return sb.String()
}

// FormatToon formats a result in the TOON (Token-Oriented Object Notation) format.
// TOON uses tabular array syntax for uniform rows: header[N]{field1,field2,...}:
// followed by comma-delimited data rows. See https://github.com/toon-format/toon
func FormatToon(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return "count: " + strconv.Itoa(result.Count)
	}

	syncGroupsFromRows(result)

	if len(result.Groups) > 0 && len(result.Fields) == 0 {
		return formatGroupsToon(result)
	}

	if len(result.Rows) == 0 {
		return "results[0]:\n"
	}

	fields := result.Fields
	if len(fields) == 0 {
		fields = defaultFieldsForKind(result.Rows[0].Kind)
	}

	var sb strings.Builder

	// Header: results[N]{field1,field2,...}:
	fmt.Fprintf(&sb, "results[%d]{%s}:\n", len(result.Rows), strings.Join(fields, ","))

	// Data rows: comma-separated values, indented by one space
	for _, row := range result.Rows {
		sb.WriteByte(' ')
		for i, f := range fields {
			if i > 0 {
				sb.WriteByte(',')
			}
			v := fieldValue(row, f, g)
			sb.WriteString(toonValue(v))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// FormatYAML formats results as raw YAML from the underlying schema/operation objects.
// For multiple results, outputs a YAML stream with --- separators.
// This enables piping into yq for content-level queries.
func FormatYAML(result *Result, g *graph.SchemaGraph) string {
	if result.Explain != "" {
		return result.Explain
	}

	if result.IsCount {
		return strconv.Itoa(result.Count)
	}

	syncGroupsFromRows(result)

	if len(result.Groups) > 0 && len(result.Fields) == 0 {
		return formatGroupsJSON(result)
	}

	if len(result.Rows) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, row := range result.Rows {
		if i > 0 {
			sb.WriteString("---\n")
		}

		node := getRootNode(row, g)
		// Use path for full attribution; fall back to name for non-schema rows
		key := valueToString(fieldValue(row, "path", g))
		if key == "" {
			key = valueToString(fieldValue(row, "name", g))
		}

		if node == nil {
			sb.WriteString("# ")
			sb.WriteString(key)
			sb.WriteString(" (no YAML node available)\n")
			continue
		}

		wrapper := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: key},
				node,
			},
		}
		data, err := yaml.Marshal(wrapper)
		if err != nil {
			sb.WriteString("# error marshalling: " + err.Error() + "\n")
			continue
		}
		sb.Write(data)
	}

	return sb.String()
}

// getRootNode extracts the underlying yaml.Node from a result row.
func getRootNode(row Row, g *graph.SchemaGraph) *yaml.Node {
	switch row.Kind {
	case SchemaResult:
		if row.SchemaIdx < 0 || row.SchemaIdx >= len(g.Schemas) {
			return nil
		}
		s := &g.Schemas[row.SchemaIdx]
		if s.Schema == nil {
			return nil
		}
		schema := s.Schema.GetSchema()
		if schema == nil {
			return nil
		}
		return schema.GetRootNode()
	case OperationResult:
		if row.OpIdx < 0 || row.OpIdx >= len(g.Operations) {
			return nil
		}
		o := &g.Operations[row.OpIdx]
		if o.Operation == nil {
			return nil
		}
		return o.Operation.GetRootNode()
	default:
		return nil
	}
}

func formatGroupsToon(result *Result) string {
	var sb strings.Builder

	// Groups as tabular array
	fmt.Fprintf(&sb, "groups[%d]{key,count,names}:\n", len(result.Groups))
	for _, grp := range result.Groups {
		names := strings.Join(grp.Names, ";")
		fmt.Fprintf(&sb, " %s,%d,%s\n", toonEscape(grp.Key), grp.Count, toonEscape(names))
	}
	return sb.String()
}

// toonValue encodes an expr.Value for TOON format.
func toonValue(v expr.Value) string {
	switch v.Kind {
	case expr.KindString:
		return toonEscape(v.Str)
	case expr.KindInt:
		return strconv.Itoa(v.Int)
	case expr.KindBool:
		return strconv.FormatBool(v.Bool)
	default:
		return "null"
	}
}

// toonEscape quotes a string if it needs escaping for TOON format.
// A string must be quoted if it: is empty, contains comma/colon/quote/backslash/
// brackets/braces/control chars, has leading/trailing whitespace, or matches
// true/false/null or a numeric pattern.
func toonEscape(s string) string {
	if s == "" {
		return `""`
	}
	if s == "true" || s == "false" || s == "null" {
		return `"` + s + `"`
	}
	// Check if it looks numeric
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return `"` + s + `"`
	}
	needsQuote := false
	for _, ch := range s {
		if ch == ',' || ch == ':' || ch == '"' || ch == '\\' ||
			ch == '[' || ch == ']' || ch == '{' || ch == '}' ||
			ch == '\n' || ch == '\r' || ch == '\t' ||
			ch < 0x20 {
			needsQuote = true
			break
		}
	}
	if s[0] == ' ' || s[len(s)-1] == ' ' {
		needsQuote = true
	}
	if !needsQuote {
		return s
	}
	// Quote with escaping
	var sb strings.Builder
	sb.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(ch)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func jsonValue(v expr.Value) string {
	switch v.Kind {
	case expr.KindString:
		return fmt.Sprintf("%q", v.Str)
	case expr.KindInt:
		return strconv.Itoa(v.Int)
	case expr.KindBool:
		return strconv.FormatBool(v.Bool)
	default:
		return "null"
	}
}

func formatGroups(result *Result) string {
	var sb strings.Builder
	for _, g := range result.Groups {
		fmt.Fprintf(&sb, "%s: count=%d", g.Key, g.Count)
		if len(g.Names) > 0 {
			names := slices.Clone(g.Names)
			if len(names) > 5 {
				names = names[:5]
				names = append(names, "...")
			}
			fmt.Fprintf(&sb, " names=[%s]", strings.Join(names, ", "))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatGroupsJSON(result *Result) string {
	var sb strings.Builder
	sb.WriteString("[\n")
	for i, g := range result.Groups {
		if i > 0 {
			sb.WriteString(",\n")
		}
		fmt.Fprintf(&sb, `  {"key": %q, "count": %d, "names": [`, g.Key, g.Count)
		for j, n := range g.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%q", n)
		}
		sb.WriteString("]}")
	}
	sb.WriteString("\n]")
	return sb.String()
}

func defaultFieldsForKind(kind ResultKind) []string {
	switch kind {
	case SchemaResult:
		return []string{"name", "type", "depth", "in_degree", "out_degree"}
	case OperationResult:
		return []string{"name", "method", "path", "schema_count"}
	case GroupRowResult:
		return []string{"key", "count", "names"}
	case ParameterResult:
		return []string{"name", "in", "required", "deprecated", "operation"}
	case ResponseResult:
		return []string{"status_code", "description", "content_type_count", "operation"}
	case RequestBodyResult:
		return []string{"description", "required", "content_type_count", "operation"}
	case ContentTypeResult:
		return []string{"media_type", "has_schema", "status_code", "operation"}
	case HeaderResult:
		return []string{"name", "required", "status_code", "operation"}
	default:
		return []string{"name"}
	}
}

// syncGroupsFromRows rebuilds result.Groups from GroupRowResult rows.
// This ensures Groups stays in sync after filtering/sorting/limiting.
func syncGroupsFromRows(result *Result) {
	hasGroupRows := false
	for _, row := range result.Rows {
		if row.Kind == GroupRowResult {
			hasGroupRows = true
			break
		}
	}
	if !hasGroupRows {
		return
	}
	result.Groups = nil
	for _, row := range result.Rows {
		if row.Kind == GroupRowResult {
			result.Groups = append(result.Groups, GroupResult{
				Key:   row.GroupKey,
				Count: row.GroupCount,
				Names: row.GroupNames,
			})
		}
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
