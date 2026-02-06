package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspAdditionalPropertiesConstrained = "owasp-additional-properties-constrained"

type OwaspAdditionalPropertiesConstrainedRule struct{}

func (r *OwaspAdditionalPropertiesConstrainedRule) ID() string {
	return RuleOwaspAdditionalPropertiesConstrained
}
func (r *OwaspAdditionalPropertiesConstrainedRule) Category() string {
	return CategorySecurity
}
func (r *OwaspAdditionalPropertiesConstrainedRule) Description() string {
	return "Schemas with additionalProperties set to true or a schema should define maxProperties to limit object size. Without size limits, APIs are vulnerable to resource exhaustion attacks where clients send excessively large objects."
}
func (r *OwaspAdditionalPropertiesConstrainedRule) Summary() string {
	return "Schemas with additionalProperties should define maxProperties."
}
func (r *OwaspAdditionalPropertiesConstrainedRule) HowToFix() string {
	return "When additionalProperties is true or a schema, add a maxProperties limit to bound object size."
}
func (r *OwaspAdditionalPropertiesConstrainedRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-additional-properties-constrained"
}
func (r *OwaspAdditionalPropertiesConstrainedRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *OwaspAdditionalPropertiesConstrainedRule) Versions() []string {
	return []string{"3.0", "3.1"}
}

func (r *OwaspAdditionalPropertiesConstrainedRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Check if type contains "object"
		types := schema.GetType()
		hasObjectType := false
		for _, typ := range types {
			if typ == "object" {
				hasObjectType = true
				break
			}
		}

		if !hasObjectType {
			continue
		}

		// Check additionalProperties
		additionalProps := schema.GetAdditionalProperties()
		if additionalProps == nil {
			// Not set - no constraint needed
			continue
		}

		// Check if additionalProperties allows additional properties
		// (either as a schema object or as true)
		allowsAdditional := false

		if additionalProps.IsBool() {
			// It's a boolean value
			boolVal := additionalProps.GetBool()
			if boolVal != nil && *boolVal {
				// additionalProperties: true
				allowsAdditional = true
			}
		} else {
			// It's a schema object - allows additional properties
			allowsAdditional = true
		}

		// If additional properties are allowed, maxProperties should be defined
		if allowsAdditional {
			maxProps := schema.GetMaxProperties()
			if maxProps == nil {
				if rootNode := refSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspAdditionalPropertiesConstrained,
						errors.New("schema should define maxProperties when additionalProperties is set to true or a schema"),
						rootNode,
					))
				}
			}
		}
	}

	return errs
}
