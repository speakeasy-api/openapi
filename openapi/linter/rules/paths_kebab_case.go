package rules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStylePathsKebabCase = "style-paths-kebab-case"

type PathsKebabCaseRule struct{}

func (r *PathsKebabCaseRule) ID() string {
	return RuleStylePathsKebabCase
}

func (r *PathsKebabCaseRule) Description() string {
	return "Path segments should use kebab-case (lowercase with hyphens) for consistency and readability. Kebab-case paths are easier to read, follow REST conventions, and avoid case-sensitivity issues across different systems."
}

func (r *PathsKebabCaseRule) Summary() string {
	return "Path segments should use kebab-case."
}

func (r *PathsKebabCaseRule) HowToFix() string {
	return "Rename non-kebab-case path segments to lowercase with hyphens."
}

func (r *PathsKebabCaseRule) Category() string {
	return CategoryStyle
}

func (r *PathsKebabCaseRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *PathsKebabCaseRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-paths-kebab-case"
}

func (r *PathsKebabCaseRule) Versions() []string {
	return nil // applies to all versions
}

var pathKebabCaseRegex = regexp.MustCompile(`^[{}a-z\d-.]+$`)
var variableRegex = regexp.MustCompile(`^\{(\w.*)}\.?.*$`)

// checkPathKebabCase returns non-kebab-case segments in the path
func checkPathKebabCase(path string) []string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil
	}
	segments := parts[1:] // skip first empty segment
	var invalidSegments []string

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		// Skip variable segments like {id} or {userId}
		if variableRegex.MatchString(seg) {
			continue
		}
		// Check if segment matches kebab-case pattern
		if !pathKebabCaseRegex.MatchString(seg) {
			invalidSegments = append(invalidSegments, seg)
		}
	}

	return invalidSegments
}

func (r *PathsKebabCaseRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
		invalidSegments := checkPathKebabCase(pathKey)

		if len(invalidSegments) > 0 {
			node := paths.GetCore().GetMapKeyNodeOrRoot(pathKey, paths.GetRootNode())
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStylePathsKebabCase,
				fmt.Errorf("path segments `%s` are not kebab-case", strings.Join(invalidSegments, "`, `")),
				node,
			))
		}
	}

	return errs
}
