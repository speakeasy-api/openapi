package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOpenAPITags = "style-openapi-tags"

type OpenAPITagsRule struct{}

func (r *OpenAPITagsRule) ID() string {
	return RuleStyleOpenAPITags
}

func (r *OpenAPITagsRule) Description() string {
	return "The OpenAPI specification should define a non-empty tags array at the root level to organize and categorize API operations. Tags help structure API documentation and enable logical grouping of related endpoints."
}

func (r *OpenAPITagsRule) Category() string {
	return CategoryStyle
}

func (r *OpenAPITagsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OpenAPITagsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-openapi-tags"
}

func (r *OpenAPITagsRule) Versions() []string {
	return nil // applies to all versions
}

func (r *OpenAPITagsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	doc := docInfo.Document
	tags := doc.GetTags()

	if len(tags) == 0 {
		return []error{validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleOpenAPITags,
			errors.New("OpenAPI object must have a non-empty tags array"),
			doc.GetCore().GetRootNode(),
		)}
	}

	return nil
}
