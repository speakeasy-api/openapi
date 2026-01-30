package rules

import (
	"context"
	"errors"
	"strconv"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOperationErrorResponse = "style-operation-error-response"

type OperationErrorResponseRule struct{}

func (r *OperationErrorResponseRule) ID() string       { return RuleStyleOperationErrorResponse }
func (r *OperationErrorResponseRule) Category() string { return CategoryStyle }
func (r *OperationErrorResponseRule) Description() string {
	return "Operations should define at least one 4xx error response to document potential client errors. Documenting error responses helps API consumers handle failures gracefully and understand what went wrong when requests fail."
}
func (r *OperationErrorResponseRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-operation-error-response"
}
func (r *OperationErrorResponseRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OperationErrorResponseRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OperationErrorResponseRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Use the pre-computed operation index
	for _, opNode := range docInfo.Index.Operations {
		operation := opNode.Node
		if operation == nil {
			continue
		}

		responses := operation.GetResponses()
		if responses == nil {
			continue
		}

		// Check if any response code is in the 4xx range
		has4xxResponse := false
		if responses.Map != nil {
			for code := range responses.All() {
				// Try to parse the code as an integer
				codeVal, err := strconv.Atoi(code)
				if err == nil && codeVal >= 400 && codeVal < 500 {
					has4xxResponse = true
					break
				}
			}
		}

		if !has4xxResponse {
			// Get the responses node for error reporting
			responsesNode := responses.GetRootNode()
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStyleOperationErrorResponse,
				errors.New("operation must define at least one 4xx error response"),
				responsesNode,
			))
		}
	}

	return errs
}
