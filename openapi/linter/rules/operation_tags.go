package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOperationTags = "style-operation-tags"

type OperationTagsRule struct{}

func (r *OperationTagsRule) ID() string {
	return RuleStyleOperationTags
}

func (r *OperationTagsRule) Description() string {
	return "Operations should have at least one tag to enable logical grouping and organization in documentation. Tags help developers navigate the API by categorizing related operations together."
}

func (r *OperationTagsRule) Summary() string {
	return "Operations should have at least one tag."
}

func (r *OperationTagsRule) HowToFix() string {
	return "Add at least one tag to each operation."
}

func (r *OperationTagsRule) Category() string {
	return CategoryStyle
}

func (r *OperationTagsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OperationTagsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-operation-tags"
}

func (r *OperationTagsRule) Versions() []string {
	return nil // applies to all versions
}

func (r *OperationTagsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	for _, opNode := range docInfo.Index.Operations {
		operation := opNode.Node
		if operation == nil {
			continue
		}

		tags := operation.GetTags()
		if len(tags) == 0 {
			// Get operation identifier (prefer operationId, fallback to method + path)
			opIdentifier := operation.GetOperationID()
			if opIdentifier == "" {
				method, path := openapi.ExtractMethodAndPath(opNode.Location)
				if method != "" {
					opIdentifier = fmt.Sprintf("`%s` %s", strings.ToUpper(method), path)
				}
			}

			rootNode := operation.GetRootNode()
			errs = append(errs, &validation.Error{
				UnderlyingError: fmt.Errorf("the %s is missing tags", opIdentifier),
				Node:            rootNode,
				Severity:        config.GetSeverity(r.DefaultSeverity()),
				Rule:            RuleStyleOperationTags,
				Fix:             &addOperationTagFix{operationNode: rootNode},
			})
		}
	}

	return errs
}
