package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

const RuleOwaspNoCredentialsInURL = "owasp-no-credentials-in-url"

// credentialPattern matches parameter names that look like credentials
// Matches: client_secret, clientsecret, token, access_token, accesstoken, refresh_token, refreshtoken,
// id_token, idtoken, password, secret, api-key, apikey (case insensitive)
var credentialPattern = regexp.MustCompile(`(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`)

type OwaspNoCredentialsInURLRule struct{}

func (r *OwaspNoCredentialsInURLRule) ID() string       { return RuleOwaspNoCredentialsInURL }
func (r *OwaspNoCredentialsInURLRule) Category() string { return CategorySecurity }
func (r *OwaspNoCredentialsInURLRule) Description() string {
	return "URL parameters must not contain credentials like API keys, passwords, or secrets. Credentials in URLs are logged by servers, proxies, and browsers, creating significant security risks."
}
func (r *OwaspNoCredentialsInURLRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-no-credentials-in-url"
}
func (r *OwaspNoCredentialsInURLRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspNoCredentialsInURLRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspNoCredentialsInURLRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Helper function to check a parameter
	checkParameter := func(paramNode *openapi.IndexNode[*openapi.ReferencedParameter]) {
		param := paramNode.Node
		if param == nil {
			return
		}

		// Get the parameter object
		paramObj := param.GetObject()
		if paramObj == nil {
			return
		}

		// Only check query and path parameters (header and cookie are OK)
		location := paramObj.GetIn()
		if location != "query" && location != "path" {
			return
		}

		// Check if the parameter name matches the credential pattern
		paramName := paramObj.GetName()
		if credentialPattern.MatchString(paramName) {
			// Get the root node to find the name key
			if rootNode := paramObj.GetRootNode(); rootNode != nil {
				_, nameValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "name")
				if found && nameValueNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspNoCredentialsInURL,
						fmt.Errorf("URL parameter '%s' appears to contain credentials - avoid passing sensitive data in URLs", paramName),
						nameValueNode,
					))
				}
			}
		}
	}

	// Check both inline and component parameters
	for _, paramNode := range docInfo.Index.InlineParameters {
		checkParameter(paramNode)
	}
	for _, paramNode := range docInfo.Index.ComponentParameters {
		checkParameter(paramNode)
	}

	return errs
}
