package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticPathDeclarations = "semantic-path-declarations"

type PathDeclarationsRule struct{}

func (r *PathDeclarationsRule) ID() string       { return RuleSemanticPathDeclarations }
func (r *PathDeclarationsRule) Category() string { return CategorySemantic }
func (r *PathDeclarationsRule) Description() string {
	return "Path parameter declarations must not be empty - declarations like /api/{} are invalid. Empty path parameters create ambiguous routes and will cause runtime errors in most API frameworks."
}
func (r *PathDeclarationsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-path-declarations"
}
func (r *PathDeclarationsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *PathDeclarationsRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *PathDeclarationsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	doc := docInfo.Document

	// Check each path for empty parameter declarations
	for pathKey := range doc.GetPaths().All() {
		if strings.Contains(pathKey, "{}") {
			node := doc.GetPaths().GetCore().GetMapKeyNodeOrRoot(pathKey, doc.GetPaths().GetRootNode())

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleSemanticPathDeclarations,
				fmt.Errorf("path %q contains empty parameter declaration `{}`", pathKey),
				node,
			))
		}
	}

	return errs
}
