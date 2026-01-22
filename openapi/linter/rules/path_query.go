package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticPathQuery = "semantic-path-query"

type PathQueryRule struct{}

func (r *PathQueryRule) ID() string       { return RuleSemanticPathQuery }
func (r *PathQueryRule) Category() string { return CategorySemantic }
func (r *PathQueryRule) Description() string {
	return "Paths must not include query strings - query parameters should be defined in the parameters array instead. Including query strings in paths creates ambiguity, breaks code generation, and violates OpenAPI specification structure."
}
func (r *PathQueryRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-path-query"
}
func (r *PathQueryRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *PathQueryRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *PathQueryRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	doc := docInfo.Document

	// Check each path for query strings
	for pathKey := range doc.GetPaths().All() {
		if strings.Contains(pathKey, "?") {
			node := doc.GetPaths().GetCore().GetMapKeyNodeOrRoot(pathKey, doc.GetPaths().GetRootNode())

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleSemanticPathQuery,
				fmt.Errorf("path %q contains query string - use parameters array instead", pathKey),
				node,
			))
		}
	}

	return errs
}
