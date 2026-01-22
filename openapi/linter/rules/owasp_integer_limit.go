package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspIntegerLimit = "owasp-integer-limit"

type OwaspIntegerLimitRule struct{}

func (r *OwaspIntegerLimitRule) ID() string {
	return RuleOwaspIntegerLimit
}
func (r *OwaspIntegerLimitRule) Category() string {
	return CategorySecurity
}
func (r *OwaspIntegerLimitRule) Description() string {
	return "Integer schemas must specify minimum and maximum values (or exclusive variants) to prevent unbounded inputs. Without numeric limits, APIs are vulnerable to overflow attacks and unexpected behavior from extreme values."
}
func (r *OwaspIntegerLimitRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-integer-limit"
}
func (r *OwaspIntegerLimitRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspIntegerLimitRule) Versions() []string {
	return []string{"3.0", "3.1"}
}

func (r *OwaspIntegerLimitRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Check if type contains "integer"
		types := schema.GetType()
		hasIntegerType := false
		for _, typ := range types {
			if typ == "integer" {
				hasIntegerType = true
				break
			}
		}

		if !hasIntegerType {
			continue
		}

		// Check if schema has appropriate minimum and maximum constraints
		minimum := schema.GetMinimum()
		maximum := schema.GetMaximum()
		exclusiveMinimum := schema.GetExclusiveMinimum()
		exclusiveMaximum := schema.GetExclusiveMaximum()

		// Valid combinations:
		// 1. minimum AND maximum
		// 2. minimum AND exclusiveMaximum
		// 3. exclusiveMinimum AND maximum
		// 4. exclusiveMinimum AND exclusiveMaximum

		hasMin := minimum != nil || exclusiveMinimum != nil
		hasMax := maximum != nil || exclusiveMaximum != nil

		if !hasMin || !hasMax {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspIntegerLimit,
					fmt.Errorf("schema of type 'integer' must specify minimum and maximum (or exclusiveMinimum and exclusiveMaximum)"),
					rootNode,
				))
			}
		}
	}

	return errs
}
