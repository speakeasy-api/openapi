package rules

import (
	"context"
	"errors"
	"strings"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

const RuleSemanticAllOfTypeConflict = "semantic-allof-type-conflict"

// AllOfTypeConflictRule checks only explicit types on a schema and its immediate
// allOf branches. It does not attempt general schema satisfiability analysis.
type AllOfTypeConflictRule struct{}

func (r *AllOfTypeConflictRule) ID() string       { return RuleSemanticAllOfTypeConflict }
func (r *AllOfTypeConflictRule) Category() string { return CategorySemantic }
func (r *AllOfTypeConflictRule) Description() string {
	return "Explicit type constraints on a schema and its immediate allOf branches must have a non-empty intersection. A conflict means no value can satisfy those constraints, not that the OpenAPI document is structurally invalid."
}
func (r *AllOfTypeConflictRule) Summary() string {
	return "Explicit allOf type constraints must admit a common value type."
}
func (r *AllOfTypeConflictRule) HowToFix() string {
	return "Align the schema and allOf branch types, or use anyOf or oneOf if the branches are intended as alternatives."
}
func (r *AllOfTypeConflictRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-allof-type-conflict"
}
func (r *AllOfTypeConflictRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *AllOfTypeConflictRule) Versions() []string { return nil }

func (r *AllOfTypeConflictRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}
	version := docInfo.Document.GetOpenAPI()
	isOAS30 := strings.HasPrefix(version, "3.0.")
	if !isOAS30 && !strings.HasPrefix(version, "3.1.") && !strings.HasPrefix(version, "3.2.") {
		return nil
	}
	// Custom dialects may redefine the keywords this bounded rule relies on.
	if docInfo.Document.GetJSONSchemaDialect() != "" {
		return nil
	}

	var errs []error
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		if schemaNode == nil || schemaNode.Node == nil || schemaNode.Node.IsReference() {
			continue
		}
		schema := schemaNode.Node.GetSchema()
		if schema == nil || schema.GetCore() == nil || !schema.GetCore().AllOf.Present || len(schema.GetAllOf()) == 0 {
			continue
		}
		intersection, safe := allOfExplicitTypes(schema, isOAS30)
		if !safe {
			continue
		}
		for _, branch := range schema.GetAllOf() {
			branchSchema := allOfTypeBranchSchema(branch)
			branchTypes, branchSafe := allOfExplicitTypes(branchSchema, isOAS30)
			if !branchSafe {
				safe = false
				break
			}
			intersection &= branchTypes
		}
		if !safe || intersection != 0 {
			continue
		}
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleSemanticAllOfTypeConflict,
			errors.New("`allOf` has incompatible explicit type constraints on this schema and its immediate branches; no value can satisfy their intersection"),
			schema.GetCore().AllOf.GetKeyNodeOrRoot(schema.GetRootNode()),
		))
	}
	return errs
}

// Numeric types use separate integer and non-integer bits, so number includes
// integer without making integer and number disjoint.
const (
	allOfTypeObject uint8 = 1 << iota
	allOfTypeString
	allOfTypeInteger
	allOfTypeNonInteger
	allOfTypeBoolean
	allOfTypeArray
	allOfTypeNull
	allOfAllTypes = allOfTypeObject | allOfTypeString | allOfTypeInteger | allOfTypeNonInteger | allOfTypeBoolean | allOfTypeArray | allOfTypeNull
)

func allOfExplicitTypes(schema *oas3.Schema, isOAS30 bool) (uint8, bool) {
	if schema == nil || schema.IsReference() || schema.GetCore() == nil || schema.Schema != nil || schema.ID != nil {
		return 0, false
	}
	core := schema.GetCore()
	if core.Schema.KeyNode != nil || core.ID.KeyNode != nil || (isOAS30 && core.Nullable.ValueNode != nil && !core.Nullable.Present) {
		return 0, false
	}
	if !core.Type.Present {
		if core.Type.ValueNode != nil {
			return 0, false
		}
		// No local type is unconstrained. In particular, types inside anyOf,
		// oneOf, or nested allOf are not inferred here.
		return allOfAllTypes, true
	}
	typeNode := core.Type.ValueNode
	if typeNode == nil {
		return 0, false
	}
	typeNodes := []*yaml.Node{typeNode}
	if typeNode.Kind == yaml.SequenceNode {
		if isOAS30 || len(typeNode.Content) == 0 {
			return 0, false
		}
		typeNodes = typeNode.Content
	}
	var types, declaredTypes uint8
	for _, node := range typeNodes {
		if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
			return 0, false
		}
		var typeBit uint8
		switch node.Value {
		case "object":
			typeBit = allOfTypeObject
		case "string":
			typeBit = allOfTypeString
		case "integer":
			typeBit = allOfTypeInteger
		case "number":
			typeBit = allOfTypeInteger | allOfTypeNonInteger
		case "boolean":
			typeBit = allOfTypeBoolean
		case "array":
			typeBit = allOfTypeArray
		case "null":
			if isOAS30 {
				return 0, false
			}
			typeBit = allOfTypeNull
		default:
			return 0, false
		}
		// Duplicate members are invalid. Track the number declaration with
		// its non-integer bit so [number, integer] remains a valid union.
		declarationBit := typeBit
		if node.Value == "number" {
			declarationBit = allOfTypeNonInteger
		}
		if declaredTypes&declarationBit != 0 {
			return 0, false
		}
		declaredTypes |= declarationBit
		types |= typeBit
	}
	if isOAS30 && core.Nullable.Present {
		nullableNode := core.Nullable.ValueNode
		if nullableNode == nil || nullableNode.Kind != yaml.ScalarNode || nullableNode.Tag != "!!bool" {
			return 0, false
		}
		if schema.GetNullable() {
			types |= allOfTypeNull
		}
	}
	return types, true
}

func allOfTypeBranchSchema(branch *oas3.JSONSchemaReferenceable) *oas3.Schema {
	if branch == nil || branch.IsBool() {
		return nil
	}
	schema := branch.GetSchema()
	if schema == nil || !branch.IsReference() {
		return schema
	}
	// Only an already-resolved, single-hop, pure local component reference
	// is supported. Do not resolve documents or follow chains, and do not use
	// a reference's sibling type as though it were its target's constraint.
	root := schema.GetRootNode()
	if !strings.HasPrefix(string(branch.GetRef()), "#/components/schemas/") || root == nil || root.Kind != yaml.MappingNode || len(root.Content) != 2 || root.Content[0].Value != "$ref" {
		return nil
	}
	resolution := branch.GetReferenceResolutionInfo()
	if resolution == nil || resolution.Object == nil || resolution.Object.IsReference() || resolution.Object.IsBool() {
		return nil
	}
	return resolution.Object.GetSchema()
}
