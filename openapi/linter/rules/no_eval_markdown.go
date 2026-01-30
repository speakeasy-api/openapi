package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticNoEvalInMarkdown = "semantic-no-eval-in-markdown"

const noEvalPattern = "eval\\("

var noEvalRegex = regexp.MustCompile(noEvalPattern)

type NoEvalInMarkdownRule struct{}

func (r *NoEvalInMarkdownRule) ID() string       { return RuleSemanticNoEvalInMarkdown }
func (r *NoEvalInMarkdownRule) Category() string { return CategorySemantic }
func (r *NoEvalInMarkdownRule) Description() string {
	return "Markdown descriptions must not contain eval() statements, which pose serious security risks. Including eval() in documentation could enable code injection attacks if the documentation is rendered in contexts that execute JavaScript."
}
func (r *NoEvalInMarkdownRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-no-eval-in-markdown"
}
func (r *NoEvalInMarkdownRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *NoEvalInMarkdownRule) Versions() []string {
	return nil
}

func (r *NoEvalInMarkdownRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		if noEvalRegex.MatchString(desc) {
			// Get the precise YAML node for the description field
			errNode := GetFieldValueNode(descNode.Node, "description", docInfo.Document)

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleSemanticNoEvalInMarkdown,
				fmt.Errorf("description contains content with `%s`, forbidden", noEvalPattern),
				errNode,
			))
		}
	}

	return errs
}
