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

const RuleOwaspNoHttpBasic = "owasp-no-http-basic"

type OwaspNoHttpBasicRule struct{}

func (r *OwaspNoHttpBasicRule) ID() string       { return RuleOwaspNoHttpBasic }
func (r *OwaspNoHttpBasicRule) Category() string { return CategorySecurity }
func (r *OwaspNoHttpBasicRule) Description() string {
	return "Security schemes must not use `HTTP Basic` authentication without additional security layers. `HTTP Basic` sends credentials in easily-decoded base64 encoding, making it vulnerable to interception without `HTTPS`."
}
func (r *OwaspNoHttpBasicRule) Summary() string {
	return "Security schemes must not use `HTTP Basic` authentication."
}
func (r *OwaspNoHttpBasicRule) HowToFix() string {
	return "Replace `HTTP Basic` schemes with more secure authentication (e.g., `OAuth 2.0` or `bearer` tokens)."
}
func (r *OwaspNoHttpBasicRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-no-http-basic"
}
func (r *OwaspNoHttpBasicRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspNoHttpBasicRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspNoHttpBasicRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Get the scheme value (basic, bearer, etc.)
		httpScheme := secScheme.GetScheme()
		httpSchemeLower := strings.ToLower(httpScheme)

		// Check if it's basic or negotiate (both insecure)
		if httpSchemeLower == "basic" || httpSchemeLower == "negotiate" {
			// Get the root node to find the scheme key
			if rootNode := secScheme.GetRootNode(); rootNode != nil {
				_, schemeValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "scheme")
				if found && schemeValueNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspNoHttpBasic,
						fmt.Errorf("security scheme `%s` uses `HTTP` `%s` authentication, which is insecure - use `OAuth 2.0` or another secure method", name, httpScheme),
						schemeValueNode,
					))
				}
			}
		}
	}

	return errs
}
