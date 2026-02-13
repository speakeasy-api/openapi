package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticLinkOperation = "semantic-link-operation"

type LinkOperationRule struct{}

func (r *LinkOperationRule) ID() string {
	return RuleSemanticLinkOperation
}

func (r *LinkOperationRule) Category() string {
	return CategorySemantic
}

func (r *LinkOperationRule) Description() string {
	return "Link operationId must reference an existing operation in the API specification. This ensures that links point to valid operations, including those defined in external documents that are referenced in the specification."
}

func (r *LinkOperationRule) Summary() string {
	return "Link operationIds must reference existing operations."
}

func (r *LinkOperationRule) HowToFix() string {
	return "Update link.operationId values to reference defined operations."
}

func (r *LinkOperationRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-link-operation"
}

func (r *LinkOperationRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}

func (r *LinkOperationRule) Versions() []string {
	return nil
}

func (r *LinkOperationRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	doc := docInfo.Document
	var errs []error

	// Build a map of all operation IDs for efficient lookup
	operationIDs := make(map[string]bool)
	for _, opNode := range docInfo.Index.Operations {
		if opNode.Node != nil {
			opID := opNode.Node.GetOperationID()
			if opID != "" {
				operationIDs[opID] = true
			}
		}
	}

	// Check all links (inline, component, and external)
	allLinks := docInfo.Index.GetAllLinks()

	for _, linkNode := range allLinks {
		if linkNode.Node == nil || linkNode.Node.Object == nil {
			continue
		}

		link := linkNode.Node.Object
		operationID := link.GetOperationID()
		operationRef := link.GetOperationRef()

		// Validate operationId if present
		if operationID != "" {
			if !operationIDs[operationID] {
				node := GetFieldValueNode(link, "operationId", doc)
				if node == nil {
					node = link.GetRootNode()
				}
				if node == nil {
					node = doc.GetRootNode()
				}

				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					validation.RuleValidationOperationNotFound,
					fmt.Errorf("link.operationId value `%s` does not exist in document", operationID),
					node,
				))
			}
		}

		// TODO: Validate operationRef
		// For now, operationRef URI format validation remains in Link.Validate()
		// Full operationRef resolution validation requires tracking which document each operation belongs to
		// and resolving relative URIs correctly. This will be implemented in a future enhancement.
		_ = operationRef
	}

	return errs
}
