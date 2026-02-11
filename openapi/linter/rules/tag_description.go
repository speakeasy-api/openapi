package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleTagDescription = "style-tag-description"

type TagDescriptionRule struct{}

func (r *TagDescriptionRule) ID() string {
	return RuleStyleTagDescription
}

func (r *TagDescriptionRule) Description() string {
	return "Tags should include descriptions that explain the purpose and scope of the operations they group. Tag descriptions provide context in documentation and help developers understand the organization of API functionality."
}

func (r *TagDescriptionRule) Summary() string {
	return "Tags should include descriptions."
}

func (r *TagDescriptionRule) HowToFix() string {
	return "Add descriptions to each tag to explain the grouped operations."
}

func (r *TagDescriptionRule) Category() string {
	return CategoryStyle
}

func (r *TagDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}

func (r *TagDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-tag-description"
}

func (r *TagDescriptionRule) Versions() []string {
	return nil // applies to all versions
}

func (r *TagDescriptionRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	doc := docInfo.Document

	tags := doc.GetTags()
	if len(tags) == 0 {
		return nil
	}

	for _, tag := range tags {
		if tag == nil {
			continue
		}

		description := tag.GetDescription()
		name := tag.GetName()

		if description == "" {
			rootNode := tag.GetRootNode()
			errs = append(errs, &validation.Error{
				UnderlyingError: fmt.Errorf("tag `%s` must have a description", name),
				Node:            rootNode,
				Severity:        config.GetSeverity(r.DefaultSeverity()),
				Rule:            RuleStyleTagDescription,
				Fix:             &addDescriptionFix{targetNode: rootNode, targetLabel: "tag '" + name + "'"},
			})
		}
	}

	return errs
}
