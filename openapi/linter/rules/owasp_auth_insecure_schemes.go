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

const RuleOwaspAuthInsecureSchemes = "owasp-auth-insecure-schemes"

type OwaspAuthInsecureSchemesRule struct{}

func (r *OwaspAuthInsecureSchemesRule) ID() string       { return RuleOwaspAuthInsecureSchemes }
func (r *OwaspAuthInsecureSchemesRule) Category() string { return CategorySecurity }
func (r *OwaspAuthInsecureSchemesRule) Description() string {
	return "Authentication schemes using outdated or insecure methods must be avoided or upgraded. Insecure authentication schemes like API keys in query parameters or HTTP Basic over HTTP expose credentials and create security vulnerabilities."
}
func (r *OwaspAuthInsecureSchemesRule) Summary() string {
	return "Security schemes must not use outdated or insecure HTTP schemes."
}
func (r *OwaspAuthInsecureSchemesRule) HowToFix() string {
	return "Replace insecure HTTP schemes (negotiate/oauth) with modern authentication like OAuth 2.0 or bearer tokens."
}
func (r *OwaspAuthInsecureSchemesRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-auth-insecure-schemes"
}
func (r *OwaspAuthInsecureSchemesRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspAuthInsecureSchemesRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspAuthInsecureSchemesRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	doc := docInfo.Document
	components := doc.GetComponents()
	if components == nil {
		return nil
	}

	securitySchemes := components.GetSecuritySchemes()
	if securitySchemes == nil {
		return nil
	}

	var errs []error

	// Iterate through all security schemes
	for name, scheme := range securitySchemes.All() {
		if scheme == nil {
			continue
		}

		// Get the security scheme object
		secScheme := scheme.GetObject()
		if secScheme == nil {
			continue
		}

		// Check if this is an HTTP type security scheme
		schemeType := secScheme.GetType()
		if schemeType != "http" {
			continue
		}

		// Get the scheme value (basic, bearer, negotiate, oauth, etc.)
		httpScheme := secScheme.GetScheme()
		httpSchemeLower := strings.ToLower(httpScheme)

		// Check if it's negotiate or oauth (both insecure/outdated)
		if httpSchemeLower == "negotiate" || httpSchemeLower == "oauth" {
			// Get the root node to find the scheme key
			if rootNode := secScheme.GetRootNode(); rootNode != nil {
				_, schemeValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "scheme")
				if found && schemeValueNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspAuthInsecureSchemes,
						fmt.Errorf("security scheme '%s' uses '%s' which is outdated or insecure - use modern authentication like OAuth 2.0 or bearer tokens", name, httpScheme),
						schemeValueNode,
					))
				}
			}
		}
	}

	return errs
}
