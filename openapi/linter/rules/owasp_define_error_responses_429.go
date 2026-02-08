package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

const RuleOwaspDefineErrorResponses429 = "owasp-define-error-responses-429"

type OwaspDefineErrorResponses429Rule struct{}

func (r *OwaspDefineErrorResponses429Rule) ID() string { return RuleOwaspDefineErrorResponses429 }
func (r *OwaspDefineErrorResponses429Rule) Category() string {
	return CategorySecurity
}
func (r *OwaspDefineErrorResponses429Rule) Description() string {
	return "Operations should define a 429 Too Many Requests response with a proper schema to indicate rate limiting. Rate limit responses help clients understand when they've exceeded usage thresholds and need to slow down requests."
}
func (r *OwaspDefineErrorResponses429Rule) Summary() string {
	return "Operations should define a 429 Too Many Requests response with a schema."
}
func (r *OwaspDefineErrorResponses429Rule) HowToFix() string {
	return "Add a 429 response with a response body schema to operations that may be rate limited."
}
func (r *OwaspDefineErrorResponses429Rule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-define-error-responses-429"
}
func (r *OwaspDefineErrorResponses429Rule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OwaspDefineErrorResponses429Rule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspDefineErrorResponses429Rule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
			// No responses at all - report missing 429
			if rootNode := op.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspDefineErrorResponses429,
					fmt.Errorf("operation %s %s is missing 429 Too Many Requests response", method, path),
					rootNode,
				))
			}
			continue
		}

		// Check if 429 response exists
		response429, has429 := responses.Get("429")
		if !has429 {
			// Missing 429 response
			if rootNode := responses.GetRootNode(); rootNode != nil {
				errs = append(errs, &validation.Error{
					UnderlyingError: fmt.Errorf("operation %s %s is missing 429 Too Many Requests response", method, path),
					Node:            rootNode,
					Severity:        config.GetSeverity(r.DefaultSeverity()),
					Rule:            RuleOwaspDefineErrorResponses429,
					Fix:             &addErrorResponseFix{responsesNode: rootNode, statusCode: "429", description: "Too Many Requests"},
				})
			}
			continue
		}

		// 429 exists, check if it has content with schema
		if response429 != nil {
			responseObj := response429.GetObject()
			if responseObj != nil {
				content := responseObj.GetContent()
				if content == nil || content.Len() == 0 {
					// 429 exists but has no content/schema
					if rootNode := responseObj.GetRootNode(); rootNode != nil {
						_, responseValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "description")
						if !found {
							responseValueNode = rootNode
						}
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleOwaspDefineErrorResponses429,
							fmt.Errorf("operation %s %s has 429 response but missing content schema", method, path),
							responseValueNode,
						))
					}
				}
			}
		}
	}

	return errs
}
