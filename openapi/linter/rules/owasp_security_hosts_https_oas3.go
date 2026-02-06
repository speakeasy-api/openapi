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

const RuleOwaspSecurityHostsHttpsOAS3 = "owasp-security-hosts-https-oas3"

type OwaspSecurityHostsHttpsOAS3Rule struct{}

func (r *OwaspSecurityHostsHttpsOAS3Rule) ID() string {
	return RuleOwaspSecurityHostsHttpsOAS3
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) Category() string {
	return CategorySecurity
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) Description() string {
	return "Server URLs must begin with https:// as the only permitted protocol. Using HTTPS is essential for protecting API traffic from interception, tampering, and eavesdropping attacks."
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) Summary() string {
	return "Server URLs must use HTTPS."
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) HowToFix() string {
	return "Update server URLs to use https:// instead of http://."
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-security-hosts-https-oas3"
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspSecurityHostsHttpsOAS3Rule) Versions() []string {
	return []string{"3.0", "3.1"} // Only applies to OpenAPI 3.x
}

func (r *OwaspSecurityHostsHttpsOAS3Rule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error

	doc := docInfo.Document
	servers := doc.GetServers()
	if len(servers) == 0 {
		return nil
	}

	// Check each server URL
	for _, server := range servers {
		if server == nil {
			continue
		}

		url := server.GetURL()
		if url == "" {
			continue
		}

		// Check if URL starts with https
		if !strings.HasPrefix(url, "https") {
			// Get the root node to find the url key
			if rootNode := server.GetRootNode(); rootNode != nil {
				_, urlValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "url")
				if found && urlValueNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspSecurityHostsHttpsOAS3,
						fmt.Errorf("server URL `%s` must use HTTPS protocol for security", url),
						urlValueNode,
					))
				}
			}
		}
	}

	return errs
}
