package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

const RuleSemanticDuplicatedEnum = "semantic-duplicated-enum"

type DuplicatedEnumRule struct{}

func (r *DuplicatedEnumRule) ID() string       { return RuleSemanticDuplicatedEnum }
func (r *DuplicatedEnumRule) Category() string { return CategorySemantic }
func (r *DuplicatedEnumRule) Description() string {
	return "Enum arrays should not contain duplicate values. Duplicate enum values are redundant and can cause confusion or unexpected behavior in client code generation and validation."
}
func (r *DuplicatedEnumRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-duplicated-enum"
}
func (r *DuplicatedEnumRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *DuplicatedEnumRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *DuplicatedEnumRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Use the pre-computed schema indexes to find all schemas with enums
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		// Check if schema has enum values
		enumValues := schema.GetEnum()
		if len(enumValues) == 0 {
			continue
		}

		// Check for duplicates
		duplicateIndices := findDuplicateIndices(enumValues)
		for value, indices := range duplicateIndices {
			// Report on first duplicate occurrence (second index in the list)
			if len(indices) > 1 {
				errs = append(errs, validation.NewSliceError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleSemanticDuplicatedEnum,
					fmt.Errorf("enum contains a duplicate: `%s`", value),
					schema.GetCore(),
					schema.GetCore().Enum,
					indices[1], // Report at second occurrence
				))
			}
		}
	}

	return errs
}

// findDuplicateIndices identifies duplicate enum values and their indices
func findDuplicateIndices(enumValues []*yaml.Node) map[string][]int {
	seen := make(map[string][]int)

	for i, node := range enumValues {
		key := nodeToString(node)
		seen[key] = append(seen[key], i)
	}

	// Filter to only duplicates
	duplicates := make(map[string][]int)
	for key, indices := range seen {
		if len(indices) > 1 {
			duplicates[key] = indices
		}
	}

	return duplicates
}

// nodeToString converts a yaml.Node to a string representation for comparison
func nodeToString(node *yaml.Node) string {
	if node == nil {
		return "null"
	}

	switch node.Tag {
	case "!!null":
		return "null"
	case "!!str":
		return "string:" + node.Value
	case "!!int":
		return "int:" + node.Value
	case "!!float":
		return "float:" + node.Value
	case "!!bool":
		return "bool:" + node.Value
	default:
		return node.Value
	}
}
