package oq

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Parse splits a pipeline query string into stages.
func Parse(query string) ([]Stage, error) {
	// Split by pipe, respecting quoted strings
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
			// First part is a source
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
	// Extract the keyword
	keyword, rest := splitFirst(s)
	keyword = strings.ToLower(keyword)

	switch keyword {
	case "where":
		if rest == "" {
			return Stage{}, errors.New("where requires an expression")
		}
		return Stage{Kind: StageWhere, Expr: rest}, nil

	case "select":
		if rest == "" {
			return Stage{}, errors.New("select requires field names")
		}
		fields := parseCSV(rest)
		return Stage{Kind: StageSelect, Fields: fields}, nil

	case "sort":
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			return Stage{}, errors.New("sort requires a field name")
		}
		desc := false
		if len(parts) >= 2 && strings.ToLower(parts[1]) == "desc" {
			desc = true
		}
		return Stage{Kind: StageSort, SortField: parts[0], SortDesc: desc}, nil

	case "take", "head":
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return Stage{}, fmt.Errorf("take requires a number: %w", err)
		}
		return Stage{Kind: StageTake, Limit: n}, nil

	case "unique":
		return Stage{Kind: StageUnique}, nil

	case "group-by":
		if rest == "" {
			return Stage{}, errors.New("group-by requires a field name")
		}
		fields := parseCSV(rest)
		return Stage{Kind: StageGroupBy, Fields: fields}, nil

	case "count":
		return Stage{Kind: StageCount}, nil

	case "refs-out":
		return Stage{Kind: StageRefsOut}, nil

	case "refs-in":
		return Stage{Kind: StageRefsIn}, nil

	case "reachable":
		return Stage{Kind: StageReachable}, nil

	case "ancestors":
		return Stage{Kind: StageAncestors}, nil

	case "properties":
		return Stage{Kind: StageProperties}, nil

	case "union-members":
		return Stage{Kind: StageUnionMembers}, nil

	case "items":
		return Stage{Kind: StageItems}, nil

	case "ops":
		return Stage{Kind: StageOps}, nil

	case "schemas":
		return Stage{Kind: StageSchemas}, nil

	case "explain":
		return Stage{Kind: StageExplain}, nil

	case "fields":
		return Stage{Kind: StageFields}, nil

	case "sample":
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return Stage{}, fmt.Errorf("sample requires a number: %w", err)
		}
		return Stage{Kind: StageSample, Limit: n}, nil

	case "path":
		from, to := parseTwoArgs(rest)
		if from == "" || to == "" {
			return Stage{}, errors.New("path requires two schema names")
		}
		return Stage{Kind: StagePath, PathFrom: from, PathTo: to}, nil

	case "top":
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			return Stage{}, errors.New("top requires a number and a field name")
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return Stage{}, fmt.Errorf("top requires a number: %w", err)
		}
		return Stage{Kind: StageTop, Limit: n, SortField: parts[1]}, nil

	case "bottom":
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			return Stage{}, errors.New("bottom requires a number and a field name")
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return Stage{}, fmt.Errorf("bottom requires a number: %w", err)
		}
		return Stage{Kind: StageBottom, Limit: n, SortField: parts[1]}, nil

	case "format":
		f := strings.TrimSpace(rest)
		if f != "table" && f != "json" && f != "markdown" && f != "toon" {
			return Stage{}, fmt.Errorf("format must be table, json, markdown, or toon, got %q", f)
		}
		return Stage{Kind: StageFormat, Format: f}, nil

	case "connected":
		return Stage{Kind: StageConnected}, nil

	case "blast-radius":
		return Stage{Kind: StageBlastRadius}, nil

	case "neighbors":
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return Stage{}, fmt.Errorf("neighbors requires a depth number: %w", err)
		}
		return Stage{Kind: StageNeighbors, Limit: n}, nil

	case "orphans":
		return Stage{Kind: StageOrphans}, nil

	case "leaves":
		return Stage{Kind: StageLeaves}, nil

	case "cycles":
		return Stage{Kind: StageCycles}, nil

	case "clusters":
		return Stage{Kind: StageClusters}, nil

	case "tag-boundary":
		return Stage{Kind: StageTagBoundary}, nil

	case "shared-refs":
		return Stage{Kind: StageSharedRefs}, nil

	default:
		return Stage{}, fmt.Errorf("unknown stage: %q", keyword)
	}
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
	var parts []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '"':
			inQuote = !inQuote
			current.WriteByte(ch)
		case ch == '|' && !inQuote:
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
