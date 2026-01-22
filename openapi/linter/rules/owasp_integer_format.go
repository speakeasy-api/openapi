package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspIntegerFormat = "owasp-integer-format"

type OwaspIntegerFormatRule struct{}

func (r *OwaspIntegerFormatRule) ID() string {
	return RuleOwaspIntegerFormat
}
func (r *OwaspIntegerFormatRule) Category() string {
	return CategorySecurity
}
func (r *OwaspIntegerFormatRule) Description() string {
	return "Integer schemas must specify a format of int32 or int64 to define the expected size and range. Explicit integer formats prevent overflow vulnerabilities and ensure clients and servers agree on numeric boundaries."
}
func (r *OwaspIntegerFormatRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-integer-format"
}
func (r *OwaspIntegerFormatRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspIntegerFormatRule) Versions() []string {
	return []string{"3.0", "3.1"}
}

func (r *OwaspIntegerFormatRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		// Check if format is int32 or int64
		format := schema.GetFormat()
		if format != "int32" && format != "int64" {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleOwaspIntegerFormat,
					fmt.Errorf("schema of type 'integer' must specify format as 'int32' or 'int64'"),
					rootNode,
				))
			}
		}
	}

	return errs
}
