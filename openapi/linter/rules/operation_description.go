package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOperationDescription = "style-operation-description"

type OperationDescriptionRule struct{}

func (r *OperationDescriptionRule) ID() string {
	return RuleStyleOperationDescription
}

func (r *OperationDescriptionRule) Description() string {
	return "Operations should include either a description or summary field to explain their purpose and behavior. Clear operation documentation helps developers understand what each endpoint does and how to use it effectively."
}

func (r *OperationDescriptionRule) Summary() string {
	return "Operations must include a description or summary."
}

func (r *OperationDescriptionRule) HowToFix() string {
	return "Add a summary or description to each operation."
}

func (r *OperationDescriptionRule) Category() string {
	return CategoryStyle
}

func (r *OperationDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OperationDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-operation-description"
}

func (r *OperationDescriptionRule) Versions() []string {
	return nil // applies to all versions
}

func (r *OperationDescriptionRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	for _, opNode := range docInfo.Index.Operations {
		operation := opNode.Node
		if operation == nil {
			continue
		}

		description := operation.GetDescription()
		summary := operation.GetSummary()

		if description == "" && summary == "" {
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
				UnderlyingError: fmt.Errorf("the %s is missing a description or summary", opIdentifier),
				Node:            rootNode,
				Severity:        config.GetSeverity(r.DefaultSeverity()),
				Rule:            RuleStyleOperationDescription,
				Fix:             &addDescriptionFix{targetNode: rootNode, targetLabel: "operation " + opIdentifier},
			})
		}
	}

	return errs
}
