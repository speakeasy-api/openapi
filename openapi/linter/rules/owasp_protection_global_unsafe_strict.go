package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspProtectionGlobalUnsafeStrict = "owasp-protection-global-unsafe-strict"

type OwaspProtectionGlobalUnsafeStrictRule struct{}

func (r *OwaspProtectionGlobalUnsafeStrictRule) ID() string {
	return RuleOwaspProtectionGlobalUnsafeStrict
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) Category() string {
	return CategorySecurity
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) Description() string {
	return "Unsafe operations (POST, PUT, PATCH, DELETE) must be protected by non-empty security schemes without explicit opt-outs. Strict authentication requirements ensure write operations cannot bypass security even with empty security arrays."
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) Summary() string {
	return "Unsafe operations must have non-empty security schemes (no opt-outs)."
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) HowToFix() string {
	return "Define non-empty security requirements globally or per unsafe operation (no empty security arrays)."
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-protection-global-unsafe-strict"
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *OwaspProtectionGlobalUnsafeStrictRule) Versions() []string {
	return nil
}

func (r *OwaspProtectionGlobalUnsafeStrictRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	doc := docInfo.Document

	// Check if there's a global security requirement with actual schemes
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

		// Check if operation has security with actual schemes
		opSecurity := op.GetSecurity()
		hasOpSecurity := len(opSecurity) > 0

		// Strict mode: operation must have actual security schemes (no empty arrays allowed)
		if !hasGlobalSecurity && !hasOpSecurity {
			if rootNode := op.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspProtectionGlobalUnsafeStrict,
					fmt.Errorf("operation %s %s must be protected by a security scheme (empty security array not allowed)", method, path),
					rootNode,
				))
			}
		}
	}

	return errs
}
