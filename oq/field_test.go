package oq_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq"
	"github.com/speakeasy-api/openapi/oq/expr"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
)

func TestFieldValuePublic_SchemaContent_Success(t *testing.T) {
	t.Parallel()

	child := oas3.NewJSONSchemaFromBool(true)
	schema := &oas3.Schema{
		Description: pointer.From("description"), Title: pointer.From("title"),
		Format: pointer.From("uuid"), Pattern: pointer.From("^[a-z]+$"),
		Nullable: pointer.From(true), ReadOnly: pointer.From(true), WriteOnly: pointer.From(true),
		Deprecated: pointer.From(true), UniqueItems: pointer.From(true),
		Discriminator: &oas3.Discriminator{
			PropertyName: "kind",
			Mapping:      sequencedmap.New(sequencedmap.NewElem("pet", "#/components/schemas/Pet")),
		},
		Required: []string{"id"}, Enum: []values.Value{yml.CreateStringNode("a"), yml.CreateStringNode("b")},
		Minimum: pointer.From(2.0), Maximum: pointer.From(10.0),
		MinLength: pointer.From(int64(3)), MaxLength: pointer.From(int64(12)),
		MinItems: pointer.From(int64(1)), MaxItems: pointer.From(int64(4)),
		MinProperties: pointer.From(int64(2)), MaxProperties: pointer.From(int64(5)),
		Default: yml.CreateStringNode("fallback"), Const: yml.CreateStringNode("fixed"),
		ContentEncoding: pointer.From("base64"), ContentMediaType: pointer.From("image/png"),
		MultipleOf: pointer.From(2.0), Anchor: pointer.From("pet"), ID: pointer.From("https://example.com/pet"),
		Schema:               pointer.From("https://json-schema.org/draft/2020-12/schema"),
		PrefixItems:          []*oas3.JSONSchemaReferenceable{child},
		DependentSchemas:     sequencedmap.New(sequencedmap.NewElem("id", child)),
		Defs:                 sequencedmap.New(sequencedmap.NewElem("Pet", child)),
		Examples:             []values.Value{yml.CreateStringNode("example")},
		AdditionalProperties: child, PatternProperties: sequencedmap.New(sequencedmap.NewElem("^x-", child)),
		XML: &oas3.XML{Name: pointer.From("pet")}, ExternalDocs: &oas3.ExternalDocumentation{URL: "https://example.com"},
		Not: child, If: child, Then: child, Else: child, Contains: child, PropertyNames: child,
		UnevaluatedItems: child, UnevaluatedProperties: child,
		Extensions: extensions.New(extensions.NewElem("x-test-value", yml.CreateStringNode("extension"))),
	}

	g := &graph.SchemaGraph{Schemas: []graph.SchemaNode{{Schema: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)}}}
	row := oq.Row{Kind: oq.SchemaResult, SchemaIdx: 0, Traversal: "Pet/allOf/Base"}

	tests := []struct {
		field string
		want  expr.Value
	}{
		{"description", expr.StringVal("description")}, {"title", expr.StringVal("title")},
		{"format", expr.StringVal("uuid")}, {"pattern", expr.StringVal("^[a-z]+$")},
		{"nullable", expr.BoolVal(true)}, {"readOnly", expr.BoolVal(true)}, {"writeOnly", expr.BoolVal(true)},
		{"deprecated", expr.BoolVal(true)}, {"uniqueItems", expr.BoolVal(true)},
		{"discriminatorProperty", expr.StringVal("kind")}, {"discriminatorMappingCount", expr.IntVal(1)},
		{"requiredProperties", expr.ArrayVal([]string{"id"})}, {"requiredCount", expr.IntVal(1)},
		{"enum", expr.ArrayVal([]string{"a", "b"})}, {"enumCount", expr.IntVal(2)},
		{"minimum", expr.IntVal(2)}, {"maximum", expr.IntVal(10)},
		{"minLength", expr.IntVal(3)}, {"maxLength", expr.IntVal(12)},
		{"minItems", expr.IntVal(1)}, {"maxItems", expr.IntVal(4)},
		{"minProperties", expr.IntVal(2)}, {"maxProperties", expr.IntVal(5)},
		{"default", expr.StringVal("fallback")}, {"const", expr.StringVal("fixed")},
		{"contentEncoding", expr.StringVal("base64")}, {"contentMediaType", expr.StringVal("image/png")},
		{"extensionCount", expr.IntVal(1)}, {"x-test-value", expr.StringVal("extension")}, {"x_test_value", expr.StringVal("extension")},
		{"multiple_of", expr.IntVal(2)}, {"anchor", expr.StringVal("pet")}, {"id", expr.StringVal("https://example.com/pet")},
		{"schema", expr.StringVal("Base")},
		{"prefix_items", expr.IntVal(1)}, {"dependent_schemas", expr.IntVal(1)}, {"defs", expr.IntVal(1)}, {"examples", expr.IntVal(1)},
		{"additional_properties", expr.BoolVal(true)}, {"pattern_properties", expr.BoolVal(true)},
		{"xml", expr.BoolVal(true)}, {"external_docs", expr.BoolVal(true)},
		{"not", expr.BoolVal(true)}, {"if", expr.BoolVal(true)}, {"then", expr.BoolVal(true)}, {"else", expr.BoolVal(true)},
		{"contains", expr.BoolVal(true)}, {"property_names", expr.BoolVal(true)},
		{"unevaluated_items", expr.BoolVal(true)}, {"unevaluated_properties", expr.BoolVal(true)},
		{"unknown", expr.NullVal()}, {"x-missing", expr.NullVal()},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, oq.FieldValuePublic(row, tt.field, g), "field should preserve its value and expression type")
		})
	}
}

func TestFieldValuePublic_MissingSchemaContent_ReturnsDefaults(t *testing.T) {
	t.Parallel()

	g := &graph.SchemaGraph{Schemas: []graph.SchemaNode{{Schema: oas3.NewJSONSchemaFromBool(true)}}}
	row := oq.Row{Kind: oq.SchemaResult, SchemaIdx: 0}
	tests := []struct {
		fields []string
		want   expr.Value
	}{
		{[]string{"description", "title", "format", "pattern", "discriminatorProperty", "contentEncoding", "contentMediaType"}, expr.StringVal("")},
		{[]string{"nullable", "readOnly", "writeOnly", "deprecated", "uniqueItems"}, expr.BoolVal(false)},
		{[]string{"discriminatorMappingCount", "requiredCount", "enumCount", "extensionCount"}, expr.IntVal(0)},
		{[]string{"requiredProperties", "enum"}, expr.ArrayVal(nil)},
		{[]string{"minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems", "minProperties", "maxProperties", "default", "unknown"}, expr.NullVal()},
	}
	for _, tt := range tests {
		for _, field := range tt.fields {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, oq.FieldValuePublic(row, field, g), "boolean schemas should return the documented default for content fields")
			})
		}
	}
}

func TestFieldValuePublic_AbsentRawFields_ReturnsNull(t *testing.T) {
	t.Parallel()

	g := &graph.SchemaGraph{Schemas: []graph.SchemaNode{{Schema: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{})}}}
	row := oq.Row{Kind: oq.SchemaResult, SchemaIdx: 0}
	fields := []string{"const", "multipleOf", "anchor", "id", "prefixItems", "dependentSchemas", "defs", "examples", "additionalProperties", "patternProperties", "xml", "externalDocs", "not", "if", "then", "else", "contains", "propertyNames", "unevaluatedItems", "unevaluatedProperties", "x-missing"}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expr.NullVal(), oq.FieldValuePublic(row, field, g), "an absent raw field should not be reported as present")
		})
	}
}

func TestFieldValuePublic_AbsentNavigation_ReturnsNull(t *testing.T) {
	t.Parallel()

	g := &graph.SchemaGraph{}
	tests := []struct {
		name string
		row  oq.Row
	}{
		{"negative schema index", oq.Row{Kind: oq.SchemaResult, SchemaIdx: -1}},
		{"out of range schema index", oq.Row{Kind: oq.SchemaResult}},
		{"negative operation index", oq.Row{Kind: oq.OperationResult, OpIdx: -1}},
		{"out of range operation index", oq.Row{Kind: oq.OperationResult}},
		{"parameter", oq.Row{Kind: oq.ParameterResult}}, {"response", oq.Row{Kind: oq.ResponseResult}},
		{"request body", oq.Row{Kind: oq.RequestBodyResult}}, {"content type", oq.Row{Kind: oq.ContentTypeResult}},
		{"header", oq.Row{Kind: oq.HeaderResult}}, {"security scheme", oq.Row{Kind: oq.SecuritySchemeResult}},
		{"server", oq.Row{Kind: oq.ServerResult}}, {"tag", oq.Row{Kind: oq.TagResult}}, {"link", oq.Row{Kind: oq.LinkResult}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expr.NullVal(), oq.FieldValuePublic(tt.row, "name", g), "missing navigation objects should produce null rather than panic")
		})
	}
}

func TestFieldValuePublic_NavigationMetadata_Success(t *testing.T) {
	t.Parallel()

	g := &graph.SchemaGraph{Operations: []graph.OperationNode{{Name: "getPets"}}}
	tests := []struct {
		name   string
		row    oq.Row
		fields map[string]expr.Value
	}{
		{
			name: "parameter flags",
			row: oq.Row{Kind: oq.ParameterResult, Parameter: &openapi.Parameter{
				AllowEmptyValue: pointer.From(true), AllowReserved: pointer.From(true), Explode: pointer.From(true),
				Style: pointer.From(openapi.SerializationStyle("form")),
			}},
			fields: map[string]expr.Value{"allowEmptyValue": expr.BoolVal(true), "allowReserved": expr.BoolVal(true), "explode": expr.BoolVal(true), "style": expr.StringVal("form"), "operation": expr.StringVal("getPets")},
		},
		{
			name: "link metadata",
			row: oq.Row{Kind: oq.LinkResult, LinkName: "next", StatusCode: "200", Link: &openapi.Link{
				OperationID: pointer.From("nextPage"), OperationRef: pointer.From("#/paths/~1next/get"), Description: pointer.From("next page"), Server: &openapi.Server{},
			}},
			fields: map[string]expr.Value{"name": expr.StringVal("next"), "operationId": expr.StringVal("nextPage"), "operationRef": expr.StringVal("#/paths/~1next/get"), "description": expr.StringVal("next page"), "hasServer": expr.BoolVal(true), "statusCode": expr.StringVal("200"), "operation": expr.StringVal("getPets")},
		},
		{
			name:   "empty link defaults",
			row:    oq.Row{Kind: oq.LinkResult, SourceOpIdx: -1, Link: &openapi.Link{}},
			fields: map[string]expr.Value{"operationId": expr.StringVal(""), "operationRef": expr.StringVal(""), "description": expr.StringVal(""), "parameterCount": expr.IntVal(0), "hasServer": expr.BoolVal(false), "hasRequestBody": expr.BoolVal(false), "operation": expr.StringVal("")},
		},
		{
			name:   "security scopes without scheme",
			row:    oq.Row{Kind: oq.SecurityRequirementResult, Scopes: []string{"read", "write"}, SchemeName: "oauth"},
			fields: map[string]expr.Value{"schemeName": expr.StringVal("oauth"), "schemeType": expr.StringVal(""), "scopes": expr.ArrayVal([]string{"read", "write"}), "scopeCount": expr.IntVal(2)},
		},
		{
			name:   "empty tag metadata",
			row:    oq.Row{Kind: oq.TagResult, Tag: &openapi.Tag{Name: "pets"}},
			fields: map[string]expr.Value{"name": expr.StringVal("pets"), "description": expr.StringVal(""), "summary": expr.StringVal("")},
		},
		{
			name:   "empty server variables",
			row:    oq.Row{Kind: oq.ServerResult, Server: &openapi.Server{URL: "https://example.com"}},
			fields: map[string]expr.Value{"url": expr.StringVal("https://example.com"), "variableCount": expr.IntVal(0)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for field, want := range tt.fields {
				assert.Equal(t, want, oq.FieldValuePublic(tt.row, field, g), "navigation field %s should return its metadata or default", field)
			}
		})
	}
}
