package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticNoScriptTagsInMarkdown = "semantic-no-script-tags-in-markdown"

const noScriptPattern = "<script"

var noScriptRegex = regexp.MustCompile(noScriptPattern)

type NoScriptTagsInMarkdownRule struct{}

func (r *NoScriptTagsInMarkdownRule) ID() string       { return RuleSemanticNoScriptTagsInMarkdown }
func (r *NoScriptTagsInMarkdownRule) Category() string { return CategorySemantic }
func (r *NoScriptTagsInMarkdownRule) Description() string {
	return "Markdown descriptions must not contain <script> tags, which pose serious security risks. Including script tags in documentation could enable cross-site scripting (XSS) attacks if the documentation is rendered in web contexts."
}
func (r *NoScriptTagsInMarkdownRule) Summary() string {
	return "Markdown descriptions must not include <script> tags."
}
func (r *NoScriptTagsInMarkdownRule) HowToFix() string {
	return "Remove <script> tags from markdown descriptions."
}
func (r *NoScriptTagsInMarkdownRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-no-script-tags-in-markdown"
}
func (r *NoScriptTagsInMarkdownRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *NoScriptTagsInMarkdownRule) Versions() []string {
	return nil
}

func (r *NoScriptTagsInMarkdownRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Use the pre-computed DescriptionNodes index
	for _, descNode := range docInfo.Index.DescriptionNodes {
		desc := descNode.Node.GetDescription()
		if desc == "" {
			continue
		}

		if noScriptRegex.MatchString(desc) {
			// Get the precise YAML node for the description field
			errNode := GetFieldValueNode(descNode.Node, "description", docInfo.Document)

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleSemanticNoScriptTagsInMarkdown,
				fmt.Errorf("description contains content with `%s`, forbidden", noScriptPattern),
				errNode,
			))
		}
	}

	return errs
}
