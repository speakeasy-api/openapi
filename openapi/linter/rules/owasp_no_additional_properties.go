package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspNoAdditionalProperties = "owasp-no-additional-properties"

type OwaspNoAdditionalPropertiesRule struct{}

func (r *OwaspNoAdditionalPropertiesRule) ID() string {
	return RuleOwaspNoAdditionalProperties
}
func (r *OwaspNoAdditionalPropertiesRule) Category() string {
	return CategorySecurity
}
func (r *OwaspNoAdditionalPropertiesRule) Description() string {
	return "Object schemas must not allow arbitrary additional properties (set additionalProperties to false or omit it). Allowing unexpected properties can lead to mass assignment vulnerabilities where attackers inject unintended fields."
}
func (r *OwaspNoAdditionalPropertiesRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-no-additional-properties"
}
func (r *OwaspNoAdditionalPropertiesRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspNoAdditionalPropertiesRule) Versions() []string {
	return []string{"3.0", "3.1"}
}

func (r *OwaspNoAdditionalPropertiesRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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
			// Not set - this is OK
			continue
		}

		// additionalProperties can be either a boolean or a schema
		// If it's a boolean, check if it's true (violation)
		// If it's a schema, that's also a violation
		isViolation := false

		if additionalProps.IsBool() {
			// It's a boolean value
			boolVal := additionalProps.GetBool()
			if boolVal != nil && *boolVal {
				// additionalProperties: true is a violation
				isViolation = true
			}
		} else {
			// It's a schema object - this is a violation
			isViolation = true
		}

		if isViolation {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspNoAdditionalProperties,
					fmt.Errorf("additionalProperties should not be set to true or define a schema - set to false or omit it"),
					rootNode,
				))
			}
		}
	}

	return errs
}
