package oas3

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSchema_IsEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema1  *Schema
		schema2  *Schema
		expected bool
	}{
		{
			name:     "both nil schemas should be equal",
			schema1:  nil,
			schema2:  nil,
			expected: true,
		},
		{
			name:     "empty schemas should be equal",
			schema1:  &Schema{},
			schema2:  &Schema{},
			expected: true,
		},
		{
			name: "schemas with same basic fields should be equal",
			schema1: &Schema{
				Title:       pointer.From("Test Schema"),
				Description: pointer.From("A test schema"),
				Type:        NewTypeFromString(SchemaTypeString),
			},
			schema2: &Schema{
				Title:       pointer.From("Test Schema"),
				Description: pointer.From("A test schema"),
				Type:        NewTypeFromString(SchemaTypeString),
			},
			expected: true,
		},
		{
			name: "schemas with same reference should be equal",
			schema1: &Schema{
				Ref: pointer.From(references.Reference("#/components/schemas/User")),
			},
			schema2: &Schema{
				Ref: pointer.From(references.Reference("#/components/schemas/User")),
			},
			expected: true,
		},
		{
			name: "schemas with same numeric constraints should be equal",
			schema1: &Schema{
				Type:       NewTypeFromString(SchemaTypeNumber),
				Minimum:    pointer.From(0.0),
				Maximum:    pointer.From(100.0),
				MultipleOf: pointer.From(5.0),
			},
			schema2: &Schema{
				Type:       NewTypeFromString(SchemaTypeNumber),
				Minimum:    pointer.From(0.0),
				Maximum:    pointer.From(100.0),
				MultipleOf: pointer.From(5.0),
			},
			expected: true,
		},
		{
			name: "schemas with same string constraints should be equal",
			schema1: &Schema{
				Type:      NewTypeFromString(SchemaTypeString),
				MinLength: pointer.From(int64(1)),
				MaxLength: pointer.From(int64(50)),
				Pattern:   pointer.From("^[a-zA-Z]+$"),
				Format:    pointer.From("email"),
			},
			schema2: &Schema{
				Type:      NewTypeFromString(SchemaTypeString),
				MinLength: pointer.From(int64(1)),
				MaxLength: pointer.From(int64(50)),
				Pattern:   pointer.From("^[a-zA-Z]+$"),
				Format:    pointer.From("email"),
			},
			expected: true,
		},
		{
			name: "schemas with same array constraints should be equal",
			schema1: &Schema{
				Type:        NewTypeFromString(SchemaTypeArray),
				MinItems:    pointer.From(int64(1)),
				MaxItems:    pointer.From(int64(10)),
				UniqueItems: pointer.From(true),
			},
			schema2: &Schema{
				Type:        NewTypeFromString(SchemaTypeArray),
				MinItems:    pointer.From(int64(1)),
				MaxItems:    pointer.From(int64(10)),
				UniqueItems: pointer.From(true),
			},
			expected: true,
		},
		{
			name: "schemas with same object constraints should be equal",
			schema1: &Schema{
				Type:          NewTypeFromString(SchemaTypeObject),
				MinProperties: pointer.From(int64(1)),
				MaxProperties: pointer.From(int64(10)),
				Required:      []string{"name", "email"},
			},
			schema2: &Schema{
				Type:          NewTypeFromString(SchemaTypeObject),
				MinProperties: pointer.From(int64(1)),
				MaxProperties: pointer.From(int64(10)),
				Required:      []string{"name", "email"},
			},
			expected: true,
		},
		{
			name: "schemas with same boolean flags should be equal",
			schema1: &Schema{
				Nullable:   pointer.From(true),
				ReadOnly:   pointer.From(false),
				WriteOnly:  pointer.From(true),
				Deprecated: pointer.From(false),
			},
			schema2: &Schema{
				Nullable:   pointer.From(true),
				ReadOnly:   pointer.From(false),
				WriteOnly:  pointer.From(true),
				Deprecated: pointer.From(false),
			},
			expected: true,
		},
		{
			name: "schemas with same $id should be equal",
			schema1: &Schema{
				ID: pointer.From("https://example.com/schemas/user"),
			},
			schema2: &Schema{
				ID: pointer.From("https://example.com/schemas/user"),
			},
			expected: true,
		},
		{
			name: "schemas with same external docs should be equal",
			schema1: &Schema{
				ExternalDocs: &ExternalDocumentation{
					URL:         "https://example.com/docs",
					Description: pointer.From("External documentation"),
				},
			},
			schema2: &Schema{
				ExternalDocs: &ExternalDocumentation{
					URL:         "https://example.com/docs",
					Description: pointer.From("External documentation"),
				},
			},
			expected: true,
		},
		{
			name: "schemas with same XML metadata should be equal",
			schema1: &Schema{
				XML: &XML{
					Name:      pointer.From("user"),
					Namespace: pointer.From("http://example.com/schema"),
					Prefix:    pointer.From("ex"),
					Attribute: pointer.From(false),
					Wrapped:   pointer.From(true),
				},
			},
			schema2: &Schema{
				XML: &XML{
					Name:      pointer.From("user"),
					Namespace: pointer.From("http://example.com/schema"),
					Prefix:    pointer.From("ex"),
					Attribute: pointer.From(false),
					Wrapped:   pointer.From(true),
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := tt.schema1.IsEqual(tt.schema2)
			assert.Equal(t, tt.expected, actual, "schemas should match expected equality")
		})
	}
}

func TestSchema_IsEqual_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema1  *Schema
		schema2  *Schema
		expected bool
	}{
		{
			name:     "nil vs non-nil schema should not be equal",
			schema1:  nil,
			schema2:  &Schema{},
			expected: false,
		},
		{
			name:     "non-nil vs nil schema should not be equal",
			schema1:  &Schema{},
			schema2:  nil,
			expected: false,
		},
		{
			name: "schemas with different titles should not be equal",
			schema1: &Schema{
				Title: pointer.From("Schema A"),
			},
			schema2: &Schema{
				Title: pointer.From("Schema B"),
			},
			expected: false,
		},
		{
			name: "schemas with different types should not be equal",
			schema1: &Schema{
				Type: NewTypeFromString(SchemaTypeString),
			},
			schema2: &Schema{
				Type: NewTypeFromString(SchemaTypeNumber),
			},
			expected: false,
		},
		{
			name: "schemas with different references should not be equal",
			schema1: &Schema{
				Ref: pointer.From(references.Reference("#/components/schemas/User")),
			},
			schema2: &Schema{
				Ref: pointer.From(references.Reference("#/components/schemas/Product")),
			},
			expected: false,
		},
		{
			name: "schemas with different minimum values should not be equal",
			schema1: &Schema{
				Type:    NewTypeFromString(SchemaTypeNumber),
				Minimum: pointer.From(0.0),
			},
			schema2: &Schema{
				Type:    NewTypeFromString(SchemaTypeNumber),
				Minimum: pointer.From(1.0),
			},
			expected: false,
		},
		{
			name: "schemas with different required fields should not be equal",
			schema1: &Schema{
				Type:     NewTypeFromString(SchemaTypeObject),
				Required: []string{"name", "email"},
			},
			schema2: &Schema{
				Type:     NewTypeFromString(SchemaTypeObject),
				Required: []string{"name", "phone"},
			},
			expected: false,
		},
		{
			name: "schemas with different boolean flags should not be equal",
			schema1: &Schema{
				Nullable: pointer.From(true),
			},
			schema2: &Schema{
				Nullable: pointer.From(false),
			},
			expected: false,
		},
		{
			name: "schemas with different external docs should not be equal",
			schema1: &Schema{
				ExternalDocs: &ExternalDocumentation{
					URL: "https://example.com/docs",
				},
			},
			schema2: &Schema{
				ExternalDocs: &ExternalDocumentation{
					URL: "https://different.com/docs",
				},
			},
			expected: false,
		},
		{
			name: "schema with external docs vs schema without should not be equal",
			schema1: &Schema{
				ExternalDocs: &ExternalDocumentation{
					URL: "https://example.com/docs",
				},
			},
			schema2:  &Schema{},
			expected: false,
		},
		{
			name: "schemas with different $id should not be equal",
			schema1: &Schema{
				ID: pointer.From("https://example.com/schemas/user"),
			},
			schema2: &Schema{
				ID: pointer.From("https://example.com/schemas/product"),
			},
			expected: false,
		},
		{
			name: "schema with $id vs schema without $id should not be equal",
			schema1: &Schema{
				ID: pointer.From("https://example.com/schemas/user"),
			},
			schema2:  &Schema{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := tt.schema1.IsEqual(tt.schema2)
			assert.Equal(t, tt.expected, actual, "schemas should match expected equality")
		})
	}
}

func TestSchema_IsEqual_MismatchFields(t *testing.T) {
	t.Parallel()

	child := func(title string) *JSONSchema[Referenceable] {
		return NewJSONSchemaFromSchema[Referenceable](&Schema{Title: pointer.From(title)})
	}
	value := func(text string) values.Value {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: text}
	}

	tests := []struct {
		name  string
		left  *Schema
		right *Schema
	}{
		{name: "exclusive maximum", left: &Schema{ExclusiveMaximum: NewExclusiveMaximumFromFloat64(1)}, right: &Schema{ExclusiveMaximum: NewExclusiveMaximumFromFloat64(2)}},
		{name: "exclusive minimum", left: &Schema{ExclusiveMinimum: NewExclusiveMinimumFromFloat64(1)}, right: &Schema{ExclusiveMinimum: NewExclusiveMinimumFromFloat64(2)}},
		{name: "all of", left: &Schema{AllOf: []*JSONSchema[Referenceable]{child("left")}}, right: &Schema{AllOf: []*JSONSchema[Referenceable]{child("right")}}},
		{name: "one of", left: &Schema{OneOf: []*JSONSchema[Referenceable]{child("left")}}, right: &Schema{OneOf: []*JSONSchema[Referenceable]{child("right")}}},
		{name: "any of", left: &Schema{AnyOf: []*JSONSchema[Referenceable]{child("left")}}, right: &Schema{AnyOf: []*JSONSchema[Referenceable]{child("right")}}},
		{name: "prefix items", left: &Schema{PrefixItems: []*JSONSchema[Referenceable]{child("left")}}, right: &Schema{PrefixItems: []*JSONSchema[Referenceable]{child("right")}}},
		{name: "contains", left: &Schema{Contains: child("left")}, right: &Schema{Contains: child("right")}},
		{name: "if", left: &Schema{If: child("left")}, right: &Schema{If: child("right")}},
		{name: "else", left: &Schema{Else: child("left")}, right: &Schema{Else: child("right")}},
		{name: "then", left: &Schema{Then: child("left")}, right: &Schema{Then: child("right")}},
		{name: "not", left: &Schema{Not: child("left")}, right: &Schema{Not: child("right")}},
		{name: "property names", left: &Schema{PropertyNames: child("left")}, right: &Schema{PropertyNames: child("right")}},
		{name: "unevaluated items", left: &Schema{UnevaluatedItems: child("left")}, right: &Schema{UnevaluatedItems: child("right")}},
		{name: "unevaluated properties", left: &Schema{UnevaluatedProperties: child("left")}, right: &Schema{UnevaluatedProperties: child("right")}},
		{name: "items", left: &Schema{Items: child("left")}, right: &Schema{Items: child("right")}},
		{name: "additional properties", left: &Schema{AdditionalProperties: child("left")}, right: &Schema{AdditionalProperties: child("right")}},
		{name: "dependent schemas", left: &Schema{DependentSchemas: sequencedmap.New(sequencedmap.NewElem("key", child("left")))}, right: &Schema{DependentSchemas: sequencedmap.New(sequencedmap.NewElem("key", child("right")))}},
		{name: "pattern properties", left: &Schema{PatternProperties: sequencedmap.New(sequencedmap.NewElem("key", child("left")))}, right: &Schema{PatternProperties: sequencedmap.New(sequencedmap.NewElem("key", child("right")))}},
		{name: "properties", left: &Schema{Properties: sequencedmap.New(sequencedmap.NewElem("key", child("left")))}, right: &Schema{Properties: sequencedmap.New(sequencedmap.NewElem("key", child("right")))}},
		{name: "defs", left: &Schema{Defs: sequencedmap.New(sequencedmap.NewElem("key", child("left")))}, right: &Schema{Defs: sequencedmap.New(sequencedmap.NewElem("key", child("right")))}},
		{name: "minimum contains", left: &Schema{MinContains: pointer.From(int64(1))}, right: &Schema{MinContains: pointer.From(int64(2))}},
		{name: "maximum contains", left: &Schema{MaxContains: pointer.From(int64(1))}, right: &Schema{MaxContains: pointer.From(int64(2))}},
		{name: "anchor", left: &Schema{Anchor: pointer.From("left")}, right: &Schema{Anchor: pointer.From("right")}},
		{name: "multiple of", left: &Schema{MultipleOf: pointer.From(1.0)}, right: &Schema{MultipleOf: pointer.From(2.0)}},
		{name: "maximum", left: &Schema{Maximum: pointer.From(1.0)}, right: &Schema{Maximum: pointer.From(2.0)}},
		{name: "maximum length", left: &Schema{MaxLength: pointer.From(int64(1))}, right: &Schema{MaxLength: pointer.From(int64(2))}},
		{name: "minimum length", left: &Schema{MinLength: pointer.From(int64(1))}, right: &Schema{MinLength: pointer.From(int64(2))}},
		{name: "pattern", left: &Schema{Pattern: pointer.From("left")}, right: &Schema{Pattern: pointer.From("right")}},
		{name: "content encoding", left: &Schema{ContentEncoding: pointer.From("left")}, right: &Schema{ContentEncoding: pointer.From("right")}},
		{name: "content media type", left: &Schema{ContentMediaType: pointer.From("left")}, right: &Schema{ContentMediaType: pointer.From("right")}},
		{name: "content schema", left: &Schema{ContentSchema: child("left")}, right: &Schema{ContentSchema: child("right")}},
		{name: "format", left: &Schema{Format: pointer.From("left")}, right: &Schema{Format: pointer.From("right")}},
		{name: "maximum items", left: &Schema{MaxItems: pointer.From(int64(1))}, right: &Schema{MaxItems: pointer.From(int64(2))}},
		{name: "minimum items", left: &Schema{MinItems: pointer.From(int64(1))}, right: &Schema{MinItems: pointer.From(int64(2))}},
		{name: "unique items", left: &Schema{UniqueItems: pointer.From(true)}, right: &Schema{UniqueItems: pointer.From(false)}},
		{name: "maximum properties", left: &Schema{MaxProperties: pointer.From(int64(1))}, right: &Schema{MaxProperties: pointer.From(int64(2))}},
		{name: "minimum properties", left: &Schema{MinProperties: pointer.From(int64(1))}, right: &Schema{MinProperties: pointer.From(int64(2))}},
		{name: "read only", left: &Schema{ReadOnly: pointer.From(true)}, right: &Schema{ReadOnly: pointer.From(false)}},
		{name: "write only", left: &Schema{WriteOnly: pointer.From(true)}, right: &Schema{WriteOnly: pointer.From(false)}},
		{name: "deprecated", left: &Schema{Deprecated: pointer.From(true)}, right: &Schema{Deprecated: pointer.From(false)}},
		{name: "schema dialect", left: &Schema{Schema: pointer.From("left")}, right: &Schema{Schema: pointer.From("right")}},
		{name: "examples", left: &Schema{Examples: []values.Value{value("left")}}, right: &Schema{Examples: []values.Value{value("right")}}},
		{name: "enum", left: &Schema{Enum: []values.Value{value("left")}}, right: &Schema{Enum: []values.Value{value("right")}}},
		{name: "default", left: &Schema{Default: value("left")}, right: &Schema{Default: value("right")}},
		{name: "const", left: &Schema{Const: value("left")}, right: &Schema{Const: value("right")}},
		{name: "example", left: &Schema{Example: value("left")}, right: &Schema{Example: value("right")}},
		{name: "discriminator", left: &Schema{Discriminator: &Discriminator{PropertyName: "left"}}, right: &Schema{Discriminator: &Discriminator{PropertyName: "right"}}},
		{name: "XML", left: &Schema{XML: &XML{Name: pointer.From("left")}}, right: &Schema{XML: &XML{Name: pointer.From("right")}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, tt.left.IsEqual(tt.right), "schemas differing in %s should not be equal", tt.name)
		})
	}
}

func TestSchema_ShallowCopy_PopulatedCollections(t *testing.T) {
	t.Parallel()

	child := NewJSONSchemaFromSchema[Referenceable](&Schema{Title: pointer.From("child")})
	property := NewJSONSchemaFromSchema[Referenceable](&Schema{Title: pointer.From("property")})
	schema := &Schema{
		AllOf:            []*JSONSchema[Referenceable]{child},
		Required:         []string{"name"},
		DependentSchemas: sequencedmap.New(sequencedmap.NewElem("dependency", child)),
		Properties:       sequencedmap.New(sequencedmap.NewElem("name", property)),
	}

	copied := schema.ShallowCopy()
	assert.NotSame(t, schema, copied)
	assert.Same(t, child, copied.AllOf[0], "slice elements remain shared in a shallow copy")
	copied.AllOf[0] = property
	copied.Required[0] = "changed"
	copied.Properties.Set("other", child)
	assert.Same(t, child, schema.AllOf[0], "slice backing storage should be independent")
	assert.Equal(t, []string{"name"}, schema.Required, "copied string slices should be independent")
	assert.Equal(t, 1, schema.Properties.Len(), "copied maps should be independent")
	assert.Equal(t, 2, copied.Properties.Len())
}

func TestSchema_IsEqual_WithComplexTypes(t *testing.T) {
	t.Parallel()

	// Test with discriminator
	t.Run("schemas with same discriminator should be equal", func(t *testing.T) {
		t.Parallel()
		mapping := sequencedmap.New(
			sequencedmap.NewElem("cat", "#/components/schemas/Cat"),
			sequencedmap.NewElem("dog", "#/components/schemas/Dog"),
		)

		schema1 := &Schema{
			Discriminator: &Discriminator{
				PropertyName: "petType",
				Mapping:      mapping,
			},
		}

		schema2 := &Schema{
			Discriminator: &Discriminator{
				PropertyName: "petType",
				Mapping:      mapping,
			},
		}

		assert.True(t, schema1.IsEqual(schema2))
	})

	// Test with extensions
	t.Run("schemas with same extensions should be equal", func(t *testing.T) {
		t.Parallel()
		ext1 := extensions.New(
			extensions.NewElem("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}),
		)
		ext2 := extensions.New(
			extensions.NewElem("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}),
		)

		schema1 := &Schema{Extensions: ext1}
		schema2 := &Schema{Extensions: ext2}

		assert.True(t, schema1.IsEqual(schema2))
	})

	// Test with different extensions
	t.Run("schemas with different extensions should not be equal", func(t *testing.T) {
		t.Parallel()
		ext1 := extensions.New(
			extensions.NewElem("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "test1"}),
		)
		ext2 := extensions.New(
			extensions.NewElem("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "test2"}),
		)

		schema1 := &Schema{Extensions: ext1}
		schema2 := &Schema{Extensions: ext2}

		assert.False(t, schema1.IsEqual(schema2))
	})
}

func TestSchema_IsEqual_WithValues(t *testing.T) {
	t.Parallel()

	// Test with same default values
	t.Run("schemas with same default values should be equal", func(t *testing.T) {
		t.Parallel()
		defaultValue := &yaml.Node{Kind: yaml.ScalarNode, Value: "default"}

		schema1 := &Schema{Default: defaultValue}
		schema2 := &Schema{Default: defaultValue}

		assert.True(t, schema1.IsEqual(schema2))
	})

	// Test with different default values
	t.Run("schemas with different default values should not be equal", func(t *testing.T) {
		t.Parallel()
		defaultValue1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "default1"}
		defaultValue2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "default2"}

		schema1 := &Schema{Default: defaultValue1}
		schema2 := &Schema{Default: defaultValue2}

		assert.False(t, schema1.IsEqual(schema2))
	})

	// Test with same enum values
	t.Run("schemas with same enum values should be equal", func(t *testing.T) {
		t.Parallel()
		enum1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"}
		enum2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"}

		schema1 := &Schema{Enum: []values.Value{enum1, enum2}}
		schema2 := &Schema{Enum: []values.Value{enum1, enum2}}

		assert.True(t, schema1.IsEqual(schema2))
	})

	// Test with different enum values
	t.Run("schemas with different enum values should not be equal", func(t *testing.T) {
		t.Parallel()
		enum1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"}
		enum2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"}
		enum3 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value3"}

		schema1 := &Schema{Enum: []values.Value{enum1, enum2}}
		schema2 := &Schema{Enum: []values.Value{enum1, enum3}}

		assert.False(t, schema1.IsEqual(schema2))
	})
}

func TestSchema_IsEqual_WithEmptyNilCollections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schema1  *Schema
		schema2  *Schema
		expected bool
	}{
		{
			name: "nil Required slice vs empty Required slice should be equal",
			schema1: &Schema{
				Type:     NewTypeFromString(SchemaTypeObject),
				Required: nil,
			},
			schema2: &Schema{
				Type:     NewTypeFromString(SchemaTypeObject),
				Required: []string{},
			},
			expected: true,
		},
		{
			name: "empty Required slice vs nil Required slice should be equal",
			schema1: &Schema{
				Type:     NewTypeFromString(SchemaTypeObject),
				Required: []string{},
			},
			schema2: &Schema{
				Type:     NewTypeFromString(SchemaTypeObject),
				Required: nil,
			},
			expected: true,
		},
		{
			name: "nil Examples slice vs empty Examples slice should be equal",
			schema1: &Schema{
				Examples: nil,
			},
			schema2: &Schema{
				Examples: []values.Value{},
			},
			expected: true,
		},
		{
			name: "empty Examples slice vs nil Examples slice should be equal",
			schema1: &Schema{
				Examples: []values.Value{},
			},
			schema2: &Schema{
				Examples: nil,
			},
			expected: true,
		},
		{
			name: "nil Enum slice vs empty Enum slice should be equal",
			schema1: &Schema{
				Enum: nil,
			},
			schema2: &Schema{
				Enum: []values.Value{},
			},
			expected: true,
		},
		{
			name: "empty Enum slice vs nil Enum slice should be equal",
			schema1: &Schema{
				Enum: []values.Value{},
			},
			schema2: &Schema{
				Enum: nil,
			},
			expected: true,
		},
		{
			name: "nil AllOf slice vs empty AllOf slice should be equal",
			schema1: &Schema{
				AllOf: nil,
			},
			schema2: &Schema{
				AllOf: []*JSONSchema[Referenceable]{},
			},
			expected: true,
		},
		{
			name: "empty AllOf slice vs nil AllOf slice should be equal",
			schema1: &Schema{
				AllOf: []*JSONSchema[Referenceable]{},
			},
			schema2: &Schema{
				AllOf: nil,
			},
			expected: true,
		},
		{
			name: "nil OneOf slice vs empty OneOf slice should be equal",
			schema1: &Schema{
				OneOf: nil,
			},
			schema2: &Schema{
				OneOf: []*JSONSchema[Referenceable]{},
			},
			expected: true,
		},
		{
			name: "empty OneOf slice vs nil OneOf slice should be equal",
			schema1: &Schema{
				OneOf: []*JSONSchema[Referenceable]{},
			},
			schema2: &Schema{
				OneOf: nil,
			},
			expected: true,
		},
		{
			name: "nil AnyOf slice vs empty AnyOf slice should be equal",
			schema1: &Schema{
				AnyOf: nil,
			},
			schema2: &Schema{
				AnyOf: []*JSONSchema[Referenceable]{},
			},
			expected: true,
		},
		{
			name: "empty AnyOf slice vs nil AnyOf slice should be equal",
			schema1: &Schema{
				AnyOf: []*JSONSchema[Referenceable]{},
			},
			schema2: &Schema{
				AnyOf: nil,
			},
			expected: true,
		},
		{
			name: "nil PrefixItems slice vs empty PrefixItems slice should be equal",
			schema1: &Schema{
				PrefixItems: nil,
			},
			schema2: &Schema{
				PrefixItems: []*JSONSchema[Referenceable]{},
			},
			expected: true,
		},
		{
			name: "empty PrefixItems slice vs nil PrefixItems slice should be equal",
			schema1: &Schema{
				PrefixItems: []*JSONSchema[Referenceable]{},
			},
			schema2: &Schema{
				PrefixItems: nil,
			},
			expected: true,
		},
		{
			name: "nil Properties map vs empty Properties map should be equal",
			schema1: &Schema{
				Properties: nil,
			},
			schema2: &Schema{
				Properties: sequencedmap.New[string, *JSONSchema[Referenceable]](),
			},
			expected: true,
		},
		{
			name: "empty Properties map vs nil Properties map should be equal",
			schema1: &Schema{
				Properties: sequencedmap.New[string, *JSONSchema[Referenceable]](),
			},
			schema2: &Schema{
				Properties: nil,
			},
			expected: true,
		},
		{
			name: "nil DependentSchemas map vs empty DependentSchemas map should be equal",
			schema1: &Schema{
				DependentSchemas: nil,
			},
			schema2: &Schema{
				DependentSchemas: sequencedmap.New[string, *JSONSchema[Referenceable]](),
			},
			expected: true,
		},
		{
			name: "empty DependentSchemas map vs nil DependentSchemas map should be equal",
			schema1: &Schema{
				DependentSchemas: sequencedmap.New[string, *JSONSchema[Referenceable]](),
			},
			schema2: &Schema{
				DependentSchemas: nil,
			},
			expected: true,
		},
		{
			name: "nil PatternProperties map vs empty PatternProperties map should be equal",
			schema1: &Schema{
				PatternProperties: nil,
			},
			schema2: &Schema{
				PatternProperties: sequencedmap.New[string, *JSONSchema[Referenceable]](),
			},
			expected: true,
		},
		{
			name: "empty PatternProperties map vs nil PatternProperties map should be equal",
			schema1: &Schema{
				PatternProperties: sequencedmap.New[string, *JSONSchema[Referenceable]](),
			},
			schema2: &Schema{
				PatternProperties: nil,
			},
			expected: true,
		},
		{
			name: "nil Extensions vs empty Extensions should be equal",
			schema1: &Schema{
				Extensions: nil,
			},
			schema2: &Schema{
				Extensions: extensions.New(),
			},
			expected: true,
		},
		{
			name: "empty Extensions vs nil Extensions should be equal",
			schema1: &Schema{
				Extensions: extensions.New(),
			},
			schema2: &Schema{
				Extensions: nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := tt.schema1.IsEqual(tt.schema2)
			assert.Equal(t, tt.expected, actual, "schemas should match expected equality for empty/nil collections")
		})
	}
}
