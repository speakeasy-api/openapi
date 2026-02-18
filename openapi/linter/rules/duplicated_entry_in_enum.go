package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v4"
)

const RuleSemanticDuplicatedEnum = "semantic-duplicated-enum"

type DuplicatedEnumRule struct{}

func (r *DuplicatedEnumRule) ID() string       { return RuleSemanticDuplicatedEnum }
func (r *DuplicatedEnumRule) Category() string { return CategorySemantic }
func (r *DuplicatedEnumRule) Description() string {
	return "Enum arrays should not contain duplicate values. Duplicate enum values are redundant and can cause confusion or unexpected behavior in client code generation and validation."
}
func (r *DuplicatedEnumRule) Summary() string {
	return "Enum arrays should not contain duplicate values."
}
func (r *DuplicatedEnumRule) HowToFix() string {
	return "Remove or consolidate duplicate entries in enum arrays."
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
		coreSchema := schema.GetCore()
		duplicateIndices := findDuplicateIndices(enumValues)
		for _, indices := range duplicateIndices {
			// Report on first duplicate occurrence (second index in the list)
			if len(indices) > 1 {
				displayValue := nodeToDisplayString(enumValues[indices[1]])
				errNode := coreSchema.Enum.GetSliceValueNodeOrRoot(indices[1], refSchema.GetRootNode())
				errs = append(errs, &validation.Error{
					UnderlyingError: fmt.Errorf("enum contains a duplicate: `%s`", displayValue),
					Node:            errNode,
					Severity:        config.GetSeverity(r.DefaultSeverity()),
					Rule:            RuleSemanticDuplicatedEnum,
					Fix:             &removeDuplicateEnumFix{enumNode: coreSchema.Enum.ValueNode, duplicateIndices: indices[1:]},
				})
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

// removeDuplicateEnumFix removes duplicate entries from an enum sequence node.
type removeDuplicateEnumFix struct {
	enumNode         *yaml.Node // the sequence node containing enum values
	duplicateIndices []int      // indices of duplicate entries to remove
}

func (f *removeDuplicateEnumFix) Description() string          { return "Remove duplicate enum entries" }
func (f *removeDuplicateEnumFix) Interactive() bool            { return false }
func (f *removeDuplicateEnumFix) Prompts() []validation.Prompt { return nil }
func (f *removeDuplicateEnumFix) SetInput([]string) error      { return nil }
func (f *removeDuplicateEnumFix) Apply(doc any) error          { return nil }

func (f *removeDuplicateEnumFix) ApplyNode(_ *yaml.Node) error {
	if f.enumNode == nil || len(f.duplicateIndices) == 0 {
		return nil
	}
	// Remove from last index first to preserve earlier indices
	for i := len(f.duplicateIndices) - 1; i >= 0; i-- {
		idx := f.duplicateIndices[i]
		if idx < len(f.enumNode.Content) {
			f.enumNode.Content = append(f.enumNode.Content[:idx], f.enumNode.Content[idx+1:]...)
		}
	}
	return nil
}

// nodeToString converts a yaml.Node to a string representation for comparison
// This includes type prefixes to distinguish between different types of the same value
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

// nodeToDisplayString converts a yaml.Node to a string representation for display in error messages
// For strings, the type prefix is omitted since it's implicit
func nodeToDisplayString(node *yaml.Node) string {
	if node == nil {
		return "null"
	}

	switch node.Tag {
	case "!!null":
		return "null"
	case "!!str":
		return node.Value // String type is implicit, no prefix needed
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
