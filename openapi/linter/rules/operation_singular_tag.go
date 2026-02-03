package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOperationSingularTag = "style-operation-singular-tag"

type OperationSingularTagRule struct{}

func (r *OperationSingularTagRule) ID() string       { return RuleStyleOperationSingularTag }
func (r *OperationSingularTagRule) Category() string { return CategoryStyle }
func (r *OperationSingularTagRule) Description() string {
	return "Operations should be associated with only a single tag to maintain clear organizational boundaries. Multiple tags can create ambiguity about where an operation belongs in the API structure and complicate documentation organization."
}
func (r *OperationSingularTagRule) Summary() string {
	return "Operations should have no more than one tag."
}
func (r *OperationSingularTagRule) HowToFix() string {
	return "Limit each operation to a single tag."
}
func (r *OperationSingularTagRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-operation-singular-tag"
}
func (r *OperationSingularTagRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OperationSingularTagRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OperationSingularTagRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Use index to iterate through all operations
	for _, opNode := range docInfo.Index.Operations {
		operation := opNode.Node

		// Check if operation has more than one tag
		opTags := operation.GetTags()
		if len(opTags) <= 1 {
			continue
		}

		// Get operation identifier (prefer operationId, fallback to method + path)
		opIdentifier := operation.GetOperationID()
		if opIdentifier == "" {
			method, path := openapi.ExtractMethodAndPath(opNode.Location)
			if method != "" {
				opIdentifier = fmt.Sprintf("`%s` operation at path `%s`", strings.ToUpper(method), path)
			}
		} else {
			opIdentifier = fmt.Sprintf("`%s` operation", opIdentifier)
		}
		if opIdentifier == "" {
			continue
		}

		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleOperationSingularTag,
			fmt.Errorf("the %s contains more than one tag (%d is too many)", opIdentifier, len(opTags)),
			operation.GetCore().Tags.ValueNode,
		))
	}

	return errs
}
