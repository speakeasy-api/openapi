package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

const RuleOAS3NoNullable = "oas3-no-nullable"

type OAS3NoNullableRule struct{}

func (r *OAS3NoNullableRule) ID() string {
	return RuleOAS3NoNullable
}
func (r *OAS3NoNullableRule) Category() string {
	return CategorySchemas
}
func (r *OAS3NoNullableRule) Description() string {
	return "The `nullable` keyword is not supported in OpenAPI 3.1+ and should be replaced with a type array that includes null (e.g., `type: [string, null]`). This change aligns OpenAPI 3.1 with JSON Schema Draft 2020-12, which uses type arrays to express nullable values."
}
func (r *OAS3NoNullableRule) Summary() string {
	return "OpenAPI 3.1 must not use the `nullable` keyword."
}
func (r *OAS3NoNullableRule) HowToFix() string {
	return "Replace `nullable` with a type array that includes `null` (e.g., `type: [string, null]`)."
}
func (r *OAS3NoNullableRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#oas3-no-nullable"
}
func (r *OAS3NoNullableRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OAS3NoNullableRule) Versions() []string {
	return []string{"3.1"} // Only applies to OpenAPI 3.1+
}

func (r *OAS3NoNullableRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Check all schemas for nullable keyword
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		coreSchema := schema.GetCore()
		if coreSchema == nil {
			continue
		}

		// Check if nullable field is present in the YAML
		if coreSchema.Nullable.Present {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, &validation.Error{
					UnderlyingError: errors.New("the `nullable` keyword is not supported in OpenAPI 3.1 - use `type: [actualType, \"null\"]` instead"),
					Node:            rootNode,
					Severity:        config.GetSeverity(r.DefaultSeverity()),
					Rule:            RuleOAS3NoNullable,
					Fix:             &removeNullableFix{schemaNode: rootNode, typeValueNode: coreSchema.Type.ValueNode},
				})
			}
		}
	}

	return errs
}

// removeNullableFix removes the nullable key and adds "null" to the type array.
type removeNullableFix struct {
	schemaNode    *yaml.Node // the schema mapping node
	typeValueNode *yaml.Node // the existing type value node (may be nil)
}

func (f *removeNullableFix) Description() string {
	return "Replace nullable with type array including null"
}
func (f *removeNullableFix) Interactive() bool            { return false }
func (f *removeNullableFix) Prompts() []validation.Prompt { return nil }
func (f *removeNullableFix) SetInput([]string) error      { return nil }
func (f *removeNullableFix) Apply(doc any) error          { return nil }

func (f *removeNullableFix) ApplyNode(_ *yaml.Node) error {
	if f.schemaNode == nil {
		return nil
	}

	ctx := context.Background()

	// Remove the nullable key/value pair
	yml.DeleteMapNodeElement(ctx, "nullable", f.schemaNode)

	// Add "null" to the type field
	if f.typeValueNode != nil {
		switch f.typeValueNode.Kind {
		case yaml.ScalarNode:
			// type: string → type: [string, "null"]
			existingType := f.typeValueNode.Value
			f.typeValueNode.Kind = yaml.SequenceNode
			f.typeValueNode.Tag = "!!seq"
			f.typeValueNode.Value = ""
			f.typeValueNode.Content = []*yaml.Node{
				yml.CreateStringNode(existingType),
				yml.CreateStringNode("null"),
			}
		case yaml.SequenceNode:
			// type: [string, integer] → type: [string, integer, "null"]
			// Check if "null" already present
			for _, n := range f.typeValueNode.Content {
				if n.Value == "null" {
					return nil
				}
			}
			f.typeValueNode.Content = append(f.typeValueNode.Content, yml.CreateStringNode("null"))
		}
	} else {
		// No type field exists — add type: "null"
		yml.CreateOrUpdateMapNodeElement(ctx, "type", nil, yml.CreateStringNode("null"), f.schemaNode)
	}

	return nil
}
