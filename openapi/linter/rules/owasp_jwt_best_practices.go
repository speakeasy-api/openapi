package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

const RuleOwaspJWTBestPractices = "owasp-jwt-best-practices"

type OwaspJWTBestPracticesRule struct{}

func (r *OwaspJWTBestPracticesRule) ID() string {
	return RuleOwaspJWTBestPractices
}
func (r *OwaspJWTBestPracticesRule) Category() string {
	return CategorySecurity
}
func (r *OwaspJWTBestPracticesRule) Description() string {
	return "Security schemes using OAuth2 or JWT must explicitly declare support for RFC8725 (JWT Best Current Practices) in the description. RFC8725 compliance ensures JWTs are validated properly and protected against common attacks like algorithm confusion."
}
func (r *OwaspJWTBestPracticesRule) Summary() string {
	return "OAuth2/JWT schemes must mention RFC8725 in their description."
}
func (r *OwaspJWTBestPracticesRule) HowToFix() string {
	return "Update OAuth2/JWT security scheme descriptions to mention RFC8725 compliance."
}
func (r *OwaspJWTBestPracticesRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-jwt-best-practices"
}
func (r *OwaspJWTBestPracticesRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspJWTBestPracticesRule) Versions() []string {
	return []string{"3.0", "3.1"} // OAS3 only
}

func (r *OwaspJWTBestPracticesRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error

	doc := docInfo.Document
	components := doc.GetComponents()
	if components == nil {
		return nil
	}

	securitySchemes := components.GetSecuritySchemes()
	if securitySchemes == nil || securitySchemes.Len() == 0 {
		return nil
	}

	// Check each security scheme
	for name, scheme := range securitySchemes.All() {
		schemeObj := scheme.GetObject()
		if schemeObj == nil {
			continue
		}

		schemeType := schemeObj.GetType()
		bearerFormat := schemeObj.GetBearerFormat()

		// Check if this is OAuth2 or JWT bearer
		isOAuth2 := schemeType == "oauth2"
		isJWT := strings.ToLower(bearerFormat) == "jwt"

		if !isOAuth2 && !isJWT {
			continue
		}

		// Check if description contains RFC8725
		description := schemeObj.GetDescription()
		if !strings.Contains(description, "RFC8725") {
			// Try to get the description node for better error location
			rootNode := scheme.GetRootNode()
			if rootNode != nil {
				_, descNode, found := yml.GetMapElementNodes(ctx, rootNode, "description")
				if found && descNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspJWTBestPractices,
						fmt.Errorf("security scheme `%s` must explicitly declare support for RFC8725 in the description", name),
						descNode,
					))
				} else {
					// No description field - report on the scheme itself
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspJWTBestPractices,
						fmt.Errorf("security scheme `%s` must explicitly declare support for RFC8725 in the description", name),
						rootNode,
					))
				}
			}
		}
	}

	return errs
}
