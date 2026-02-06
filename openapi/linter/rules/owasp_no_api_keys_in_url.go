package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

//nolint:gosec
const RuleOwaspNoAPIKeysInURL = "owasp-no-api-keys-in-url"

type OwaspNoAPIKeysInURLRule struct{}

func (r *OwaspNoAPIKeysInURLRule) ID() string       { return RuleOwaspNoAPIKeysInURL }
func (r *OwaspNoAPIKeysInURLRule) Category() string { return CategorySecurity }
func (r *OwaspNoAPIKeysInURLRule) Description() string {
	return "API keys must not be passed via URL parameters (query or path) as they are logged and cached. URL-based API keys appear in browser history, server logs, and proxy caches, creating security exposure."
}
func (r *OwaspNoAPIKeysInURLRule) Summary() string {
	return "API keys must not be passed via URL parameters."
}
func (r *OwaspNoAPIKeysInURLRule) HowToFix() string {
	return "Move API keys to header-based authentication instead of query or path parameters."
}
func (r *OwaspNoAPIKeysInURLRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-no-api-keys-in-url"
}
func (r *OwaspNoAPIKeysInURLRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspNoAPIKeysInURLRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspNoAPIKeysInURLRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Check if this is an API key type security scheme
		schemeType := secScheme.GetType()
		if schemeType != "apiKey" {
			continue
		}

		// Get the location where the API key is passed
		location := secScheme.GetIn()

		// Check if it's in query or path (both are insecure for API keys)
		if location == "query" || location == "path" {
			// Get the root node to find the "in" key
			if rootNode := secScheme.GetRootNode(); rootNode != nil {
				_, inValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "in")
				if found && inValueNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspNoAPIKeysInURL,
						fmt.Errorf("security scheme `%s` passes API key via URL `%s` parameter - use header instead for security", name, location),
						inValueNode,
					))
				}
			}
		}
	}

	return errs
}
