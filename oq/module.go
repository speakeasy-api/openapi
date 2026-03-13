package oq

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadModule loads function definitions from a .oq module file.
func LoadModule(path string, searchPaths []string) ([]FuncDef, error) {
	resolved, err := resolveModulePath(path, searchPaths)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolved) //nolint:gosec // module paths are user-provided query inputs, not untrusted
	if err != nil {
		return nil, fmt.Errorf("reading module %q: %w", resolved, err)
	}

	q, err := parseDeclarations(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing module %q: %w", resolved, err)
	}

	return q.Defs, nil
}

func resolveModulePath(path string, searchPaths []string) (string, error) {
	if !strings.HasSuffix(path, ".oq") {
		path += ".oq"
	}

	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	allPaths := make([]string, 0, len(searchPaths)+2)
	allPaths = append(allPaths, ".")
	allPaths = append(allPaths, searchPaths...)
	if home, err := os.UserHomeDir(); err == nil {
		allPaths = append(allPaths, filepath.Join(home, ".config", "oq"))
	}

	for _, dir := range allPaths {
		full := filepath.Join(dir, path)
		if _, err := os.Stat(full); err == nil {
			return full, nil
		}
	}

	return "", fmt.Errorf("module %q not found in search paths", path)
}

// ExpandDefs performs text-level macro expansion on pipeline segments.
// Each segment that matches a def name gets replaced with the def's body
// (with params substituted).
func ExpandDefs(pipelineText string, defs []FuncDef) (string, error) {
	if len(defs) == 0 {
		return pipelineText, nil
	}

	defMap := make(map[string]FuncDef, len(defs))
	for _, d := range defs {
		defMap[d.Name] = d
	}

	parts := splitPipeline(pipelineText)
	var expanded []string

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i == 0 {
			// Source — don't expand
			expanded = append(expanded, part)
			continue
		}

		keyword, args, isCall := splitKeywordCall(part)
		if !isCall {
			keyword, _ = splitFirst(part)
		}

		def, ok := defMap[strings.ToLower(keyword)]
		if !ok {
			expanded = append(expanded, part)
			continue
		}

		body := def.Body
		if isCall && len(def.Params) > 0 {
			callArgs := splitSemicolonArgs(args)
			if len(callArgs) != len(def.Params) {
				return "", fmt.Errorf("def %q expects %d params, got %d", def.Name, len(def.Params), len(callArgs))
			}
			for j, param := range def.Params {
				body = strings.ReplaceAll(body, param, strings.TrimSpace(callArgs[j]))
			}
		} else if !isCall && len(def.Params) > 0 {
			return "", fmt.Errorf("def %q requires %d params", def.Name, len(def.Params))
		}

		expanded = append(expanded, body)
	}

	return strings.Join(expanded, " | "), nil
}
