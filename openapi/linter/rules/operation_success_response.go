package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

const RuleStyleOperationSuccessResponse = "style-operation-success-response"

type OperationSuccessResponseRule struct{}

func (r *OperationSuccessResponseRule) ID() string { return RuleStyleOperationSuccessResponse }

func (r *OperationSuccessResponseRule) Category() string { return CategoryStyle }

func (r *OperationSuccessResponseRule) Description() string {
	return "Operations should define at least one 2xx or 3xx response code to indicate successful execution. Success responses are essential for API consumers to understand what data they'll receive when requests complete successfully."
}

func (r *OperationSuccessResponseRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-operation-success-response"
}

func (r *OperationSuccessResponseRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OperationSuccessResponseRule) Versions() []string { return nil }

func (r *OperationSuccessResponseRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	doc := docInfo.Document
	isOAS3 := strings.HasPrefix(doc.GetOpenAPI(), "3.")

	var errs []error

	// Use the pre-computed Operations index
	for _, opNode := range docInfo.Index.Operations {
		op := opNode.Node

		responses := op.GetResponses()
		responseSeen := false
		responseInvalidType := false
		invalidCodes := []int{}

		if responses != nil {
			for code := range responses.All() {
				codeVal, err := strconv.Atoi(code)
				if err == nil && codeVal >= 200 && codeVal < 400 {
					responseSeen = true
				}
			}

			if isOAS3 {
				responseInvalidType, invalidCodes = findIntegerResponseCodes(op)
				if responseInvalidType {
					responseSeen = true
				}
			}
		}

		if !responseSeen || responseInvalidType {
			opName := op.GetOperationID()
			if opName == "" {
				opName = "undefined operation (no operationId)"
			}

			errNode := getOperationResponsesKeyNode(op, doc)

			if !responseSeen {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleStyleOperationSuccessResponse,
					fmt.Errorf("operation `%s` must define at least a single `2xx` or `3xx` response", opName),
					errNode,
				))
			}

			if responseInvalidType {
				for _, code := range invalidCodes {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleStyleOperationSuccessResponse,
						fmt.Errorf("operation `%s` uses an `integer` instead of a `string` for response code `%d`", opName, code),
						errNode,
					))
				}
			}
		}
	}

	return errs
}

func getOperationResponsesKeyNode(op *openapi.Operation, doc *openapi.OpenAPI) *yaml.Node {
	if op == nil {
		if doc != nil {
			return doc.GetCore().GetRootNode()
		}
		return nil
	}

	core := op.GetCore()
	if core != nil && core.Responses.Present && core.Responses.KeyNode != nil {
		return core.Responses.KeyNode
	}

	if core != nil && core.GetRootNode() != nil {
		return core.GetRootNode()
	}

	if doc != nil {
		return doc.GetCore().GetRootNode()
	}

	return nil
}

func findIntegerResponseCodes(op *openapi.Operation) (bool, []int) {
	core := op.GetCore()
	if core == nil || !core.Responses.Present || core.Responses.ValueNode == nil {
		return false, nil
	}

	valueNode := core.Responses.ValueNode
	if valueNode.Kind != yaml.MappingNode {
		return false, nil
	}

	invalidCodes := []int{}
	for i := 0; i < len(valueNode.Content); i += 2 {
		keyNode := valueNode.Content[i]
		if keyNode == nil || keyNode.Kind != yaml.ScalarNode {
			continue
		}

		if keyNode.Tag != "!!int" {
			continue
		}

		codeVal, err := strconv.Atoi(keyNode.Value)
		if err != nil {
			continue
		}
		invalidCodes = append(invalidCodes, codeVal)
	}

	if len(invalidCodes) == 0 {
		return false, nil
	}

	return true, invalidCodes
}
