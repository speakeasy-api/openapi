package asyncapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func unmarshalSchemaUnion(t *testing.T, doc string) (*asyncapi.SchemaUnion, []error) {
	t.Helper()

	var u asyncapi.SchemaUnion
	validationErrs, err := marshaller.Unmarshal(t.Context(), strings.NewReader(doc), &u)
	require.NoError(t, err, "unmarshal should succeed")
	return &u, validationErrs
}

func marshalSchemaUnion(t *testing.T, u *asyncapi.SchemaUnion) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, marshaller.Marshal(t.Context(), u, &buf), "marshal should succeed")
	return buf.String()
}

func TestSchemaUnion_Unmarshal_SelectsShape_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		doc           string
		isSchema      bool
		isMultiFormat bool
		isReference   bool
		isBool        bool
	}{
		{
			name:     "native schema",
			doc:      "type: object\nproperties:\n  id:\n    type: string\n",
			isSchema: true,
		},
		{
			name:     "boolean schema",
			doc:      "true\n",
			isSchema: true,
			isBool:   true,
		},
		{
			name:        "reference object",
			doc:         "$ref: '#/components/schemas/User'\n",
			isSchema:    true,
			isReference: true,
		},
		{
			name:        "reference wins over schemaFormat",
			doc:         "$ref: '#/components/schemas/User'\nschemaFormat: application/vnd.apache.avro;version=1.9.0\n",
			isSchema:    true,
			isReference: true,
		},
		{
			name:          "multi format schema",
			doc:           "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n  name: User\n",
			isMultiFormat: true,
		},
		{
			name:     "object with only schema key is a native schema",
			doc:      "schema:\n  type: string\n",
			isSchema: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, _ := unmarshalSchemaUnion(t, tt.doc)
			assert.Equal(t, tt.isSchema, u.IsSchema(), "IsSchema should match")
			assert.Equal(t, tt.isMultiFormat, u.IsMultiFormat(), "IsMultiFormat should match")
			assert.Equal(t, tt.isReference, u.IsReference(), "IsReference should match")
			assert.Equal(t, tt.isBool, u.GetSchema().IsBool(), "IsBool should match")
		})
	}
}

func TestMultiFormatSchema_Unmarshal_JSONSchemaBasedFormats_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
	}{
		{
			name: "native asyncapi format",
			doc:  "schemaFormat: application/vnd.aai.asyncapi+json;version=3.1.0\nschema:\n  type: string\n",
		},
		{
			name: "draft 07 format",
			doc:  "schemaFormat: application/schema+json;version=draft-07\nschema:\n  type: string\n",
		},
		{
			name: "schema before schemaFormat",
			doc:  "schema:\n  type: string\nschemaFormat: application/schema+yaml;version=draft-07\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, validationErrs := unmarshalSchemaUnion(t, tt.doc)
			assert.Empty(t, validationErrs, "should have no unmarshal errors")
			require.True(t, u.IsMultiFormat(), "should be a multi format schema")

			def := u.GetMultiFormat().GetSchema()
			require.NotNil(t, def, "schema definition should be set")
			assert.Nil(t, def.GetRaw(), "raw should not be set for JSON Schema based formats")
			require.NotNil(t, def.GetSchema(), "typed schema should be set")
			assert.Equal(t, []string{"string"}, def.GetSchema().GetSchema().GetType(), "typed schema should be parsed")
		})
	}
}

func TestMultiFormatSchema_Unmarshal_ForeignFormatsPreserved_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		doc         string
		expectedRef references.Reference
	}{
		{
			name: "structured avro",
			doc: `schemaFormat: application/vnd.apache.avro;version=1.9.0
schema:
  type: record
  name: User
  namespace: com.company
  fields:
    - name: displayName
      type: string
    - name: age
      type: int
`,
		},
		{
			name: "protobuf string",
			doc: `schemaFormat: application/vnd.google.protobuf;version=3
schema: |
  message Point {
    required int32 x = 1;
  }
`,
		},
		{
			name: "avro resource reference from the specification",
			doc: `schemaFormat: application/vnd.apache.avro+yaml;version=1.9.0
schema:
  $ref: path/to/user-create.avsc#/UserCreate
`,
			expectedRef: "path/to/user-create.avsc#/UserCreate",
		},
		{
			name: "avro with nested $ref is not a reference",
			doc: `schemaFormat: application/vnd.apache.avro;version=1.9.0
schema:
  type: record
  name: User
  fields:
    - name: address
      type:
        $ref: ./address.avsc
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, validationErrs := unmarshalSchemaUnion(t, tt.doc)
			assert.Empty(t, validationErrs, "should have no unmarshal errors")
			require.True(t, u.IsMultiFormat(), "should be a multi format schema")

			def := u.GetMultiFormat().GetSchema()
			require.NotNil(t, def, "schema definition should be set")
			assert.Nil(t, def.GetSchema(), "typed schema should not be set for foreign formats")
			assert.NotNil(t, def.GetRaw(), "raw value should be preserved")
			assert.Equal(t, tt.expectedRef, def.GetReference(), "reference should match")

			assert.Equal(t, tt.doc, marshalSchemaUnion(t, u), "unmodified document should round trip unchanged")
		})
	}
}

func TestSchemaUnion_Mutate_RoundTrip_Success(t *testing.T) {
	t.Parallel()

	doc := `# payload schema
type: object
required:
  - id
properties:
  id:
    type: string
    description: The id
  tags:
    type: array
    items:
      - type: string
      - type: integer
    additionalItems: false
dependencies:
  id:
    - tags
  tags:
    required: [id]
x-custom: kept
`

	u, validationErrs := unmarshalSchemaUnion(t, doc)
	require.Empty(t, validationErrs, "should have no unmarshal errors")

	schema := u.GetSchema().GetSchema()
	require.NotNil(t, schema, "schema should be set")

	tags, ok := schema.Properties.Get("tags")
	require.True(t, ok, "tags property should exist")
	require.True(t, tags.GetSchema().Items.IsList(), "items should be a list")
	assert.Len(t, tags.GetSchema().Items.Schemas, 2, "items should have two schemas")

	idDep, ok := schema.Dependencies.Get("id")
	require.True(t, ok, "id dependency should exist")
	assert.Equal(t, []string{"tags"}, idDep.Properties, "property dependency should be parsed")
	tagsDep, ok := schema.Dependencies.Get("tags")
	require.True(t, ok, "tags dependency should exist")
	assert.Equal(t, []string{"id"}, tagsDep.Schema.GetSchema().Required, "schema dependency should be parsed")

	id, ok := schema.Properties.Get("id")
	require.True(t, ok, "id property should exist")
	id.GetSchema().Description = pointer.From("The unique id")
	tags.GetSchema().Items = asyncapi.NewItemsFromSchema(asyncapi.NewJSONSchemaFromSchema(&asyncapi.Schema{Type: asyncapi.NewTypeFromString("string")}))

	expected := `# payload schema
type: object
required:
  - id
properties:
  id:
    type: string
    description: The unique id
  tags:
    type: array
    items:
      type: string
    additionalItems: false
dependencies:
  id:
    - tags
  tags:
    required: [id]
x-custom: kept
`
	assert.Equal(t, expected, marshalSchemaUnion(t, u), "mutations should sync while preserving the rest of the document")
}

func TestSchemaUnion_Mutate_SwitchShape_Success(t *testing.T) {
	t.Parallel()

	u, _ := unmarshalSchemaUnion(t, "type: string\n")

	raw := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte("type: record\nname: User\n"), raw), "raw yaml should parse")

	u.Schema = nil
	u.MultiFormat = asyncapi.NewMultiFormatSchema("application/vnd.apache.avro;version=1.9.0", asyncapi.NewSchemaDefinitionFromRaw(raw.Content[0]))

	out := marshalSchemaUnion(t, u)
	assert.Equal(t, "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n  name: User\n", out, "should marshal the new multi format schema")

	reparsed, validationErrs := unmarshalSchemaUnion(t, out)
	assert.Empty(t, validationErrs, "reparsing should have no errors")
	assert.True(t, reparsed.IsMultiFormat(), "reparsed union should be a multi format schema")
}

func TestSchemaUnion_GetSchemaFormat_Success(t *testing.T) {
	t.Parallel()

	native := asyncapi.NewSchemaUnionFromSchema(asyncapi.NewJSONSchemaFromBool(true))
	assert.Equal(t, "application/vnd.aai.asyncapi+json;version=3.0.0", native.GetSchemaFormat("3.0.0"), "native schemas should use the document version")

	avro := asyncapi.NewSchemaUnionFromMultiFormat(asyncapi.NewMultiFormatSchema("application/vnd.apache.avro;version=1.9.0", nil))
	assert.Equal(t, "application/vnd.apache.avro;version=1.9.0", avro.GetSchemaFormat("3.0.0"), "multi format schemas should use their declared format")
}

func TestMultiFormatSchema_Getters_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *asyncapi.MultiFormatSchema

	assert.Empty(t, m.GetSchemaFormat(), "nil multi format schema should have no schema format")
	assert.Nil(t, m.GetSchema(), "nil multi format schema should have no schema")
	assert.Equal(t, 0, m.GetExtensions().Len(), "nil multi format schema should have empty extensions")
}

func TestMultiFormatSchema_Getters_Populated_Success(t *testing.T) {
	t.Parallel()

	u, validationErrs := unmarshalSchemaUnion(t, "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\nx-owner: team\n")
	require.Empty(t, validationErrs, "should have no unmarshal errors")

	m := u.GetMultiFormat()
	assert.Equal(t, "application/vnd.apache.avro;version=1.9.0", m.GetSchemaFormat(), "schema format should match")
	require.NotNil(t, m.GetSchema(), "schema should be set")
	assert.Equal(t, 3, m.GetSchema().GetRootNode().Line, "schema root node line should match")
	assert.Equal(t, 1, m.GetExtensions().Len(), "extensions should be parsed")
	assert.True(t, m.GetExtensions().Has("x-owner"), "extension should be present")
}

func TestSchemaDefinition_Getters_Nil_Success(t *testing.T) {
	t.Parallel()

	var d *asyncapi.SchemaDefinition

	assert.Nil(t, d.GetCore(), "nil definition should have no core")
	assert.Nil(t, d.GetRootNode(), "nil definition should have no root node")
	assert.Nil(t, d.GetSchema(), "nil definition should have no schema")
	assert.Nil(t, d.GetRaw(), "nil definition should have no raw value")
	assert.False(t, d.IsReference(), "nil definition should not be a reference")
	assert.Equal(t, references.Reference(""), d.GetReference(), "nil definition should have an empty reference")
	assert.NotPanics(t, func() { d.SetCore(&core.SchemaDefinition{}) }, "setting the core of a nil definition should be a no-op")
}

func TestSchemaDefinition_GetReference_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	refMapping := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("$ref"),
		yml.CreateStringNode("./user.avsc"),
	})

	tests := []struct {
		name                string
		definition          *asyncapi.SchemaDefinition
		expectedRef         references.Reference
		expectedIsReference bool
	}{
		{
			name:                "typed reference object",
			definition:          asyncapi.NewSchemaDefinitionFromSchema(asyncapi.NewJSONSchemaFromReference("#/components/schemas/User")),
			expectedRef:         "#/components/schemas/User",
			expectedIsReference: true,
		},
		{
			name:       "typed schema",
			definition: asyncapi.NewSchemaDefinitionFromSchema(asyncapi.NewJSONSchemaFromBool(true)),
		},
		{
			name:       "no raw value",
			definition: asyncapi.NewSchemaDefinitionFromRaw(nil),
		},
		{
			name:       "raw string",
			definition: asyncapi.NewSchemaDefinitionFromRaw(yml.CreateStringNode("message Point {}")),
		},
		{
			name: "raw mapping with non scalar $ref",
			definition: asyncapi.NewSchemaDefinitionFromRaw(yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("$ref"),
				yml.CreateMapNode(ctx, nil),
			})),
		},
		{
			name: "raw mapping without $ref",
			definition: asyncapi.NewSchemaDefinitionFromRaw(yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("type"),
				yml.CreateStringNode("record"),
			})),
		},
		{
			name:                "raw reference object",
			definition:          asyncapi.NewSchemaDefinitionFromRaw(refMapping),
			expectedRef:         "./user.avsc",
			expectedIsReference: true,
		},
		{
			name:                "raw alias to reference object",
			definition:          asyncapi.NewSchemaDefinitionFromRaw(&yaml.Node{Kind: yaml.AliasNode, Alias: refMapping}),
			expectedRef:         "./user.avsc",
			expectedIsReference: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedRef, tt.definition.GetReference(), "reference should match")
			assert.Equal(t, tt.expectedIsReference, tt.definition.IsReference(), "IsReference should match")
		})
	}
}

func TestSchemaDefinition_SetCore_Success(t *testing.T) {
	t.Parallel()

	raw := yml.CreateStringNode("message Point {}")
	d := asyncapi.NewSchemaDefinitionFromRaw(nil)

	d.SetCore(&core.SchemaDefinition{Raw: raw})
	assert.Same(t, raw, d.GetCore().Raw, "core should be set")

	d.SetCore(nil)
	assert.Same(t, raw, d.GetCore().Raw, "setting a nil core should be a no-op")
}

func TestSchemaDefinition_Populate_Success(t *testing.T) {
	t.Parallel()

	u, validationErrs := unmarshalSchemaUnion(t, "schemaFormat: application/schema+json;version=draft-07\nschema:\n  type: string\n")
	require.Empty(t, validationErrs, "should have no unmarshal errors")
	parsed := u.GetMultiFormat().GetSchema().GetCore()

	d := asyncapi.NewSchemaDefinitionFromRaw(yml.CreateStringNode("stale"))
	require.NoError(t, d.Populate(*parsed), "populating from a core value should succeed")

	assert.Nil(t, d.GetRaw(), "raw value should be replaced")
	assert.Equal(t, []string{"string"}, d.GetSchema().GetSchema().GetType(), "typed schema should be populated")
	assert.Same(t, parsed.GetRootNode(), d.GetRootNode(), "root node should be taken from the core")
}

func TestSchemaDefinition_Populate_NilCore_Success(t *testing.T) {
	t.Parallel()

	raw := yml.CreateStringNode("message Point {}")
	d := asyncapi.NewSchemaDefinitionFromRaw(raw)
	require.NoError(t, d.Populate((*core.SchemaDefinition)(nil)), "populating from a nil core should succeed")

	assert.Same(t, raw, d.GetRaw(), "definition should be unchanged")
}

func TestSchemaDefinition_Populate_Error(t *testing.T) {
	t.Parallel()

	d := &asyncapi.SchemaDefinition{}
	err := d.Populate("schema")
	require.EqualError(t, err, "expected core.SchemaDefinition, got string", "populating from the wrong type should fail")
}

func TestSchemaUnion_Getters_Nil_Success(t *testing.T) {
	t.Parallel()

	var u *asyncapi.SchemaUnion

	assert.Nil(t, u.GetCore(), "nil union should have no core")
	assert.Nil(t, u.GetRootNode(), "nil union should have no root node")
	assert.False(t, u.IsSchema(), "nil union should not be a schema")
	assert.Nil(t, u.GetSchema(), "nil union should have no schema")
	assert.False(t, u.IsMultiFormat(), "nil union should not be a multi format schema")
	assert.Nil(t, u.GetMultiFormat(), "nil union should have no multi format schema")
	assert.False(t, u.IsReference(), "nil union should not be a reference")
	assert.Equal(t, references.Reference(""), u.GetRef(), "nil union should have an empty reference")
	assert.Equal(t, "application/vnd.aai.asyncapi+json;version=3.1.0", u.GetSchemaFormat("3.1.0"), "nil union should use the native format")
	assert.NotPanics(t, func() { u.SetCore(&core.SchemaUnion{}) }, "setting the core of a nil union should be a no-op")
}

func TestSchemaUnion_SetCore_Success(t *testing.T) {
	t.Parallel()

	root := yml.CreateStringNode("root")
	c := &core.SchemaUnion{}
	c.SetRootNode(root)

	u := asyncapi.NewSchemaUnionFromSchema(asyncapi.NewJSONSchemaFromBool(true))
	u.SetCore(c)
	assert.Same(t, root, u.GetRootNode(), "core should be set")

	u.SetCore(nil)
	assert.Same(t, root, u.GetRootNode(), "setting a nil core should be a no-op")
}

func TestSchemaUnion_Populate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		doc                  string
		expectedIsSchema     bool
		expectedIsMulti      bool
		expectedSchemaFormat string
	}{
		{
			name:                 "native schema",
			doc:                  "type: string\n",
			expectedIsSchema:     true,
			expectedSchemaFormat: "application/vnd.aai.asyncapi+json;version=3.0.0",
		},
		{
			name:                 "multi format schema",
			doc:                  "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n",
			expectedIsMulti:      true,
			expectedSchemaFormat: "application/vnd.apache.avro;version=1.9.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, validationErrs := unmarshalSchemaUnion(t, tt.doc)
			require.Empty(t, validationErrs, "should have no unmarshal errors")

			u := &asyncapi.SchemaUnion{
				Schema:      asyncapi.NewJSONSchemaFromBool(true),
				MultiFormat: asyncapi.NewMultiFormatSchema("stale", nil),
			}
			require.NoError(t, u.Populate(*parsed.GetCore()), "populating from a core value should succeed")

			assert.Equal(t, tt.expectedIsSchema, u.IsSchema(), "IsSchema should match")
			assert.Equal(t, tt.expectedIsMulti, u.IsMultiFormat(), "IsMultiFormat should match")
			assert.Equal(t, tt.expectedSchemaFormat, u.GetSchemaFormat("3.0.0"), "schema format should match")
			assert.Same(t, parsed.GetRootNode(), u.GetRootNode(), "root node should be taken from the core")
		})
	}
}

func TestSchemaUnion_Populate_EmptyCore_Success(t *testing.T) {
	t.Parallel()

	u := asyncapi.NewSchemaUnionFromSchema(asyncapi.NewJSONSchemaFromBool(true))
	require.NoError(t, u.Populate(&core.SchemaUnion{}), "populating from an empty core should succeed")

	assert.Nil(t, u.GetSchema(), "schema should be cleared")
	assert.Nil(t, u.GetMultiFormat(), "multi format schema should be cleared")
}

func TestSchemaUnion_Populate_NilCore_Success(t *testing.T) {
	t.Parallel()

	schema := asyncapi.NewJSONSchemaFromBool(true)
	u := asyncapi.NewSchemaUnionFromSchema(schema)
	require.NoError(t, u.Populate((*core.SchemaUnion)(nil)), "populating from a nil core should succeed")

	assert.Same(t, schema, u.GetSchema(), "union should be unchanged")
}

func TestSchemaUnion_Populate_Error(t *testing.T) {
	t.Parallel()

	u := &asyncapi.SchemaUnion{}
	err := u.Populate(true)
	require.EqualError(t, err, "expected core.SchemaUnion, got bool", "populating from the wrong type should fail")
}
