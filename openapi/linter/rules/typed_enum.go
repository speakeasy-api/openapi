package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

const RuleSemanticTypedEnum = "semantic-typed-enum"

type TypedEnumRule struct{}

func (r *TypedEnumRule) ID() string       { return RuleSemanticTypedEnum }
func (r *TypedEnumRule) Category() string { return CategorySemantic }
func (r *TypedEnumRule) Description() string {
	return "Enum values must match the specified type - for example, if type is 'string', all enum values must be strings. Type mismatches in enums cause validation failures and break code generation tools."
}
func (r *TypedEnumRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-typed-enum"
}
func (r *TypedEnumRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *TypedEnumRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *TypedEnumRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Get OpenAPI version for version-specific error messages
	openapiVersion := docInfo.Document.GetOpenAPI()

	// Use the pre-computed schema indexes to find all schemas with enums
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		// Check if schema has enum values
		if len(schema.GetEnum()) == 0 {
			continue
		}

		// Get the schema type
		schemaTypes := schema.GetType()
		if len(schemaTypes) == 0 {
			// No type specified, skip validation
			continue
		}

		// Validate each enum value against the type
		for i, enumValueNode := range schema.GetEnum() {
			if !isNodeMatchingType(enumValueNode, schemaTypes, schema.GetNullable()) {
				errorMsg := createTypeMismatchError(i, enumValueNode, schemaTypes, schema.GetNullable(), openapiVersion)
				errs = append(errs, validation.NewSliceError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleSemanticTypedEnum,
					fmt.Errorf("%s", errorMsg),
					schema.GetCore(),
					schema.GetCore().Enum,
					i,
				))
			}
		}
	}

	return errs
}

// createTypeMismatchError creates an appropriate error message for type mismatches
func createTypeMismatchError(index int, node *yaml.Node, schemaTypes []oas3.SchemaType, nullable bool, openapiVersion string) string {
	isNull := isNullNode(node)

	if isNull && !nullable && !containsType(schemaTypes, oas3.SchemaTypeNull) {
		// Special error message for null values without proper nullable declaration
		if len(openapiVersion) >= 3 && openapiVersion[:3] == "3.0" {
			// OpenAPI 3.0.x - suggest nullable: true
			return fmt.Sprintf("enum contains null at index %d but schema does not have 'nullable: true'. Add 'nullable: true' to allow null values", index)
		}
		// OpenAPI 3.1.x or later - suggest type array with null
		typeWithNull := formatTypeArrayWithNull(schemaTypes)
		return fmt.Sprintf("enum contains null at index %d but schema type does not include null. Change 'type: %v' to 'type: %s' to allow null values", index, schemaTypes, typeWithNull)
	}

	// Generic type mismatch error
	return fmt.Sprintf("enum value at index %d does not match schema type %v", index, schemaTypes)
}

// isNullNode checks if a YAML node represents a null value
func isNullNode(node *yaml.Node) bool {
	if node == nil {
		return true
	}
	if node.Kind == yaml.AliasNode {
		return true
	}
	return node.Tag == "!!null"
}

// formatTypeArrayWithNull formats a type array suggestion with null included
func formatTypeArrayWithNull(schemaTypes []oas3.SchemaType) string {
	if len(schemaTypes) == 0 {
		return `["null"]`
	}
	if len(schemaTypes) == 1 {
		return fmt.Sprintf(`[%q, "null"]`, schemaTypes[0])
	}
	// Multiple types - add null to the array
	types := make([]string, len(schemaTypes)+1)
	for i, t := range schemaTypes {
		types[i] = fmt.Sprintf("%q", t)
	}
	types[len(types)-1] = `"null"`

	var result strings.Builder
	result.WriteString("[")
	for i, t := range types {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(t)
	}
	result.WriteString("]")
	return result.String()
}

// isNodeMatchingType checks if a yaml.Node value matches the schema type
func isNodeMatchingType(node *yaml.Node, schemaTypes []oas3.SchemaType, nullable bool) bool {
	if node == nil || node.Kind == yaml.AliasNode {
		// nil or alias nodes - check for null type or nullable schema
		return containsType(schemaTypes, oas3.SchemaTypeNull) || nullable
	}

	// Check based on yaml node tag
	switch node.Tag {
	case "!!str":
		return containsType(schemaTypes, oas3.SchemaTypeString)
	case "!!int":
		// Integer can match both integer and number types
		return containsType(schemaTypes, oas3.SchemaTypeInteger) || containsType(schemaTypes, oas3.SchemaTypeNumber)
	case "!!float":
		// Float can match number or integer types
		return containsType(schemaTypes, oas3.SchemaTypeNumber) || containsType(schemaTypes, oas3.SchemaTypeInteger)
	case "!!bool":
		return containsType(schemaTypes, oas3.SchemaTypeBoolean)
	case "!!seq":
		return containsType(schemaTypes, oas3.SchemaTypeArray)
	case "!!map":
		return containsType(schemaTypes, oas3.SchemaTypeObject)
	case "!!null":
		return containsType(schemaTypes, oas3.SchemaTypeNull) || nullable
	default:
		// Unknown tag, be permissive
		return true
	}
}

func containsType(schemaType []oas3.SchemaType, targetType oas3.SchemaType) bool {
	for _, t := range schemaType {
		if t == targetType {
			return true
		}
	}
	return false
}
