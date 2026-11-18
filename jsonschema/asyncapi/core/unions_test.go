package core_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func parseYAML(t *testing.T, yml string) *yaml.Node {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node), "yaml should parse")
	return &node
}

func encodeYAML(t *testing.T, node *yaml.Node) string {
	t.Helper()

	out, err := yaml.Marshal(node)
	require.NoError(t, err, "yaml should encode")
	return string(out)
}

func TestItems_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		yml               string
		expectedIsList    bool
		expectedHasSchema bool
		expectedSchemas   int
	}{
		{
			name:              "single schema",
			yml:               "type: string\n",
			expectedHasSchema: true,
		},
		{
			name:              "single boolean schema",
			yml:               "false\n",
			expectedHasSchema: true,
		},
		{
			name:            "list of schemas",
			yml:             "- true\n- type: string\n",
			expectedIsList:  true,
			expectedSchemas: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var items core.Items
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "items", parseYAML(t, tt.yml), &items)
			require.NoError(t, err, "unmarshal should succeed")
			assert.Empty(t, validationErrs, "should have no validation errors")

			assert.Equal(t, tt.expectedIsList, items.IsList, "IsList should match")
			assert.Equal(t, tt.expectedHasSchema, items.Schema != nil, "single schema presence should match")
			assert.Len(t, items.Schemas, tt.expectedSchemas, "list length should match")
			assert.True(t, items.GetValid(), "items should be valid")
			assert.NotNil(t, items.GetRootNode(), "root node should be set")
		})
	}
}

func TestItems_Unmarshal_Error(t *testing.T) {
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
			name:        "alias without target",
			node:        &yaml.Node{Kind: yaml.AliasNode},
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

			var items core.Items
			validationErrs, err := items.Unmarshal(t.Context(), "items", tt.node)
			require.EqualError(t, err, tt.expectedErr, "unmarshal should fail")
			assert.Nil(t, validationErrs, "should have no validation errors")
		})
	}
}

func TestItems_SyncChanges_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yml            string
		model          *asyncapi.Items
		expectedYAML   string
		expectedIsList bool
	}{
		{
			name:           "list replaces single schema",
			yml:            "type: string\n",
			model:          asyncapi.NewItemsFromSchemas(asyncapi.NewJSONSchemaFromBool(true)),
			expectedYAML:   "- true\n",
			expectedIsList: true,
		},
		{
			name:         "single schema replaces list",
			yml:          "- true\n",
			model:        asyncapi.NewItemsFromSchema(asyncapi.NewJSONSchemaFromBool(false)),
			expectedYAML: "false\n",
		},
		{
			name:           "list is updated in place",
			yml:            "- true\n",
			model:          asyncapi.NewItemsFromSchemas(asyncapi.NewJSONSchemaFromBool(false), asyncapi.NewJSONSchemaFromBool(true)),
			expectedYAML:   "- false\n- true\n",
			expectedIsList: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var items core.Items
			_, err := marshaller.UnmarshalCore(ctx, "items", parseYAML(t, tt.yml), &items)
			require.NoError(t, err, "unmarshal should succeed")

			node, err := items.SyncChanges(ctx, tt.model, items.GetRootNode())
			require.NoError(t, err, "sync should succeed")

			assert.Equal(t, tt.expectedYAML, encodeYAML(t, node), "synced yaml should match")
			assert.Equal(t, tt.expectedIsList, items.IsList, "IsList should match")
			assert.Same(t, node, items.GetRootNode(), "root node should be the synced node")
		})
	}
}

func TestItems_SyncChanges_Empty_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var items core.Items
	_, err := marshaller.UnmarshalCore(ctx, "items", parseYAML(t, "type: string\n"), &items)
	require.NoError(t, err, "unmarshal should succeed")

	node, err := items.SyncChanges(ctx, &asyncapi.Items{}, items.GetRootNode())
	require.NoError(t, err, "sync should succeed")

	assert.Nil(t, node, "no node should be produced")
	assert.Nil(t, items.Schema, "single schema should be cleared")
	assert.Nil(t, items.Schemas, "list should be cleared")
	assert.False(t, items.IsList, "IsList should be cleared")
	assert.Nil(t, items.GetRootNode(), "root node should be cleared")
}

func TestItems_SyncChanges_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		model       any
		expectedErr string
	}{
		{
			name:        "nil model",
			model:       (*asyncapi.Items)(nil),
			expectedErr: "Items.SyncChanges expected a non-nil model",
		},
		{
			name:        "non struct model",
			model:       "items",
			expectedErr: "Items.SyncChanges expected a struct, got `string`",
		},
		{
			name:        "list of the wrong type",
			model:       struct{ Schemas []int }{Schemas: []int{1}},
			expectedErr: "SyncValue expected target to be int, got struct",
		},
		{
			name:        "single schema of the wrong type",
			model:       struct{ Schema *int }{Schema: pointer.From(1)},
			expectedErr: "SyncValue expected target to be int, got struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var items core.Items
			node, err := items.SyncChanges(t.Context(), tt.model, nil)
			require.EqualError(t, err, tt.expectedErr, "sync should fail")
			assert.Nil(t, node, "no node should be produced")
		})
	}
}

func TestDependency_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		yml                string
		expectedIsList     bool
		expectedHasSchema  bool
		expectedProperties []string
	}{
		{
			name:               "schema dependency",
			yml:                "required: [a]\n",
			expectedHasSchema:  true,
			expectedProperties: []string{},
		},
		{
			name:               "property dependency",
			yml:                "- a\n- b\n",
			expectedIsList:     true,
			expectedProperties: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dependency core.Dependency
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "dependency", parseYAML(t, tt.yml), &dependency)
			require.NoError(t, err, "unmarshal should succeed")
			assert.Empty(t, validationErrs, "should have no validation errors")

			properties := make([]string, 0, len(dependency.Properties))
			for _, p := range dependency.Properties {
				properties = append(properties, p.Value)
			}

			assert.Equal(t, tt.expectedIsList, dependency.IsList, "IsList should match")
			assert.Equal(t, tt.expectedHasSchema, dependency.Schema != nil, "schema presence should match")
			assert.Equal(t, tt.expectedProperties, properties, "properties should match")
			assert.True(t, dependency.GetValid(), "dependency should be valid")
			assert.NotNil(t, dependency.GetRootNode(), "root node should be set")
		})
	}
}

func TestDependency_Unmarshal_Error(t *testing.T) {
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

			var dependency core.Dependency
			validationErrs, err := dependency.Unmarshal(t.Context(), "dependency", tt.node)
			require.EqualError(t, err, tt.expectedErr, "unmarshal should fail")
			assert.Nil(t, validationErrs, "should have no validation errors")
		})
	}
}

func TestDependency_SyncChanges_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yml            string
		model          *asyncapi.Dependency
		expectedYAML   string
		expectedIsList bool
	}{
		{
			name:           "properties replace schema",
			yml:            "required: [a]\n",
			model:          asyncapi.NewDependencyFromProperties("b"),
			expectedYAML:   "- b\n",
			expectedIsList: true,
		},
		{
			name:         "schema replaces properties",
			yml:          "- a\n",
			model:        asyncapi.NewDependencyFromSchema(asyncapi.NewJSONSchemaFromBool(true)),
			expectedYAML: "true\n",
		},
		{
			name:           "properties are updated in place",
			yml:            "- a\n",
			model:          asyncapi.NewDependencyFromProperties("a", "b"),
			expectedYAML:   "- a\n- b\n",
			expectedIsList: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var dependency core.Dependency
			_, err := marshaller.UnmarshalCore(ctx, "dependency", parseYAML(t, tt.yml), &dependency)
			require.NoError(t, err, "unmarshal should succeed")

			node, err := dependency.SyncChanges(ctx, tt.model, dependency.GetRootNode())
			require.NoError(t, err, "sync should succeed")

			assert.Equal(t, tt.expectedYAML, encodeYAML(t, node), "synced yaml should match")
			assert.Equal(t, tt.expectedIsList, dependency.IsList, "IsList should match")
			assert.Same(t, node, dependency.GetRootNode(), "root node should be the synced node")
		})
	}
}

func TestDependency_SyncChanges_Empty_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var dependency core.Dependency
	_, err := marshaller.UnmarshalCore(ctx, "dependency", parseYAML(t, "- a\n"), &dependency)
	require.NoError(t, err, "unmarshal should succeed")

	node, err := dependency.SyncChanges(ctx, &asyncapi.Dependency{}, dependency.GetRootNode())
	require.NoError(t, err, "sync should succeed")

	assert.Nil(t, node, "no node should be produced")
	assert.Nil(t, dependency.Schema, "schema should be cleared")
	assert.Nil(t, dependency.Properties, "properties should be cleared")
	assert.False(t, dependency.IsList, "IsList should be cleared")
	assert.Nil(t, dependency.GetRootNode(), "root node should be cleared")
}

func TestDependency_SyncChanges_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		model       any
		expectedErr string
	}{
		{
			name:        "nil model",
			model:       (*asyncapi.Dependency)(nil),
			expectedErr: "Dependency.SyncChanges expected a non-nil model",
		},
		{
			name:        "non struct model",
			model:       []string{"a"},
			expectedErr: "Dependency.SyncChanges expected a struct, got `slice`",
		},
		{
			name:        "properties of the wrong type",
			model:       struct{ Properties []int }{Properties: []int{1}},
			expectedErr: "SyncValue expected target to be int, got string",
		},
		{
			name:        "schema of the wrong type",
			model:       struct{ Schema *int }{Schema: pointer.From(1)},
			expectedErr: "SyncValue expected target to be int, got struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dependency core.Dependency
			node, err := dependency.SyncChanges(t.Context(), tt.model, nil)
			require.EqualError(t, err, tt.expectedErr, "sync should fail")
			assert.Nil(t, node, "no node should be produced")
		})
	}
}
