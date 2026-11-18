package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/pointer"
	coreValues "github.com/speakeasy-api/openapi/values/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func schemaTypes(schemas []*asyncapi.JSONSchema) [][]string {
	types := make([][]string, 0, len(schemas))
	for _, s := range schemas {
		types = append(types, s.GetSchema().GetType())
	}
	return types
}

func TestNewItems_Success(t *testing.T) {
	t.Parallel()

	single := asyncapi.NewJSONSchemaFromBool(true)

	tests := []struct {
		name            string
		items           *asyncapi.Items
		expectedIsList  bool
		expectedSchema  *asyncapi.JSONSchema
		expectedSchemas []*asyncapi.JSONSchema
	}{
		{
			name:           "single schema",
			items:          asyncapi.NewItemsFromSchema(single),
			expectedSchema: single,
		},
		{
			name:            "empty list of schemas",
			items:           asyncapi.NewItemsFromSchemas(),
			expectedIsList:  true,
			expectedSchemas: []*asyncapi.JSONSchema{},
		},
		{
			name:            "list of schemas",
			items:           asyncapi.NewItemsFromSchemas(single, single),
			expectedIsList:  true,
			expectedSchemas: []*asyncapi.JSONSchema{single, single},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedIsList, tt.items.IsList(), "IsList should match")
			assert.Equal(t, tt.expectedSchema, tt.items.Schema, "Schema should match")
			assert.Equal(t, tt.expectedSchemas, tt.items.Schemas, "Schemas should match")
		})
	}
}

func TestItems_Getters_Nil_Success(t *testing.T) {
	t.Parallel()

	var items *asyncapi.Items

	assert.False(t, items.IsList(), "nil items should not be a list")
	assert.Nil(t, items.GetCore(), "nil items should have no core")
	assert.Nil(t, items.GetRootNode(), "nil items should have no root node")
}

func TestItems_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		doc                  string
		expectedIsList       bool
		expectedSchemaType   []string
		expectedSchemasTypes [][]string
		expectedLine         int
	}{
		{
			name:                 "single schema",
			doc:                  "type: array\nitems:\n  type: string\n",
			expectedSchemaType:   []string{"string"},
			expectedSchemasTypes: [][]string{},
			expectedLine:         3,
		},
		{
			name:                 "list of schemas",
			doc:                  "type: array\nitems:\n  - type: string\n  - true\n",
			expectedIsList:       true,
			expectedSchemasTypes: [][]string{{"string"}, nil},
			expectedLine:         3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, validationErrs := unmarshalSchemaUnion(t, tt.doc)
			require.Empty(t, validationErrs, "should have no unmarshal errors")

			items := u.GetSchema().GetSchema().Items
			require.NotNil(t, items, "items should be set")
			assert.Equal(t, tt.expectedIsList, items.IsList(), "IsList should match")
			assert.Equal(t, tt.expectedIsList, items.GetCore().IsList, "core IsList should match")
			assert.Equal(t, tt.expectedSchemaType, items.Schema.GetSchema().GetType(), "single schema type should match")
			assert.Equal(t, tt.expectedSchemasTypes, schemaTypes(items.Schemas), "list schema types should match")
			require.NotNil(t, items.GetRootNode(), "root node should be set")
			assert.Equal(t, tt.expectedLine, items.GetRootNode().Line, "root node line should match")
		})
	}
}

func TestItems_Populate_Success(t *testing.T) {
	t.Parallel()

	u, validationErrs := unmarshalSchemaUnion(t, "type: array\nitems:\n  - type: string\n  - false\n")
	require.Empty(t, validationErrs, "should have no unmarshal errors")
	parsed := u.GetSchema().GetSchema().Items.GetCore()

	items := asyncapi.NewItemsFromSchema(asyncapi.NewJSONSchemaFromBool(true))
	require.NoError(t, items.Populate(*parsed), "populating from a core value should succeed")

	assert.True(t, items.IsList(), "items should be a list")
	assert.Nil(t, items.Schema, "previous single schema should be cleared")
	assert.Equal(t, [][]string{{"string"}, nil}, schemaTypes(items.Schemas), "schema types should match")
	assert.Equal(t, pointer.From(false), items.Schemas[1].GetBool(), "boolean schema should be populated")
	assert.Same(t, parsed.GetRootNode(), items.GetRootNode(), "root node should be taken from the core")
}

func TestItems_Populate_NilListEntry_Success(t *testing.T) {
	t.Parallel()

	items := &asyncapi.Items{}
	err := items.Populate(&core.Items{IsList: true, Schemas: []*coreValues.EitherValue[core.Schema, bool]{nil}})
	require.NoError(t, err, "populate should succeed")

	assert.Equal(t, []*asyncapi.JSONSchema{{}}, items.Schemas, "nil core entries should populate empty schemas")
}

func TestItems_Populate_NilCore_Success(t *testing.T) {
	t.Parallel()

	single := asyncapi.NewJSONSchemaFromBool(true)
	items := asyncapi.NewItemsFromSchema(single)
	require.NoError(t, items.Populate((*core.Items)(nil)), "populating from a nil core should succeed")

	assert.Same(t, single, items.Schema, "items should be unchanged")
}

func TestItems_Populate_Error(t *testing.T) {
	t.Parallel()

	items := &asyncapi.Items{}
	err := items.Populate("items")
	require.EqualError(t, err, "expected core.Items, got string", "populating from the wrong type should fail")
}

func TestNewDependency_Success(t *testing.T) {
	t.Parallel()

	schema := asyncapi.NewJSONSchemaFromBool(true)

	tests := []struct {
		name               string
		dependency         *asyncapi.Dependency
		expectedIsList     bool
		expectedSchema     *asyncapi.JSONSchema
		expectedProperties []string
	}{
		{
			name:           "schema dependency",
			dependency:     asyncapi.NewDependencyFromSchema(schema),
			expectedSchema: schema,
		},
		{
			name:               "empty property dependency",
			dependency:         asyncapi.NewDependencyFromProperties(),
			expectedIsList:     true,
			expectedProperties: []string{},
		},
		{
			name:               "property dependency",
			dependency:         asyncapi.NewDependencyFromProperties("a", "b"),
			expectedIsList:     true,
			expectedProperties: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedIsList, tt.dependency.IsList(), "IsList should match")
			assert.Equal(t, tt.expectedSchema, tt.dependency.Schema, "Schema should match")
			assert.Equal(t, tt.expectedProperties, tt.dependency.Properties, "Properties should match")
		})
	}
}

func TestDependency_Getters_Nil_Success(t *testing.T) {
	t.Parallel()

	var dependency *asyncapi.Dependency

	assert.False(t, dependency.IsList(), "nil dependency should not be a list")
	assert.Nil(t, dependency.GetCore(), "nil dependency should have no core")
	assert.Nil(t, dependency.GetRootNode(), "nil dependency should have no root node")
}

func TestDependency_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		key                string
		expectedIsList     bool
		expectedProperties []string
		expectedRequired   []string
		expectedLine       int
	}{
		{
			name:               "property dependency",
			key:                "a",
			expectedIsList:     true,
			expectedProperties: []string{"b", "c"},
			expectedLine:       3,
		},
		{
			name:             "schema dependency",
			key:              "d",
			expectedRequired: []string{"e"},
			expectedLine:     5,
		},
	}

	doc := "type: object\ndependencies:\n  a: [b, c]\n  d:\n    required: [e]\n"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, validationErrs := unmarshalSchemaUnion(t, doc)
			require.Empty(t, validationErrs, "should have no unmarshal errors")

			dependency, ok := u.GetSchema().GetSchema().Dependencies.Get(tt.key)
			require.True(t, ok, "dependency should exist")
			assert.Equal(t, tt.expectedIsList, dependency.IsList(), "IsList should match")
			assert.Equal(t, tt.expectedIsList, dependency.GetCore().IsList, "core IsList should match")
			assert.Equal(t, tt.expectedProperties, dependency.Properties, "properties should match")
			assert.Equal(t, tt.expectedRequired, dependency.Schema.GetSchema().GetRequired(), "schema should match")
			require.NotNil(t, dependency.GetRootNode(), "root node should be set")
			assert.Equal(t, tt.expectedLine, dependency.GetRootNode().Line, "root node line should match")
		})
	}
}

func TestDependency_Populate_Success(t *testing.T) {
	t.Parallel()

	u, validationErrs := unmarshalSchemaUnion(t, "type: object\ndependencies:\n  a: [b, c]\n")
	require.Empty(t, validationErrs, "should have no unmarshal errors")
	parsedDependency, ok := u.GetSchema().GetSchema().Dependencies.Get("a")
	require.True(t, ok, "dependency should exist")
	parsed := parsedDependency.GetCore()

	dependency := asyncapi.NewDependencyFromSchema(asyncapi.NewJSONSchemaFromBool(true))
	require.NoError(t, dependency.Populate(*parsed), "populating from a core value should succeed")

	assert.True(t, dependency.IsList(), "dependency should be a list")
	assert.Nil(t, dependency.Schema, "previous schema should be cleared")
	assert.Equal(t, []string{"b", "c"}, dependency.Properties, "properties should match")
	assert.Same(t, parsed.GetRootNode(), dependency.GetRootNode(), "root node should be taken from the core")
}

func TestDependency_Populate_NilCore_Success(t *testing.T) {
	t.Parallel()

	dependency := asyncapi.NewDependencyFromProperties("a")
	require.NoError(t, dependency.Populate((*core.Dependency)(nil)), "populating from a nil core should succeed")

	assert.Equal(t, []string{"a"}, dependency.Properties, "dependency should be unchanged")
}

func TestDependency_Populate_Error(t *testing.T) {
	t.Parallel()

	dependency := &asyncapi.Dependency{}
	err := dependency.Populate(42)
	require.EqualError(t, err, "expected core.Dependency, got int", "populating from the wrong type should fail")
}
