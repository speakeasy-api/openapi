package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticNoAmbiguousPaths = "semantic-no-ambiguous-paths"

type NoAmbiguousPathsRule struct{}

func (r *NoAmbiguousPathsRule) ID() string       { return RuleSemanticNoAmbiguousPaths }
func (r *NoAmbiguousPathsRule) Category() string { return CategorySemantic }
func (r *NoAmbiguousPathsRule) Description() string {
	return "Path definitions must be unambiguous and distinguishable from each other to ensure correct request routing. Ambiguous paths like `/users/{id}` and `/users/{name}` can cause runtime routing conflicts since both match the same URL pattern."
}
func (r *NoAmbiguousPathsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-no-ambiguous-paths"
}
func (r *NoAmbiguousPathsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *NoAmbiguousPathsRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

var pathVariableRegex = regexp.MustCompile(`\{[^}]+\}`)

func (r *NoAmbiguousPathsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	doc := docInfo.Document
	paths := doc.GetPaths()
	if paths == nil || paths.Len() == 0 {
		return nil
	}

	var errs []error

	// Track normalized paths with their original path
	type normalizedEntry struct {
		originalPath string
		pathItem     *openapi.PathItem
	}

	seenPaths := make(map[string][]normalizedEntry)

	// Iterate through all paths
	for pathStr, refPathItem := range paths.All() {
		if refPathItem == nil {
			continue
		}

		pathItem := refPathItem.GetObject()
		if pathItem == nil {
			continue
		}

		// Normalize the path by replacing all {param} with a placeholder
		normalizedPath := pathVariableRegex.ReplaceAllString(pathStr, "{}")

		// Check if we've seen this normalized path before
		if entries, exists := seenPaths[normalizedPath]; exists {
			// Found an ambiguous path - report it
			for _, entry := range entries {
				pathItemNode := pathItem.GetRootNode()
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleSemanticNoAmbiguousPaths,
					fmt.Errorf("paths are ambiguous with one another: `%s` and `%s`",
						entry.originalPath, pathStr),
					pathItemNode,
				))
			}
		}

		// Add this path to our tracking
		seenPaths[normalizedPath] = append(seenPaths[normalizedPath], normalizedEntry{
			originalPath: pathStr,
			pathItem:     pathItem,
		})
	}

	return errs
}
