package oq

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// declarations holds parsed includes, defs, and the raw remaining pipeline text.
type declarations struct {
	Includes     []string
	Defs         []FuncDef
	PipelineText string
}

// parseDeclarations scans for include/def declarations at the start of a query.
func parseDeclarations(query string) (*declarations, error) {
	d := &declarations{}
	remaining := strings.TrimSpace(query)

	for {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}
		if strings.HasPrefix(remaining, "include ") {
			rest := remaining[len("include "):]
			semi := findUnquotedSemicolon(rest)
			if semi < 0 {
				return nil, errors.New("include missing terminating ;")
			}
			path := strings.TrimSpace(rest[:semi])
			path = strings.Trim(path, "\"")
			if path == "" {
				return nil, errors.New("include requires a path")
			}
			d.Includes = append(d.Includes, path)
			remaining = rest[semi+1:]
			continue
		}
		if strings.HasPrefix(remaining, "def ") {
			rest := remaining[len("def "):]
			colonIdx := strings.Index(rest, ":")
			if colonIdx < 0 {
				return nil, errors.New("def missing colon separator")
			}
			sig := strings.TrimSpace(rest[:colonIdx])
			body := rest[colonIdx+1:]
			semi := findUnquotedSemicolon(body)
			if semi < 0 {
				return nil, errors.New("def missing terminating ;")
			}
			bodyStr := strings.TrimSpace(body[:semi])
			remaining = body[semi+1:]

			fd, err := parseFuncSig(sig)
			if err != nil {
				return nil, err
			}
			fd.Body = bodyStr
			d.Defs = append(d.Defs, fd)
			continue
		}
		break
	}

	d.PipelineText = remaining
	return d, nil
}

// ParseQuery parses a full query string including optional includes, defs, and pipeline.
func ParseQuery(query string) (*Query, error) {
	d, err := parseDeclarations(query)
	if err != nil {
		return nil, err
	}

	q := &Query{
		Includes: d.Includes,
		Defs:     d.Defs,
	}

	if d.PipelineText == "" {
		if len(q.Defs) > 0 || len(q.Includes) > 0 {
			return q, nil
		}
		return nil, errors.New("empty query")
	}

	// Expand defs at text level before parsing
	expanded, err := ExpandDefs(d.PipelineText, d.Defs)
	if err != nil {
		return nil, err
	}

	stages, err := parsePipeline(expanded)
	if err != nil {
		return nil, err
	}
	q.Stages = stages
	return q, nil
}

// Parse splits a pipeline query string into stages.
func Parse(query string) ([]Stage, error) {
	q, err := ParseQuery(query)
	if err != nil {
		return nil, err
	}
	return q.Stages, nil
}

func parsePipeline(query string) ([]Stage, error) {
	parts := splitPipeline(query)
	if len(parts) == 0 {
		return nil, errors.New("empty query")
	}

	var stages []Stage

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if i == 0 {
			// Allow path(A, B) as a source — it doesn't need an input set
			if stage, err := parseStage(part); err == nil && stage.Kind == StagePath {
				stages = append(stages, stage)
				continue
			}
			stages = append(stages, Stage{Kind: StageSource, Source: part})
			continue
		}

		stage, err := parseStage(part)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}

	return stages, nil
}

func parseStage(s string) (Stage, error) {
	// Try keyword-call syntax first: where(...), sort-by(...), etc.
	keyword, args, isCall := splitKeywordCall(s)
	if !isCall {
		keyword, args = splitFirst(s)
	}
	keyword = strings.ToLower(keyword)

	switch keyword {
	case "where":
		if !isCall {
			return Stage{}, errors.New("where requires parentheses: where(expr)")
		}
		if args == "" {
			return Stage{}, errors.New("where() requires an expression")
		}
		return Stage{Kind: StageWhere, Expr: args}, nil

	case "select":
		if isCall {
			return Stage{}, errors.New("select is for projection, not filtering — use where(expr) to filter")
		}
		if args == "" {
			return Stage{}, errors.New("select requires field names")
		}
		fields := parseCSV(args)
		return Stage{Kind: StageSelect, Fields: fields}, nil

	case "sort-by":
		if isCall {
			parts := splitCommaArgs(args)
			if len(parts) == 0 || parts[0] == "" {
				return Stage{}, errors.New("sort-by requires a field name")
			}
			desc := false
			if len(parts) >= 2 && strings.TrimSpace(parts[1]) == "desc" {
				desc = true
			}
			return Stage{Kind: StageSort, SortField: strings.TrimSpace(parts[0]), SortDesc: desc}, nil
		}
		return Stage{}, errors.New("sort-by requires parentheses: sort-by(field) or sort-by(field, desc)")

	case "take":
		n, err := strconv.Atoi(strings.TrimSpace(args))
		if err != nil {
			return Stage{}, fmt.Errorf("take requires a number: %w", err)
		}
		return Stage{Kind: StageTake, Limit: n}, nil

	case "last":
		n, err := strconv.Atoi(strings.TrimSpace(args))
		if err != nil {
			return Stage{}, fmt.Errorf("last requires a number: %w", err)
		}
		return Stage{Kind: StageLast, Limit: n}, nil

	case "length":
		return Stage{Kind: StageCount}, nil

	case "unique":
		return Stage{Kind: StageUnique}, nil

	case "group-by":
		if isCall {
			if args == "" {
				return Stage{}, errors.New("group-by requires a field name")
			}
			parts := splitCommaArgs(args)
			if len(parts) == 0 || parts[0] == "" {
				return Stage{}, errors.New("group-by requires a field name")
			}
			fields := []string{strings.TrimSpace(parts[0])}
			if len(parts) >= 2 {
				fields = append(fields, strings.TrimSpace(parts[1]))
			}
			return Stage{Kind: StageGroupBy, Fields: fields}, nil
		}
		return Stage{}, errors.New("group-by requires parentheses: group-by(field)")

	case "refs":
		return parseRefs(isCall, args)

	case "properties":
		if isCall && args != "" {
			limit, err := parseDepthArg(args, "properties")
			if err != nil {
				return Stage{}, err
			}
			return Stage{Kind: StageProperties, Limit: limit}, nil
		}
		return Stage{Kind: StageProperties}, nil

	case "items":
		return Stage{Kind: StageItems}, nil

	case "parent":
		return Stage{Kind: StageOrigin}, nil

	case "to-operations":
		return Stage{Kind: StageToOperations}, nil

	case "to-schemas":
		return Stage{Kind: StageToSchemas}, nil

	case "explain":
		return Stage{Kind: StageExplain}, nil

	case "fields":
		return Stage{Kind: StageFields}, nil

	case "sample":
		n, err := strconv.Atoi(strings.TrimSpace(args))
		if err != nil {
			return Stage{}, fmt.Errorf("sample requires a number: %w", err)
		}
		return Stage{Kind: StageSample, Limit: n}, nil

	case "path":
		if isCall {
			parts := splitCommaArgs(args)
			if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				return Stage{}, errors.New("path requires two schema names")
			}
			return Stage{Kind: StagePath, PathFrom: strings.TrimSpace(parts[0]), PathTo: strings.TrimSpace(parts[1])}, nil
		}
		from, to := parseTwoArgs(args)
		if from == "" || to == "" {
			return Stage{}, errors.New("path requires two schema names")
		}
		return Stage{Kind: StagePath, PathFrom: from, PathTo: to}, nil

	case "highest":
		if isCall {
			parts := splitCommaArgs(args)
			if len(parts) < 2 {
				return Stage{}, errors.New("highest requires a number and a field name")
			}
			n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return Stage{}, fmt.Errorf("highest requires a number: %w", err)
			}
			return Stage{Kind: StageHighest, Limit: n, SortField: strings.TrimSpace(parts[1])}, nil
		}
		parts := strings.Fields(args)
		if len(parts) < 2 {
			return Stage{}, errors.New("highest requires a number and a field name")
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return Stage{}, fmt.Errorf("highest requires a number: %w", err)
		}
		return Stage{Kind: StageHighest, Limit: n, SortField: parts[1]}, nil

	case "lowest":
		if isCall {
			parts := splitCommaArgs(args)
			if len(parts) < 2 {
				return Stage{}, errors.New("lowest requires a number and a field name")
			}
			n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return Stage{}, fmt.Errorf("lowest requires a number: %w", err)
			}
			return Stage{Kind: StageLowest, Limit: n, SortField: strings.TrimSpace(parts[1])}, nil
		}
		parts := strings.Fields(args)
		if len(parts) < 2 {
			return Stage{}, errors.New("lowest requires a number and a field name")
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return Stage{}, fmt.Errorf("lowest requires a number: %w", err)
		}
		return Stage{Kind: StageLowest, Limit: n, SortField: parts[1]}, nil

	case "format":
		f := strings.TrimSpace(args)
		if f != "table" && f != "json" && f != "markdown" && f != "toon" {
			return Stage{}, fmt.Errorf("format must be table, json, markdown, or toon, got %q", f)
		}
		return Stage{Kind: StageFormat, Format: f}, nil

	case "to-yaml":
		return Stage{Kind: StageToYAML}, nil

	case "blast-radius":
		return Stage{Kind: StageBlastRadius}, nil

	case "orphans":
		return Stage{Kind: StageOrphans}, nil

	case "leaves":
		return Stage{Kind: StageLeaves}, nil

	case "cycles":
		return Stage{Kind: StageCycles}, nil

	case "clusters":
		return Stage{Kind: StageClusters}, nil

	case "cross-tag":
		return Stage{Kind: StageCrossTag}, nil

	case "shared-refs":
		if isCall && args != "" {
			n, err := strconv.Atoi(strings.TrimSpace(args))
			if err != nil {
				return Stage{}, fmt.Errorf("shared-refs requires a minimum count: %w", err)
			}
			return Stage{Kind: StageSharedRefs, Limit: n}, nil
		}
		return Stage{Kind: StageSharedRefs}, nil

	case "let":
		return parseLet(args)

	// Navigation stages
	case "parameters":
		return Stage{Kind: StageParameters}, nil

	case "responses":
		return Stage{Kind: StageResponses}, nil

	case "request-body":
		return Stage{Kind: StageRequestBody}, nil

	case "content-types":
		return Stage{Kind: StageContentTypes}, nil

	case "headers":
		return Stage{Kind: StageHeaders}, nil

	case "to-schema":
		return Stage{Kind: StageToSchema}, nil

	case "operation":
		return Stage{Kind: StageOperation}, nil

	case "security":
		return Stage{Kind: StageSecurity}, nil

	case "members":
		return Stage{Kind: StageMembers}, nil

	case "callbacks":
		return Stage{Kind: StageCallbacks}, nil

	case "links":
		return Stage{Kind: StageLinks}, nil

	case "additional-properties":
		return Stage{Kind: StageAdditionalProperties}, nil

	case "pattern-properties":
		return Stage{Kind: StagePatternProperties}, nil

	default:
		return Stage{}, fmt.Errorf("unknown stage: %q", keyword)
	}
}

// parseRefs parses the refs(...) syntax.
// Syntax: refs, refs(*), refs(out), refs(out, *), refs(in, 3), refs(3), etc.
func parseRefs(isCall bool, args string) (Stage, error) {
	if !isCall {
		return Stage{Kind: StageRefs, Limit: 1}, nil
	}

	parts := splitCommaArgs(args)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return Stage{Kind: StageRefs, Limit: 1}, nil
	}

	first := strings.TrimSpace(parts[0])

	// Single arg: direction, depth, or *
	if len(parts) == 1 {
		switch first {
		case "out", "in":
			return Stage{Kind: StageRefs, RefsDir: first, Limit: 1}, nil
		default:
			limit, err := parseDepthArg(first, "refs")
			if err != nil {
				return Stage{}, fmt.Errorf("refs accepts direction (out, in), depth number, or *: %w", err)
			}
			return Stage{Kind: StageRefs, Limit: limit}, nil
		}
	}

	// Two args: direction + depth
	dir := first
	if dir != "out" && dir != "in" {
		return Stage{}, fmt.Errorf("refs first argument must be out or in, got %q", dir)
	}
	limit, err := parseDepthArg(strings.TrimSpace(parts[1]), "refs")
	if err != nil {
		return Stage{}, err
	}
	return Stage{Kind: StageRefs, RefsDir: dir, Limit: limit}, nil
}

// parseDepthArg parses a depth argument: a positive integer or "*" for unbounded.
// "*" is represented internally as Limit = -1.
func parseDepthArg(args, stageName string) (int, error) {
	arg := strings.TrimSpace(args)
	if arg == "*" {
		return -1, nil
	}
	n, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("%s requires a depth number or *: %w", stageName, err)
	}
	return n, nil
}

func parseLet(args string) (Stage, error) {
	// let $var = expr
	if args == "" || !strings.HasPrefix(args, "$") {
		return Stage{}, errors.New("let requires $variable = expression")
	}
	eqIdx := strings.Index(args, "=")
	if eqIdx < 0 {
		return Stage{}, errors.New("let requires $variable = expression")
	}
	varName := strings.TrimSpace(args[:eqIdx])
	exprStr := strings.TrimSpace(args[eqIdx+1:])
	if !strings.HasPrefix(varName, "$") || len(varName) < 2 {
		return Stage{}, errors.New("let variable must start with $")
	}
	if exprStr == "" {
		return Stage{}, errors.New("let requires an expression after =")
	}
	return Stage{Kind: StageLet, VarName: varName, Expr: exprStr}, nil
}

func parseFuncSig(sig string) (FuncDef, error) {
	fd := FuncDef{}
	parenIdx := strings.Index(sig, "(")
	if parenIdx < 0 {
		fd.Name = strings.TrimSpace(sig)
		if fd.Name == "" {
			return fd, errors.New("def requires a name")
		}
		return fd, nil
	}
	fd.Name = strings.TrimSpace(sig[:parenIdx])
	if fd.Name == "" {
		return fd, errors.New("def requires a name")
	}
	closeIdx := strings.LastIndex(sig, ")")
	if closeIdx < 0 {
		return fd, errors.New("def params missing closing )")
	}
	paramStr := sig[parenIdx+1 : closeIdx]
	for _, p := range splitCommaArgs(paramStr) {
		p = strings.TrimSpace(p)
		if p != "" {
			if !strings.HasPrefix(p, "$") {
				return fd, fmt.Errorf("def param %q must start with $", p)
			}
			fd.Params = append(fd.Params, p)
		}
	}
	return fd, nil
}

func findUnquotedSemicolon(s string) int {
	var quoteChar byte
	depth := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quoteChar != 0 {
			if ch == '\\' && i+1 < len(s) {
				i++ // skip escaped character
			} else if ch == quoteChar {
				quoteChar = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quoteChar = ch
		case '(':
			depth++
		case ')':
			depth--
		case ';':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitKeywordCall splits "where(expr)" into ("where", "expr", true).
// Returns ("", "", false) if s doesn't match keyword(...) form.
// The keyword must be a single word (no spaces before the opening paren).
func splitKeywordCall(s string) (string, string, bool) {
	s = strings.TrimSpace(s)
	parenIdx := strings.Index(s, "(")
	if parenIdx < 0 {
		return "", "", false
	}
	keyword := s[:parenIdx]
	// Keyword must not contain spaces (single word only)
	if strings.ContainsAny(keyword, " \t") {
		return "", "", false
	}
	if keyword == "" {
		return "", "", false
	}
	// Find matching closing paren (not just the last one — handle nested parens)
	rest := s[parenIdx+1:]
	depth := 1
	var quoteChar byte
	end := -1
	for i := 0; i < len(rest); i++ {
		ch := rest[i]
		if quoteChar != 0 {
			if ch == '\\' && i+1 < len(rest) {
				i++ // skip escaped character
			} else if ch == quoteChar {
				quoteChar = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quoteChar = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return "", "", false
	}
	// Ensure nothing after the closing paren
	trailing := strings.TrimSpace(rest[end+1:])
	if trailing != "" {
		return "", "", false
	}
	args := rest[:end]
	return keyword, args, true
}

// splitCommaArgs splits stage arguments on commas (respecting nesting and quotes).
func splitCommaArgs(s string) []string {
	return splitAtDelim(s, ',')
}

func parseTwoArgs(s string) (string, string) {
	s = strings.TrimSpace(s)
	var args []string
	for len(s) > 0 {
		if s[0] == '"' {
			// Quoted arg
			end := strings.Index(s[1:], "\"")
			if end < 0 {
				args = append(args, s[1:])
				break
			}
			args = append(args, s[1:end+1])
			s = strings.TrimSpace(s[end+2:])
		} else {
			idx := strings.IndexAny(s, " \t")
			if idx < 0 {
				args = append(args, s)
				break
			}
			args = append(args, s[:idx])
			s = strings.TrimSpace(s[idx+1:])
		}
		if len(args) == 2 {
			break
		}
	}
	if len(args) < 2 {
		if len(args) == 1 {
			return args[0], ""
		}
		return "", ""
	}
	return args[0], args[1]
}

// --- Pipeline splitting ---

func splitPipeline(input string) []string {
	return splitAtDelim(input, '|')
}

// splitAtDelim splits a string at unquoted, depth-0 occurrences of delim,
// respecting single/double quotes and parenthesis nesting.
func splitAtDelim(input string, delim byte) []string {
	var parts []string
	var current strings.Builder
	var quoteChar byte
	depth := 0

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case quoteChar != 0:
			current.WriteByte(ch)
			if ch == '\\' && i+1 < len(input) {
				i++
				current.WriteByte(input[i])
			} else if ch == quoteChar {
				quoteChar = 0
			}
		case ch == '"' || ch == '\'':
			quoteChar = ch
			current.WriteByte(ch)
		case ch == '(':
			depth++
			current.WriteByte(ch)
		case ch == ')':
			depth--
			current.WriteByte(ch)
		case ch == delim && depth == 0:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func splitFirst(s string) (string, string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
