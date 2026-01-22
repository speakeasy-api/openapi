package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspRateLimit = "owasp-rate-limit"

// Rate limiting headers to check for
var rateLimitHeaders = []string{
	"X-RateLimit-Limit",
	"X-Rate-Limit-Limit",
	"RateLimit-Limit",
	"RateLimit-Reset",
	"RateLimit",
}

type OwaspRateLimitRule struct{}

func (r *OwaspRateLimitRule) ID() string {
	return RuleOwaspRateLimit
}
func (r *OwaspRateLimitRule) Category() string {
	return CategorySecurity
}
func (r *OwaspRateLimitRule) Description() string {
	return "2XX and 4XX responses must define rate limiting headers (X-RateLimit-Limit, X-RateLimit-Remaining) to prevent API overload. Rate limit headers help clients manage their usage and avoid hitting limits."
}
func (r *OwaspRateLimitRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-rate-limit"
}
func (r *OwaspRateLimitRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspRateLimitRule) Versions() []string {
	return []string{"3.0", "3.1"} // OAS3 only
}

func (r *OwaspRateLimitRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Check all response codes
		for statusCode, response := range responses.All() {
			// Only check 2XX and 4XX responses
			if !strings.HasPrefix(statusCode, "2") && !strings.HasPrefix(statusCode, "4") {
				continue
			}

			responseObj := response.GetObject()
			if responseObj == nil {
				continue
			}

			headers := responseObj.GetHeaders()
			if headers == nil || headers.Len() == 0 {
				// No headers defined - report missing rate limit headers
				if rootNode := response.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspRateLimit,
						fmt.Errorf("response %s for operation %s %s is missing rate limiting headers", statusCode, method, path),
						rootNode,
					))
				}
				continue
			}

			// Check if any rate limit header is present
			hasRateLimitHeader := false
			for _, headerName := range rateLimitHeaders {
				if _, exists := headers.Get(headerName); exists {
					hasRateLimitHeader = true
					break
				}
			}

			if !hasRateLimitHeader {
				// No rate limit header found
				if rootNode := responseObj.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspRateLimit,
						fmt.Errorf("response %s for operation %s %s is missing rate limiting headers (expected one of: %s)",
							statusCode, method, path, strings.Join(rateLimitHeaders, ", ")),
						rootNode,
					))
				}
			}
		}
	}

	return errs
}
