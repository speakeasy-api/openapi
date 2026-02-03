package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspStringLimit = "owasp-string-limit"

type OwaspStringLimitRule struct{}

func (r *OwaspStringLimitRule) ID() string {
	return RuleOwaspStringLimit
}
func (r *OwaspStringLimitRule) Category() string {
	return CategorySecurity
}
func (r *OwaspStringLimitRule) Description() string {
	return "String schemas must specify maxLength, const, or enum to prevent unbounded data. Without string length limits, APIs are vulnerable to resource exhaustion from extremely long inputs."
}
func (r *OwaspStringLimitRule) Summary() string {
	return "String schemas must specify maxLength, const, or enum."
}
func (r *OwaspStringLimitRule) HowToFix() string {
	return "Add maxLength, const, or enum constraints to string schemas."
}
func (r *OwaspStringLimitRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-string-limit"
}
func (r *OwaspStringLimitRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspStringLimitRule) Versions() []string {
	return []string{"3.0", "3.1"}
}

func (r *OwaspStringLimitRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Check all schemas in the document
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		// Check if type contains "string"
		types := schema.GetType()
		hasStringType := false
		for _, typ := range types {
			if typ == "string" {
				hasStringType = true
				break
			}
		}

		if !hasStringType {
			continue
		}

		// Check if schema has maxLength, const, or enum defined
		maxLength := schema.GetMaxLength()
		constValue := schema.GetConst()
		enumValues := schema.GetEnum()

		// If none of these are defined, report error
		if maxLength == nil && constValue == nil && len(enumValues) == 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspStringLimit,
					errors.New("schema of type 'string' must specify maxLength, const, or enum to prevent unbounded data"),
					rootNode,
				))
			}
		}
	}

	return errs
}
