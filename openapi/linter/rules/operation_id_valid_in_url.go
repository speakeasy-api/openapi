package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticOperationIDValidInURL = "semantic-operation-id-valid-in-url"

// urlFriendlyPattern matches URL-friendly characters per RFC 3986 (unreserved + reserved characters)
var urlFriendlyPattern = regexp.MustCompile(`^[A-Za-z0-9-._~:/?#\[\]@!$&'()*+,;=]*$`)

type OperationIDValidInURLRule struct{}

func (r *OperationIDValidInURLRule) ID() string {
	return RuleSemanticOperationIDValidInURL
}

func (r *OperationIDValidInURLRule) Description() string {
	return "Operation IDs must use URL-friendly characters (alphanumeric, hyphens, and underscores only). URL-safe operation IDs ensure compatibility with code generators and tooling that may use them in URLs or file paths."
}

func (r *OperationIDValidInURLRule) Category() string {
	return CategorySemantic
}

func (r *OperationIDValidInURLRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}

func (r *OperationIDValidInURLRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-operation-id-valid-in-url"
}

func (r *OperationIDValidInURLRule) Versions() []string {
	return nil // applies to all versions
}

func (r *OperationIDValidInURLRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	doc := docInfo.Document
	var errs []error

	// Use the pre-computed Operations index for efficient iteration
	for _, opNode := range docInfo.Index.Operations {
		operation := opNode.Node
		if operation == nil {
			continue
		}

		operationID := operation.GetOperationID()
		if operationID == "" {
			continue
		}

		if !urlFriendlyPattern.MatchString(operationID) {
			node := GetFieldValueNode(operation, "operationId", doc)
			if node == nil {
				node = operation.GetRootNode()
			}

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleSemanticOperationIDValidInURL,
				fmt.Errorf("operationId `%s` contains characters that are not URL-friendly", operationID),
				node,
			))
		}
	}

	return errs
}
