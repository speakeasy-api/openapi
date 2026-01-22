package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspStringRestricted = "owasp-string-restricted"

type OwaspStringRestrictedRule struct{}

func (r *OwaspStringRestrictedRule) ID() string {
	return RuleOwaspStringRestricted
}
func (r *OwaspStringRestrictedRule) Category() string {
	return CategorySecurity
}
func (r *OwaspStringRestrictedRule) Description() string {
	return "String schemas must specify format, const, enum, or pattern to restrict content. String restrictions prevent injection attacks and ensure data conforms to expected formats."
}
func (r *OwaspStringRestrictedRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-string-restricted"
}
func (r *OwaspStringRestrictedRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspStringRestrictedRule) Versions() []string {
	return []string{"3.0", "3.1"}
}

func (r *OwaspStringRestrictedRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Check if schema has format, const, enum, or pattern defined
		format := schema.GetFormat()
		constValue := schema.GetConst()
		enumValues := schema.GetEnum()
		pattern := schema.GetPattern()

		// If none of these are defined, report error
		if format == "" && constValue == nil && len(enumValues) == 0 && pattern == "" {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspStringRestricted,
					errors.New("schema of type 'string' must specify format, const, enum, or pattern to restrict content"),
					rootNode,
				))
			}
		}
	}

	return errs
}
