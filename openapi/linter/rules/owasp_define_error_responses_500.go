package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

const RuleOwaspDefineErrorResponses500 = "owasp-define-error-responses-500"

type OwaspDefineErrorResponses500Rule struct{}

func (r *OwaspDefineErrorResponses500Rule) ID() string { return RuleOwaspDefineErrorResponses500 }
func (r *OwaspDefineErrorResponses500Rule) Category() string {
	return CategorySecurity
}
func (r *OwaspDefineErrorResponses500Rule) Description() string {
	return "Operations should define a 500 Internal Server Error response with a proper schema to handle unexpected failures. Documenting server error responses helps clients distinguish between client-side and server-side problems."
}
func (r *OwaspDefineErrorResponses500Rule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-define-error-responses-500"
}
func (r *OwaspDefineErrorResponses500Rule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OwaspDefineErrorResponses500Rule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspDefineErrorResponses500Rule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
			// No responses at all - report missing 500
			if rootNode := op.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspDefineErrorResponses500,
					fmt.Errorf("operation %s %s is missing 500 Internal Server Error response", method, path),
					rootNode,
				))
			}
			continue
		}

		// Check if 500 response exists
		response500, has500 := responses.Get("500")
		if !has500 {
			// Missing 500 response
			if rootNode := responses.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspDefineErrorResponses500,
					fmt.Errorf("operation %s %s is missing 500 Internal Server Error response", method, path),
					rootNode,
				))
			}
			continue
		}

		// 500 exists, check if it has content with schema
		if response500 != nil {
			responseObj := response500.GetObject()
			if responseObj != nil {
				content := responseObj.GetContent()
				if content == nil || content.Len() == 0 {
					// 500 exists but has no content/schema
					if rootNode := responseObj.GetRootNode(); rootNode != nil {
						_, responseValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "description")
						if !found {
							responseValueNode = rootNode
						}
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleOwaspDefineErrorResponses500,
							fmt.Errorf("operation %s %s has 500 response but missing content schema", method, path),
							responseValueNode,
						))
					}
				}
			}
		}
	}

	return errs
}
