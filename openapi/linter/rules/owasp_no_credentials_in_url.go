package rules

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

//nolint:gosec
const RuleOwaspNoCredentialsInURL = "owasp-no-credentials-in-url"

// credentialWords flag a parameter when one of them is the head (last) word of its
// name: clientSecret, access_token and userPassword are credentials, while secretPath,
// tokenId and secretName merely describe one.
var credentialWords = []string{"secret", "token", "password", "passwd", "pwd"}

// keyQualifiers turn a trailing "key" into a credential (api-key, secretKey); "key" on
// its own is a lookup key, not a credential (keyName, sortKey).
var keyQualifiers = []string{"api", "secret"}

// looksLikeCredential reports whether a parameter name reads as a credential rather
// than as something that refers to one.
func looksLikeCredential(name string) bool {
	words := splitParamName(name)
	if len(words) == 0 {
		return false
	}
	head := words[len(words)-1]

	// Suffix rather than equality so glued names (accesstoken, clientsecret) still match.
	for _, w := range credentialWords {
		if strings.HasSuffix(head, w) {
			return true
		}
	}
	for _, q := range keyQualifiers {
		if head == q+"key" || (head == "key" && len(words) > 1 && words[len(words)-2] == q) {
			return true
		}
	}
	return false
}

// splitParamName breaks a parameter name into lowercase words on `_`, `-`, `.`, digits
// and camelCase boundaries. Acronym runs split before their last capital, so APIKey
// becomes [api key] and userID becomes [user id].
func splitParamName(name string) []string {
	var (
		words []string
		cur   []rune
	)
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}

	runes := []rune(name)
	for i, r := range runes {
		if !unicode.IsLetter(r) {
			flush()
			continue
		}
		if unicode.IsUpper(r) && len(cur) > 0 {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || nextIsLower {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}

type OwaspNoCredentialsInURLRule struct{}

func (r *OwaspNoCredentialsInURLRule) ID() string       { return RuleOwaspNoCredentialsInURL }
func (r *OwaspNoCredentialsInURLRule) Category() string { return CategorySecurity }
func (r *OwaspNoCredentialsInURLRule) Description() string {
	return "URL parameters must not contain credentials like API keys, passwords, or secrets. Credentials in URLs are logged by servers, proxies, and browsers, creating significant security risks."
}
func (r *OwaspNoCredentialsInURLRule) Summary() string {
	return "URL parameters must not contain credentials."
}
func (r *OwaspNoCredentialsInURLRule) HowToFix() string {
	return "Remove credentials from URL parameters; use headers or request bodies instead."
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

		// Check if the parameter name reads as a credential
		paramName := paramObj.GetName()
		if looksLikeCredential(paramName) {
			// Get the root node to find the name key
			if rootNode := paramObj.GetRootNode(); rootNode != nil {
				_, nameValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "name")
				if found && nameValueNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspNoCredentialsInURL,
						fmt.Errorf("URL parameter `%s` appears to contain credentials - avoid passing sensitive data in URLs", paramName),
						nameValueNode,
					))
				}
			}
		}
	}

	// Check all parameters (inline, component, external, and references)
	for _, paramNode := range docInfo.Index.GetAllParameters() {
		checkParameter(paramNode)
	}

	return errs
}
