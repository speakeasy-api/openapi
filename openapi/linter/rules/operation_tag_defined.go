package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOperationTagDefined = "style-operation-tag-defined"

type OperationTagDefinedRule struct{}

func (r *OperationTagDefinedRule) ID() string       { return RuleStyleOperationTagDefined }
func (r *OperationTagDefinedRule) Category() string { return CategoryStyle }
func (r *OperationTagDefinedRule) Description() string {
	return "Operation tags should be declared in the global tags array at the specification root. Pre-defining tags ensures consistency, enables tag-level documentation, and helps maintain a well-organized API structure."
}
func (r *OperationTagDefinedRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-operation-tag-defined"
}
func (r *OperationTagDefinedRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OperationTagDefinedRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OperationTagDefinedRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Build map of global tags
	globalTags := make(map[string]bool)
	for _, tagNode := range docInfo.Index.Tags {
		tag := tagNode.Node
		if tag.Name != "" {
			globalTags[tag.Name] = true
		}
	}

	// Use index to iterate through all operations
	for _, opNode := range docInfo.Index.Operations {
		operation := opNode.Node

		// Get operation identifier (prefer operationId, fallback to method + path)
		opIdentifier := operation.GetOperationID()
		if opIdentifier == "" {
			method, path := openapi.ExtractMethodAndPath(opNode.Location)
			if method != "" {
				opIdentifier = fmt.Sprintf("`%s` %s", strings.ToUpper(method), path)
			}
		}
		if opIdentifier == "" {
			continue
		}

		// Check each tag in the operation
		opTags := operation.GetTags()
		for i, tagName := range opTags {
			if tagName != "" && !globalTags[tagName] {
				errs = append(errs, validation.NewSliceError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleStyleOperationTagDefined,
					fmt.Errorf("tag `%s` for %s operation is not defined as a global tag", tagName, opIdentifier),
					operation.GetCore(),
					operation.GetCore().Tags,
					i,
				))
			}
		}
	}

	return errs
}
