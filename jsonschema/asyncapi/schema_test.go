package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTypeFromArray_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		types    []string
		expected []string
	}{
		{
			name:     "no types creates an empty list",
			types:    nil,
			expected: []string{},
		},
		{
			name:     "multiple types are kept in order",
			types:    []string{"string", "null"},
			expected: []string{"string", "null"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typ := asyncapi.NewTypeFromArray(tt.types...)
			require.NotNil(t, typ.Left, "list form should be set")
			assert.Equal(t, tt.expected, *typ.Left, "types should match")
			assert.Nil(t, typ.Right, "single form should not be set")
		})
	}
}

func TestSchema_GetType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema   *asyncapi.Schema
		expected []string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "no type declared",
			schema:   &asyncapi.Schema{},
			expected: nil,
		},
		{
			name:     "single type",
			schema:   &asyncapi.Schema{Type: asyncapi.NewTypeFromString("string")},
			expected: []string{"string"},
		},
		{
			name:     "list of types",
			schema:   &asyncapi.Schema{Type: asyncapi.NewTypeFromArray("integer", "null")},
			expected: []string{"integer", "null"},
		},
		{
			name:     "empty list of types",
			schema:   &asyncapi.Schema{Type: asyncapi.NewTypeFromArray()},
			expected: []string{},
		},
		{
			name:     "type with neither form set",
			schema:   &asyncapi.Schema{Type: &asyncapi.Type{}},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.schema.GetType(), "types should match")
		})
	}
}

func TestSchema_Getters_Nil_Success(t *testing.T) {
	t.Parallel()

	var s *asyncapi.Schema

	assert.False(t, s.IsReference(), "nil schema should not be a reference")
	assert.Equal(t, references.Reference(""), s.GetRef(), "nil schema should have an empty reference")
	assert.Nil(t, s.GetRequired(), "nil schema should have no required properties")
	assert.Nil(t, s.GetProperties(), "nil schema should have no properties")
	assert.Empty(t, s.GetDiscriminator(), "nil schema should have no discriminator")
	assert.Nil(t, s.GetExternalDocs(), "nil schema should have no external docs")
	assert.False(t, s.GetDeprecated(), "nil schema should not be deprecated")
	assert.Equal(t, 0, s.GetExtensions().Len(), "nil schema should have empty extensions")
}

func TestSchema_Getters_Unset_Success(t *testing.T) {
	t.Parallel()

	s := &asyncapi.Schema{}

	assert.False(t, s.IsReference(), "schema without $ref should not be a reference")
	assert.Equal(t, references.Reference(""), s.GetRef(), "schema without $ref should have an empty reference")
	assert.Empty(t, s.GetDiscriminator(), "unset discriminator should be empty")
	assert.False(t, s.GetDeprecated(), "unset deprecated should be false")
	assert.Equal(t, 0, s.GetExtensions().Len(), "unset extensions should be empty")
}

func TestSchema_Getters_Populated_Success(t *testing.T) {
	t.Parallel()

	properties := sequencedmap.New(sequencedmap.NewElem("kind", asyncapi.NewJSONSchemaFromBool(true)))
	externalDocs := &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com")}
	exts := extensions.New(extensions.NewElem("x-owner", yml.CreateStringNode("team")))

	s := &asyncapi.Schema{
		Ref:           pointer.From(references.Reference("#/components/schemas/Pet")),
		Required:      []string{"kind"},
		Properties:    properties,
		Discriminator: pointer.From("kind"),
		ExternalDocs:  externalDocs,
		Deprecated:    pointer.From(true),
		Extensions:    exts,
	}

	assert.True(t, s.IsReference(), "schema with $ref should be a reference")
	assert.Equal(t, references.Reference("#/components/schemas/Pet"), s.GetRef(), "reference should match")
	assert.Equal(t, []string{"kind"}, s.GetRequired(), "required should match")
	assert.Same(t, properties, s.GetProperties(), "properties should match")
	assert.Equal(t, "kind", s.GetDiscriminator(), "discriminator should match")
	assert.Same(t, externalDocs, s.GetExternalDocs(), "external docs should match")
	assert.True(t, s.GetDeprecated(), "deprecated should match")
	assert.Same(t, exts, s.GetExtensions(), "extensions should match")
}

func TestJSONSchema_Getters_Success(t *testing.T) {
	t.Parallel()

	schema := &asyncapi.Schema{Type: asyncapi.NewTypeFromString("string")}

	tests := []struct {
		name                string
		js                  *asyncapi.JSONSchema
		expectedIsSchema    bool
		expectedSchema      *asyncapi.Schema
		expectedIsBool      bool
		expectedBool        *bool
		expectedIsReference bool
		expectedRef         references.Reference
	}{
		{
			name: "nil json schema",
			js:   nil,
		},
		{
			name:             "schema object",
			js:               asyncapi.NewJSONSchemaFromSchema(schema),
			expectedIsSchema: true,
			expectedSchema:   schema,
		},
		{
			name:           "boolean schema",
			js:             asyncapi.NewJSONSchemaFromBool(false),
			expectedIsBool: true,
			expectedBool:   pointer.From(false),
		},
		{
			name:                "reference object",
			js:                  asyncapi.NewJSONSchemaFromReference("#/components/schemas/User"),
			expectedIsSchema:    true,
			expectedSchema:      &asyncapi.Schema{Ref: pointer.From(references.Reference("#/components/schemas/User"))},
			expectedIsReference: true,
			expectedRef:         "#/components/schemas/User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedIsSchema, tt.js.IsSchema(), "IsSchema should match")
			assert.Equal(t, tt.expectedSchema, tt.js.GetSchema(), "GetSchema should match")
			assert.Equal(t, tt.expectedIsBool, tt.js.IsBool(), "IsBool should match")
			assert.Equal(t, tt.expectedBool, tt.js.GetBool(), "GetBool should match")
			assert.Equal(t, tt.expectedIsReference, tt.js.IsReference(), "IsReference should match")
			assert.Equal(t, tt.expectedRef, tt.js.GetRef(), "GetRef should match")
		})
	}
}

func TestExternalDocumentation_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-owner", yml.CreateStringNode("team")))

	tests := []struct {
		name                string
		docs                *asyncapi.ExternalDocumentation
		expectedIsReference bool
		expectedDescription string
		expectedURL         string
		expectedExtensions  *extensions.Extensions
	}{
		{
			name:               "nil external docs",
			docs:               nil,
			expectedExtensions: extensions.New(),
		},
		{
			name:               "unset fields",
			docs:               &asyncapi.ExternalDocumentation{},
			expectedExtensions: extensions.New(),
		},
		{
			name: "populated fields",
			docs: &asyncapi.ExternalDocumentation{
				Description: pointer.From("More info"),
				URL:         pointer.From("https://example.com/docs"),
				Extensions:  exts,
			},
			expectedDescription: "More info",
			expectedURL:         "https://example.com/docs",
			expectedExtensions:  exts,
		},
		{
			name:                "reference object",
			docs:                &asyncapi.ExternalDocumentation{Ref: pointer.From(references.Reference("#/components/externalDocs/Pets"))},
			expectedIsReference: true,
			expectedExtensions:  extensions.New(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedIsReference, tt.docs.IsReference(), "IsReference should match")
			assert.Equal(t, tt.expectedDescription, tt.docs.GetDescription(), "GetDescription should match")
			assert.Equal(t, tt.expectedURL, tt.docs.GetURL(), "GetURL should match")
			assert.Equal(t, tt.expectedExtensions, tt.docs.GetExtensions(), "GetExtensions should match")
		})
	}
}
