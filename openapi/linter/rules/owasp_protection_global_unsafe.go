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

const RuleOwaspProtectionGlobalUnsafe = "owasp-protection-global-unsafe"

// Unsafe HTTP methods that modify state and should be protected
var unsafeMethods = map[string]bool{
	"post":   true,
	"put":    true,
	"patch":  true,
	"delete": true,
}

type OwaspProtectionGlobalUnsafeRule struct{}

func (r *OwaspProtectionGlobalUnsafeRule) ID() string { return RuleOwaspProtectionGlobalUnsafe }
func (r *OwaspProtectionGlobalUnsafeRule) Category() string {
	return CategorySecurity
}
func (r *OwaspProtectionGlobalUnsafeRule) Description() string {
	return "Unsafe operations (POST, PUT, PATCH, DELETE) must be protected by security schemes to prevent unauthorized modifications. Write operations without authentication create serious security vulnerabilities allowing data tampering."
}
func (r *OwaspProtectionGlobalUnsafeRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-protection-global-unsafe"
}
func (r *OwaspProtectionGlobalUnsafeRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspProtectionGlobalUnsafeRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *OwaspProtectionGlobalUnsafeRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Only check unsafe methods
		if !unsafeMethods[strings.ToLower(method)] {
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
					RuleOwaspProtectionGlobalUnsafe,
					fmt.Errorf("operation %s %s is not protected by any security scheme", method, path),
					rootNode,
				))
			}
		}
	}

	return errs
}
