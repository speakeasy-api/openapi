package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

const RuleOwaspDefineErrorResponses401 = "owasp-define-error-responses-401"

type OwaspDefineErrorResponses401Rule struct{}

func (r *OwaspDefineErrorResponses401Rule) ID() string { return RuleOwaspDefineErrorResponses401 }
func (r *OwaspDefineErrorResponses401Rule) Category() string {
	return CategorySecurity
}
func (r *OwaspDefineErrorResponses401Rule) Description() string {
	return "Operations should define a 401 Unauthorized response with a proper schema to handle authentication failures. Documenting authentication error responses helps clients implement proper error handling and understand when credentials are invalid or missing."
}
func (r *OwaspDefineErrorResponses401Rule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-define-error-responses-401"
}
func (r *OwaspDefineErrorResponses401Rule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OwaspDefineErrorResponses401Rule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspDefineErrorResponses401Rule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
			// No responses at all - report missing 401
			if rootNode := op.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspDefineErrorResponses401,
					fmt.Errorf("operation %s %s is missing 401 Unauthorized error response", method, path),
					rootNode,
				))
			}
			continue
		}

		// Check if 401 response exists
		response401, has401 := responses.Get("401")
		if !has401 {
			// Missing 401 response
			if rootNode := responses.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspDefineErrorResponses401,
					fmt.Errorf("operation %s %s is missing 401 Unauthorized error response", method, path),
					rootNode,
				))
			}
			continue
		}

		// 401 exists, check if it has content with schema
		if response401 != nil {
			responseObj := response401.GetObject()
			if responseObj != nil {
				content := responseObj.GetContent()
				if content == nil || content.Len() == 0 {
					// 401 exists but has no content/schema
					if rootNode := responseObj.GetRootNode(); rootNode != nil {
						_, responseValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "description")
						if !found {
							responseValueNode = rootNode
						}
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleOwaspDefineErrorResponses401,
							fmt.Errorf("operation %s %s has 401 response but missing content schema", method, path),
							responseValueNode,
						))
					}
				}
			}
		}
	}

	return errs
}
