package core_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func unmarshalSchemaDefinition(t *testing.T, doc string) *core.SchemaDefinition {
	t.Helper()

	var m core.MultiFormatSchema
	_, err := marshaller.UnmarshalCore(t.Context(), "multiFormatSchema", parseYAML(t, doc), &m)
	require.NoError(t, err, "unmarshal should succeed")
	require.NotNil(t, m.Schema.Value, "schema definition should be set")
	return m.Schema.Value
}

func TestIsMultiFormatNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		expected bool
	}{
		{
			name:     "multi format schema",
			yml:      "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema: {}\n",
			expected: true,
		},
		{
			name:     "reference object with schemaFormat",
			yml:      "$ref: '#/components/schemas/User'\nschemaFormat: application/vnd.apache.avro;version=1.9.0\n",
			expected: false,
		},
		{
			name:     "native schema",
			yml:      "type: string\n",
			expected: false,
		},
		{
			name:     "boolean schema",
			yml:      "true\n",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := parseYAML(t, tt.yml).Content[0]
			assert.Equal(t, tt.expected, core.IsMultiFormatNode(t.Context(), node), "IsMultiFormatNode should match")
		})
	}
}

func TestIsMultiFormatNode_NilNode_Success(t *testing.T) {
	t.Parallel()

	assert.False(t, core.IsMultiFormatNode(t.Context(), nil), "nil node should not be a multi format schema")
}

func TestMultiFormatSchema_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yml            string
		expectedFormat string
		expectedTyped  bool
		expectedRaw    bool
	}{
		{
			name:           "json schema based format is typed",
			yml:            "schemaFormat: application/schema+json;version=draft-07\nschema:\n  type: string\n",
			expectedFormat: "application/schema+json;version=draft-07",
			expectedTyped:  true,
		},
		{
			name:           "foreign format is raw",
			yml:            "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n",
			expectedFormat: "application/vnd.apache.avro;version=1.9.0",
			expectedRaw:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var m core.MultiFormatSchema
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "multiFormatSchema", parseYAML(t, tt.yml), &m)
			require.NoError(t, err, "unmarshal should succeed")
			assert.Empty(t, validationErrs, "should have no validation errors")

			assert.Equal(t, tt.expectedFormat, m.SchemaFormat.Value, "schema format should match")
			require.NotNil(t, m.Schema.Value, "schema definition should be set")
			assert.Equal(t, tt.expectedTyped, m.Schema.Value.Schema != nil, "typed schema presence should match")
			assert.Equal(t, tt.expectedRaw, m.Schema.Value.Raw != nil, "raw value presence should match")
		})
	}
}

func TestSchemaDefinition_Unmarshal_DefaultFormat_Success(t *testing.T) {
	t.Parallel()

	var d core.SchemaDefinition
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "schema", parseYAML(t, "type: string\n"), &d)
	require.NoError(t, err, "unmarshal should succeed")
	assert.Empty(t, validationErrs, "should have no validation errors")

	require.NotNil(t, d.Schema, "without a declared format the definition should be a native schema")
	assert.True(t, d.Schema.IsLeft, "schema should be a schema object")
	assert.Nil(t, d.Raw, "raw value should not be set")
	assert.True(t, d.GetValid(), "definition should be valid")
}

func TestSchemaDefinition_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		node        *yaml.Node
		expectedErr string
	}{
		{
			name:        "nil node",
			node:        nil,
			expectedErr: "node is nil",
		},
		{
			name:        "empty document",
			node:        &yaml.Node{Kind: yaml.DocumentNode},
			expectedErr: "expected 1 node, got `0` at line `0`, column `0`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d core.SchemaDefinition
			validationErrs, err := d.Unmarshal(t.Context(), "schema", tt.node)
			require.EqualError(t, err, tt.expectedErr, "unmarshal should fail")
			assert.Nil(t, validationErrs, "should have no validation errors")
		})
	}
}

func TestSchemaDefinition_SyncChanges_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		doc           string
		model         *asyncapi.SchemaDefinition
		expectedYAML  string
		expectedTyped bool
	}{
		{
			name:          "typed schema replaces raw value",
			doc:           "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n",
			model:         asyncapi.NewSchemaDefinitionFromSchema(asyncapi.NewJSONSchemaFromBool(true)),
			expectedYAML:  "true\n",
			expectedTyped: true,
		},
		{
			name:          "typed schema is updated in place",
			doc:           "schemaFormat: application/schema+json;version=draft-07\nschema:\n  type: string\n",
			model:         asyncapi.NewSchemaDefinitionFromSchema(asyncapi.NewJSONSchemaFromSchema(&asyncapi.Schema{Type: asyncapi.NewTypeFromString("integer")})),
			expectedYAML:  "type: integer\n",
			expectedTyped: true,
		},
		{
			name:         "raw value replaces typed schema",
			doc:          "schemaFormat: application/schema+json;version=draft-07\nschema:\n  type: string\n",
			model:        asyncapi.NewSchemaDefinitionFromRaw(yml.CreateStringNode("message Point {}")),
			expectedYAML: "message Point {}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := unmarshalSchemaDefinition(t, tt.doc)

			node, err := d.SyncChanges(t.Context(), tt.model, d.GetRootNode())
			require.NoError(t, err, "sync should succeed")

			assert.Equal(t, tt.expectedYAML, encodeYAML(t, node), "synced yaml should match")
			assert.Equal(t, tt.expectedTyped, d.Schema != nil, "typed schema presence should match")
			assert.Equal(t, !tt.expectedTyped, d.Raw != nil, "raw value presence should match")
			assert.Same(t, node, d.GetRootNode(), "root node should be the synced node")
		})
	}
}

func TestSchemaDefinition_SyncChanges_Empty_Success(t *testing.T) {
	t.Parallel()

	d := unmarshalSchemaDefinition(t, "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n")

	node, err := d.SyncChanges(t.Context(), &asyncapi.SchemaDefinition{}, d.GetRootNode())
	require.NoError(t, err, "sync should succeed")

	assert.Nil(t, node, "no node should be produced")
	assert.Nil(t, d.Schema, "typed schema should be cleared")
	assert.Nil(t, d.Raw, "raw value should be cleared")
	assert.Nil(t, d.GetRootNode(), "root node should be cleared")
}

func TestSchemaDefinition_SyncChanges_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		model       any
		expectedErr string
	}{
		{
			name:        "nil model",
			model:       (*asyncapi.SchemaDefinition)(nil),
			expectedErr: "SchemaDefinition.SyncChanges expected a non-nil model",
		},
		{
			name:        "non struct model",
			model:       1,
			expectedErr: "SchemaDefinition.SyncChanges expected a struct, got `int`",
		},
		{
			name:        "typed schema of the wrong type",
			model:       struct{ Schema *int }{Schema: pointer.From(1)},
			expectedErr: "SyncValue expected target to be int, got struct",
		},
		{
			name:        "raw value of the wrong type",
			model:       struct{ Raw *int }{Raw: pointer.From(1)},
			expectedErr: "SchemaDefinition.SyncChanges expected Raw to be a *yaml.Node, got `*int`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d core.SchemaDefinition
			node, err := d.SyncChanges(t.Context(), tt.model, nil)
			require.EqualError(t, err, tt.expectedErr, "sync should fail")
			assert.Nil(t, node, "no node should be produced")
		})
	}
}

func TestSchemaUnion_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		node        *yaml.Node
		expectedErr string
	}{
		{
			name:        "nil node",
			node:        nil,
			expectedErr: "node is nil",
		},
		{
			name:        "empty document",
			node:        &yaml.Node{Kind: yaml.DocumentNode},
			expectedErr: "expected 1 node, got `0` at line `0`, column `0`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var u core.SchemaUnion
			validationErrs, err := u.Unmarshal(t.Context(), "schema", tt.node)
			require.EqualError(t, err, tt.expectedErr, "unmarshal should fail")
			assert.Nil(t, validationErrs, "should have no validation errors")
		})
	}
}

func TestSchemaUnion_SyncChanges_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		yml                 string
		model               *asyncapi.SchemaUnion
		expectedYAML        string
		expectedSchema      bool
		expectedMultiFormat bool
	}{
		{
			name:           "native schema replaces multi format schema",
			yml:            "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n",
			model:          asyncapi.NewSchemaUnionFromSchema(asyncapi.NewJSONSchemaFromSchema(&asyncapi.Schema{Type: asyncapi.NewTypeFromString("string")})),
			expectedYAML:   "type: string\n",
			expectedSchema: true,
		},
		{
			name:                "multi format schema replaces native schema",
			yml:                 "type: string\n",
			model:               asyncapi.NewSchemaUnionFromMultiFormat(asyncapi.NewMultiFormatSchema("application/vnd.google.protobuf;version=3", asyncapi.NewSchemaDefinitionFromRaw(yml.CreateStringNode("message Point {}")))),
			expectedYAML:        "schemaFormat: application/vnd.google.protobuf;version=3\nschema: message Point {}\n",
			expectedMultiFormat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var u core.SchemaUnion
			_, err := marshaller.UnmarshalCore(ctx, "schema", parseYAML(t, tt.yml), &u)
			require.NoError(t, err, "unmarshal should succeed")

			node, err := u.SyncChanges(ctx, tt.model, u.GetRootNode())
			require.NoError(t, err, "sync should succeed")

			assert.Equal(t, tt.expectedYAML, encodeYAML(t, node), "synced yaml should match")
			assert.Equal(t, tt.expectedSchema, u.Schema != nil, "native schema presence should match")
			assert.Equal(t, tt.expectedMultiFormat, u.MultiFormat != nil, "multi format schema presence should match")
			assert.Same(t, node, u.GetRootNode(), "root node should be the synced node")
		})
	}
}

func TestSchemaUnion_SyncChanges_Empty_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var u core.SchemaUnion
	_, err := marshaller.UnmarshalCore(ctx, "schema", parseYAML(t, "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema: {}\n"), &u)
	require.NoError(t, err, "unmarshal should succeed")

	node, err := u.SyncChanges(ctx, &asyncapi.SchemaUnion{}, u.GetRootNode())
	require.NoError(t, err, "sync should succeed")

	assert.Nil(t, node, "no node should be produced")
	assert.Nil(t, u.Schema, "native schema should be cleared")
	assert.Nil(t, u.MultiFormat, "multi format schema should be cleared")
	assert.Nil(t, u.GetRootNode(), "root node should be cleared")
}

func TestSchemaUnion_SyncChanges_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		model       any
		expectedErr string
	}{
		{
			name:        "nil model",
			model:       (*asyncapi.SchemaUnion)(nil),
			expectedErr: "SchemaUnion.SyncChanges expected a non-nil model",
		},
		{
			name:        "non struct model",
			model:       true,
			expectedErr: "SchemaUnion.SyncChanges expected a struct, got `bool`",
		},
		{
			name:        "native schema of the wrong type",
			model:       struct{ Schema *int }{Schema: pointer.From(1)},
			expectedErr: "SyncValue expected target to be int, got struct",
		},
		{
			name:        "multi format schema of the wrong type",
			model:       struct{ MultiFormat *int }{MultiFormat: pointer.From(1)},
			expectedErr: "SyncValue expected target to be int, got struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var u core.SchemaUnion
			node, err := u.SyncChanges(t.Context(), tt.model, nil)
			require.EqualError(t, err, tt.expectedErr, "sync should fail")
			assert.Nil(t, node, "no node should be produced")
		})
	}
}
