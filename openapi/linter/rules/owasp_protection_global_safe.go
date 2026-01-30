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

const RuleOwaspProtectionGlobalSafe = "owasp-protection-global-safe"

// Safe HTTP methods that don't modify state
var safeMethods = map[string]bool{
	"get":  true,
	"head": true,
}

type OwaspProtectionGlobalSafeRule struct{}

func (r *OwaspProtectionGlobalSafeRule) ID() string {
	return RuleOwaspProtectionGlobalSafe
}
func (r *OwaspProtectionGlobalSafeRule) Category() string {
	return CategorySecurity
}
func (r *OwaspProtectionGlobalSafeRule) Description() string {
	return "Safe operations (GET, HEAD) should be protected by security schemes or explicitly marked as public. Unprotected read operations may expose sensitive data to unauthorized users."
}
func (r *OwaspProtectionGlobalSafeRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-protection-global-safe"
}
func (r *OwaspProtectionGlobalSafeRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *OwaspProtectionGlobalSafeRule) Versions() []string {
	return []string{"3.0", "3.1"} // OAS3 only
}

func (r *OwaspProtectionGlobalSafeRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	doc := docInfo.Document

	// Check if there's a global security requirement
	globalSecurity := doc.GetSecurity()
	hasGlobalSecurity := len(globalSecurity) > 0

	// Check all operations
	for _, opNode := range docInfo.Index.Operations {
		op := opNode.Node
		if op == nil {
			continue
		}

		// Get operation details
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

		// Only check safe methods
		if !safeMethods[strings.ToLower(method)] {
			continue
		}

		// Check if operation has explicit security field (even if empty array)
		// security: [] means explicitly public and is allowed
		rootNode := op.GetRootNode()
		hasExplicitSecurity := false
		if rootNode != nil {
			_, _, found := yml.GetMapElementNodes(ctx, rootNode, "security")
			hasExplicitSecurity = found
		}

		// Operation is protected if:
		// 1. Has global security, OR
		// 2. Has explicit operation-level security field (even if empty)
		if !hasGlobalSecurity && !hasExplicitSecurity {
			if rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspProtectionGlobalSafe,
					fmt.Errorf("operation %s %s is not protected by any security scheme", method, path),
					rootNode,
				))
			}
		}
	}

	return errs
}
