package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleNoVerbsInPath = "style-no-verbs-in-path"

type NoVerbsInPathRule struct{}

func (r *NoVerbsInPathRule) ID() string {
	return RuleStyleNoVerbsInPath
}

func (r *NoVerbsInPathRule) Description() string {
	return "Path segments should not contain HTTP verbs like GET, POST, PUT, DELETE, or QUERY since the HTTP method already conveys the action. RESTful API design favors resource-oriented paths (e.g., `/users`) over action-oriented paths (e.g., `/getUsers`)."
}

func (r *NoVerbsInPathRule) Category() string {
	return CategoryStyle
}

func (r *NoVerbsInPathRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *NoVerbsInPathRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-no-verbs-in-path"
}

func (r *NoVerbsInPathRule) Versions() []string {
	return nil // applies to all versions
}

// HTTP verbs that should not appear in path segments
var httpVerbs = map[string]bool{
	"get":     true,
	"post":    true,
	"put":     true,
	"patch":   true,
	"delete":  true,
	"head":    true,
	"options": true,
	"trace":   true,
	"connect": true,
	"query":   true, // new in OpenAPI 3.2
}

// checkPathForVerbs returns the first segment containing an HTTP verb, or empty string
func checkPathForVerbs(path string) string {
	segments := strings.Split(path, "/")[1:] // skip first empty segment
	for _, seg := range segments {
		segLower := strings.ToLower(seg)
		// Check if the segment is exactly a verb
		if httpVerbs[segLower] {
			return seg
		}
		// Check if segment contains verb as a word (with delimiters)
		for verb := range httpVerbs {
			// Check for verb with delimiters like hyphens, underscores
			if strings.Contains(segLower, verb+"-") ||
				strings.Contains(segLower, verb+"_") ||
				strings.Contains(segLower, "-"+verb) ||
				strings.Contains(segLower, "_"+verb) {
				return seg
			}
		}
	}
	return ""
}

func (r *NoVerbsInPathRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	doc := docInfo.Document

	paths := doc.GetPaths()
	if paths == nil {
		return nil
	}

	for pathKey := range paths.All() {
		if verb := checkPathForVerbs(pathKey); verb != "" {
			node := paths.GetCore().GetMapKeyNodeOrRoot(pathKey, paths.GetRootNode())
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStyleNoVerbsInPath,
				fmt.Errorf("path `%s` must not contain HTTP verb `%s`", pathKey, verb),
				node,
			))
		}
	}

	return errs
}
