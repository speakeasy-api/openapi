package rules

import (
	"context"
	"fmt"

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
			if !isNodeMatchingType(enumValueNode, schemaTypes) {
				errs = append(errs, validation.NewSliceError(
					config.GetSeverity(r.DefaultSeverity()),
					RuleSemanticTypedEnum,
					fmt.Errorf("enum value at index %d does not match schema type %v", i, schemaTypes),
					schema.GetCore(),
					schema.GetCore().Enum,
					i,
				))
			}
		}
	}

	return errs
}

// isNodeMatchingType checks if a yaml.Node value matches the schema type
func isNodeMatchingType(node *yaml.Node, schemaTypes []oas3.SchemaType) bool {
	if node == nil || node.Kind == yaml.AliasNode {
		// nil or alias nodes - check for null type
		return containsType(schemaTypes, oas3.SchemaTypeNull)
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
		return containsType(schemaTypes, oas3.SchemaTypeNull)
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
