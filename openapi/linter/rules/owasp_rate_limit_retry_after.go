package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspRateLimitRetryAfter = "owasp-rate-limit-retry-after"

type OwaspRateLimitRetryAfterRule struct{}

func (r *OwaspRateLimitRetryAfterRule) ID() string {
	return RuleOwaspRateLimitRetryAfter
}
func (r *OwaspRateLimitRetryAfterRule) Category() string {
	return CategorySecurity
}
func (r *OwaspRateLimitRetryAfterRule) Description() string {
	return "429 Too Many Requests responses must include a Retry-After header indicating when clients can retry. Retry-After headers prevent thundering herd problems by telling clients exactly when to resume requests."
}
func (r *OwaspRateLimitRetryAfterRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-rate-limit-retry-after"
}
func (r *OwaspRateLimitRetryAfterRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspRateLimitRetryAfterRule) Versions() []string {
	return []string{"3.0", "3.1"} // OAS3 only
}

func (r *OwaspRateLimitRetryAfterRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
			continue
		}

		// Check for 429 response
		response429, exists := responses.Get("429")
		if !exists || response429 == nil {
			continue
		}

		responseObj := response429.GetObject()
		if responseObj == nil {
			continue
		}

		// Check if Retry-After header exists
		headers := responseObj.GetHeaders()
		if headers == nil {
			// No headers at all
			if rootNode := response429.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspRateLimitRetryAfter,
					fmt.Errorf("429 response for operation %s %s is missing Retry-After header", method, path),
					rootNode,
				))
			}
			continue
		}

		// Check for Retry-After header (case-insensitive check)
		retryAfter, hasRetryAfter := headers.Get("Retry-After")
		if !hasRetryAfter || retryAfter == nil {
			// Try alternate casing
			retryAfter, hasRetryAfter = headers.Get("retry-after")
		}

		if !hasRetryAfter || retryAfter == nil {
			if rootNode := responseObj.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspRateLimitRetryAfter,
					fmt.Errorf("429 response for operation %s %s is missing Retry-After header", method, path),
					rootNode,
				))
			}
		}
	}

	return errs
}
