package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspArrayLimit = "owasp-array-limit"

type OwaspArrayLimitRule struct{}

func (r *OwaspArrayLimitRule) ID() string {
	return RuleOwaspArrayLimit
}
func (r *OwaspArrayLimitRule) Category() string {
	return CategorySecurity
}
func (r *OwaspArrayLimitRule) Description() string {
	return "Array schemas must specify maxItems to prevent resource exhaustion attacks. Without array size limits, malicious clients could send extremely large arrays that consume excessive memory or processing time."
}
func (r *OwaspArrayLimitRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-array-limit"
}
func (r *OwaspArrayLimitRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspArrayLimitRule) Versions() []string {
	return []string{"3.0", "3.1"} // OAS3 only
}

func (r *OwaspArrayLimitRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Check all schemas
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		// Check if type contains "array"
		types := schema.GetType()
		hasArrayType := false
		for _, typ := range types {
			if typ == "array" {
				hasArrayType = true
				break
			}
		}

		if !hasArrayType {
			continue
		}

		// Check if maxItems is defined
		maxItems := schema.GetMaxItems()
		if maxItems == nil {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspArrayLimit,
					fmt.Errorf("schema of type array must specify maxItems"),
					rootNode,
				))
			}
		}
	}

	return errs
}
