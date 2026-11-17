package asyncapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
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
