package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleNoRefSiblings = "style-no-ref-siblings"

type NoRefSiblingsRule struct{}

func (r *NoRefSiblingsRule) ID() string       { return RuleStyleNoRefSiblings }
func (r *NoRefSiblingsRule) Category() string { return CategoryStyle }
func (r *NoRefSiblingsRule) Description() string {
	return "In OpenAPI 3.0.x, a $ref field should not have sibling properties alongside it in the same object. Either use $ref alone or move additional properties to the referenced schema definition. Note that OpenAPI 3.1+ allows $ref siblings per JSON Schema Draft 2020-12."
}
func (r *NoRefSiblingsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-no-ref-siblings"
}
func (r *NoRefSiblingsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *NoRefSiblingsRule) Versions() []string {
	// Only applies to OAS 3.0.x (in OAS 3.1+, $ref can have siblings per JSON Schema Draft 2020-12)
	return []string{"3.0.0", "3.0.1", "3.0.2", "3.0.3"}
}

func (r *NoRefSiblingsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Check schema references for $ref with siblings
	// SchemaReferences contains all schemas with $ref (whether or not they have siblings)
	for _, schemaNode := range docInfo.Index.SchemaReferences {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		// Check if the schema has siblings (i.e., is not reference-only)
		if !schema.IsReferenceOnly() {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStyleNoRefSiblings,
				errors.New("schema contains $ref with sibling properties, which is not allowed in OAS 3.0.x"),
				schema.GetRootNode(),
			))
		}
	}

	return errs
}
