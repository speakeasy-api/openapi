package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspDefineErrorValidation = "owasp-define-error-validation"

type OwaspDefineErrorValidationRule struct{}

func (r *OwaspDefineErrorValidationRule) ID() string { return RuleOwaspDefineErrorValidation }
func (r *OwaspDefineErrorValidationRule) Category() string {
	return CategorySecurity
}
func (r *OwaspDefineErrorValidationRule) Description() string {
	return "Operations should define validation error responses (400, 422, or 4XX) to indicate request data problems. Validation error responses help clients understand when and why their request data is invalid or malformed."
}
func (r *OwaspDefineErrorValidationRule) Summary() string {
	return "Operations should define validation error responses (400, 422, or 4XX)."
}
func (r *OwaspDefineErrorValidationRule) HowToFix() string {
	return "Add a 400, 422, or 4XX response with a schema to each operation to describe validation errors."
}
func (r *OwaspDefineErrorValidationRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-define-error-validation"
}
func (r *OwaspDefineErrorValidationRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OwaspDefineErrorValidationRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspDefineErrorValidationRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Check all operations
	for _, opNode := range docInfo.Index.Operations {
		op := opNode.Node
		if op == nil {
			continue
		}

		// Get operation details for error messages
		method := ""
		path := ""
		for _, loc := range opNode.Location {
			switch openapi.GetParentType(loc) {
			case "Paths":
				if loc.ParentKey != nil {
					path = *loc.ParentKey
				}
			case "PathItem":
				if loc.ParentKey != nil {
					method = *loc.ParentKey
				}
			}
		}

		responses := op.GetResponses()
		if responses == nil {
			// No responses at all - report missing validation error response
			if rootNode := op.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspDefineErrorValidation,
					fmt.Errorf("operation %s %s is missing validation error response (400, 422, or 4XX)", method, path),
					rootNode,
				))
			}
			continue
		}

		// Check if any of the validation error codes exist
		has400, _ := responses.Get("400")
		has422, _ := responses.Get("422")
		has4XX, _ := responses.Get("4XX")

		if has400 == nil && has422 == nil && has4XX == nil {
			// Missing all validation error responses
			if rootNode := responses.GetRootNode(); rootNode != nil {
				errs = append(errs, &validation.Error{
					UnderlyingError: fmt.Errorf("operation %s %s is missing validation error response (should have 400, 422, or 4XX)", method, path),
					Node:            rootNode,
					Severity:        config.GetSeverity(r.DefaultSeverity()),
					Rule:            RuleOwaspDefineErrorValidation,
					Fix:             &addErrorResponseFix{responsesNode: rootNode, statusCode: "400", description: "Bad Request"},
				})
			}
		}
	}

	return errs
}
