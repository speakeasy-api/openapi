package rules

import (
	"context"
	"errors"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOAS3NullableWithoutType = "oas3-nullable-without-type"

type OAS3NullableWithoutTypeRule struct{}

func (r *OAS3NullableWithoutTypeRule) ID() string       { return RuleOAS3NullableWithoutType }
func (r *OAS3NullableWithoutTypeRule) Category() string { return CategorySchemas }
func (r *OAS3NullableWithoutTypeRule) Description() string {
	return "In OpenAPI 3.0, `nullable: true` has no effect unless the same Schema Object declares a `type`."
}
func (r *OAS3NullableWithoutTypeRule) Summary() string {
	return "`nullable: true` without a local schema type has no effect."
}
func (r *OAS3NullableWithoutTypeRule) HowToFix() string {
	return "Add an explicit `type` to this Schema Object, or remove `nullable: true` if no type is intended."
}
func (r *OAS3NullableWithoutTypeRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#oas3-nullable-without-type"
}
func (r *OAS3NullableWithoutTypeRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OAS3NullableWithoutTypeRule) Versions() []string { return []string{"3.0"} }

func (r *OAS3NullableWithoutTypeRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}
	version := docInfo.Document.GetOpenAPI()
	if !strings.HasPrefix(version, "3.0.") {
		return nil
	}

	var errs []error
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		if refSchema == nil {
			continue
		}
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}
		core := schema.GetCore()
		if core == nil || !core.Nullable.Present || core.Nullable.Value == nil || !*core.Nullable.Value || core.Type.Present {
			continue
		}
		if nullableKeyNode := core.Nullable.KeyNode; nullableKeyNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleOAS3NullableWithoutType,
				errors.New("`nullable: true` has no effect without a `type` declared on the same Schema Object; this is a no-op, not a structurally invalid schema"),
				nullableKeyNode,
			))
		}
	}
	return errs
}
