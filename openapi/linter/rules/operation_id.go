package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticOperationOperationId = "semantic-operation-operation-id"

type OperationIdRule struct{}

func (r *OperationIdRule) ID() string { return RuleSemanticOperationOperationId }

func (r *OperationIdRule) Category() string { return CategorySemantic }

func (r *OperationIdRule) Description() string {
	return "Operations should define an operationId for consistent referencing across the specification and in generated code. Operation IDs enable tooling to generate meaningful function names and provide stable identifiers for API operations."
}

func (r *OperationIdRule) Summary() string {
	return "Operations should define an operationId."
}

func (r *OperationIdRule) HowToFix() string {
	return "Add an operationId to each operation."
}

func (r *OperationIdRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-operation-operation-id"
}

func (r *OperationIdRule) DefaultSeverity() validation.Severity { return validation.SeverityWarning }

func (r *OperationIdRule) Versions() []string { return nil }

func (r *OperationIdRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Use the pre-computed Operations index for efficient iteration
	for _, opNode := range docInfo.Index.Operations {
		op := opNode.Node
		method, path := openapi.ExtractMethodAndPath(opNode.Location)
		if method == "" || path == "" {
			continue
		}

		if op.GetOperationID() != "" {
			continue
		}

		errNode := op.GetRootNode()
		if errNode == nil {
			errNode = docInfo.Document.GetRootNode()
		}

		err := validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleSemanticOperationOperationId,
			fmt.Errorf("the `%s` operation does not contain an `operationId`", strings.ToUpper(method)),
			errNode,
		)
		errs = append(errs, err)
	}

	return errs
}
