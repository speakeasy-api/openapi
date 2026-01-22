package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStylePathTrailingSlash = "style-path-trailing-slash"

type PathTrailingSlashRule struct{}

func (r *PathTrailingSlashRule) ID() string       { return RuleStylePathTrailingSlash }
func (r *PathTrailingSlashRule) Category() string { return CategoryStyle }
func (r *PathTrailingSlashRule) Description() string {
	return "Path definitions should not end with a trailing slash to maintain consistency and avoid routing ambiguity. Trailing slashes in paths can cause mismatches with server routing rules and create duplicate endpoint definitions."
}
func (r *PathTrailingSlashRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-path-trailing-slash"
}
func (r *PathTrailingSlashRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *PathTrailingSlashRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *PathTrailingSlashRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
		if strings.HasSuffix(pathKey, "/") && pathKey != "/" {
			node := paths.GetCore().GetMapKeyNodeOrRoot(pathKey, paths.GetRootNode())
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStylePathTrailingSlash,
				fmt.Errorf("path `%s` must not end with a trailing slash", pathKey),
				node,
			))
		}
	}

	return errs
}
