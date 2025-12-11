package hashing

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testEnum string

const (
	testEnumA testEnum = "hello"
)

func TestHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v        any
		wantHash string
	}{
		{
			name:     "nil",
			v:        nil,
			wantHash: "cbf29ce484222325",
		},
		{
			name:     "string",
			v:        "hello",
			wantHash: "a430d84680aabd0b",
		},
		{
			name:     "enum",
			v:        testEnumA,
			wantHash: "a430d84680aabd0b",
		},
		{
			name:     "int",
			v:        42,
			wantHash: "07ee7e07b4b19223",
		},
		{
			name:     "bool",
			v:        true,
			wantHash: "5b5c98ef514dbfa5",
		},
		{
			name:     "float",
			v:        3.14,
			wantHash: "2eb1c202248cb361",
		},
		{
			name:     "pointer",
			v:        pointer.From("hello"),
			wantHash: "a430d84680aabd0b",
		},
		{
			name:     "slice",
			v:        []string{"hello", "world"},
			wantHash: "10d9315e924a5581",
		},
		{
			name:     "map",
			v:        map[string]string{"hello": "world", "nice": "day"},
			wantHash: "da5772baade734c2",
		},
		{
			name: "sequenced map",
			v: sequencedmap.New(
				&sequencedmap.Element[string, string]{
					Key:   "hello",
					Value: "world",
				},
				&sequencedmap.Element[string, string]{
					Key:   "nice",
					Value: "day",
				},
			),
			wantHash: "da5772baade734c2",
		},
		{
			name: "simple struct",
			v: struct {
				Hello string
				Nice  string
			}{
				Hello: "world",
				Nice:  "day",
			},
			wantHash: "3a239a5466995e82",
		},
		{
			name: "model",
			v: tests.TestPrimitiveHighModel{
				StringField:     "hello",
				StringPtrField:  pointer.From("world"),
				BoolField:       true,
				BoolPtrField:    nil,
				IntField:        42,
				IntPtrField:     pointer.From(66),
				Float64Field:    3.14,
				Float64PtrField: pointer.From(2.71),
			},
			wantHash: "75156be433dd08e9",
		},
		{
			name: "model with extensions",
			v: &tests.TestPrimitiveHighModel{
				StringField:     "hello",
				StringPtrField:  pointer.From("world"),
				BoolField:       true,
				BoolPtrField:    nil,
				IntField:        42,
				IntPtrField:     pointer.From(66),
				Float64Field:    3.14,
				Float64PtrField: pointer.From(2.71),
				Extensions: extensions.New(
					extensions.NewElem("hello", yml.CreateStringNode("world")),
				),
			},
			wantHash: "75156be433dd08e9",
		},
		{
			name: "model with embedded map",
			v: &tests.TestEmbeddedMapWithFieldsHighModel{
				Map: sequencedmap.New(sequencedmap.NewElem("hello", &tests.TestPrimitiveHighModel{
					StringField: "world",
				})),
				NameField: "some name",
			},
			wantHash: "4e7758d8af64f31d",
		},
		{
			name:     "boolean based json schema",
			v:        oas3.NewJSONSchemaFromBool(false),
			wantHash: "56934550d006d4b8",
		},
		{
			name: "schema based json schema",
			v: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
				Title: pointer.From("hello"),
				Type:  oas3.NewTypeFromArray([]oas3.SchemaType{oas3.SchemaTypeString}),
				Properties: sequencedmap.New(
					sequencedmap.NewElem("hello", oas3.NewJSONSchemaFromBool(false)),
					sequencedmap.NewElem("world", oas3.NewJSONSchemaFromBool(true)),
				),
			}),
			wantHash: "63f31c8e94c7e87a",
		},
		// Edge Cases and Nil Values
		{
			name:     "nil slice",
			v:        []string(nil),
			wantHash: "cbf29ce484222325",
		},
		{
			name:     "nil map",
			v:        map[string]string(nil),
			wantHash: "cbf29ce484222325",
		},
		{
			name:     "nil pointer",
			v:        (*string)(nil),
			wantHash: "cbf29ce484222325",
		},
		{
			name:     "nil interface",
			v:        interface{}(nil),
			wantHash: "cbf29ce484222325",
		},
		// Empty Collections
		{
			name:     "empty slice",
			v:        []string{},
			wantHash: "cbf29ce484222325",
		},
		{
			name:     "empty map",
			v:        map[string]string{},
			wantHash: "cbf29ce484222325",
		},
		{
			name:     "empty sequenced map",
			v:        sequencedmap.New[string, string](),
			wantHash: "cbf29ce484222325",
		},
		// Array vs Slice Testing
		{
			name:     "array",
			v:        [3]string{"hello", "world", "test"},
			wantHash: "682f36ead6dd8d19",
		},
		// Different Numeric Types
		{
			name:     "int32",
			v:        int32(42),
			wantHash: "07ee7e07b4b19223",
		},
		{
			name:     "int64",
			v:        int64(42),
			wantHash: "07ee7e07b4b19223",
		},
		{
			name:     "float32",
			v:        float32(3.14),
			wantHash: "2eb1c202248cb361",
		},
		{
			name:     "uint32",
			v:        uint32(42),
			wantHash: "07ee7e07b4b19223",
		},
		// Mixed Collections
		{
			name:     "slice of maps",
			v:        []map[string]string{{"a": "1"}, {"b": "2"}},
			wantHash: "55569882ff0df217",
		},
		{
			name:     "map of slices",
			v:        map[string][]string{"a": {"1", "2"}, "b": {"3", "4"}},
			wantHash: "bffd05b179a5cc08",
		},
		// Complex Map Key Types
		{
			name: "struct key map",
			v: map[struct{ Name string }]string{
				{Name: "key1"}: "value1",
				{Name: "key2"}: "value2",
			},
			wantHash: "9da6cef510b3dca5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotHash := Hash(tt.v)
			assert.Equal(t, tt.wantHash, gotHash)
		})
	}
}

func TestHash_Equivalence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		left     any
		right    any
		wantHash string
	}{
		{
			name:  "primitive and defined type equal",
			left:  "hello",
			right: testEnumA,
		},
		{
			name:  "primitive and pointer equal",
			left:  pointer.From("hello"),
			right: "hello",
		},
		{
			name: "map and sequenced map equal",
			left: sequencedmap.New(
				sequencedmap.NewElem("hello", "world"),
				sequencedmap.NewElem("nice", "day"),
			),
			right: map[string]string{
				"hello": "world",
				"nice":  "day",
			},
		},
		{
			name: "too different instances equal",
			left: &tests.TestPrimitiveHighModel{
				StringField:     "hello",
				StringPtrField:  pointer.From("world"),
				BoolField:       true,
				BoolPtrField:    nil,
				IntField:        42,
				IntPtrField:     pointer.From(66),
				Float64Field:    3.14,
				Float64PtrField: pointer.From(2.71),
			},
			right: &tests.TestPrimitiveHighModel{
				StringField:     "hello",
				StringPtrField:  pointer.From("world"),
				BoolField:       true,
				BoolPtrField:    nil,
				IntField:        42,
				IntPtrField:     pointer.From(66),
				Float64Field:    3.14,
				Float64PtrField: pointer.From(2.71),
			},
		},
		// Additional Equivalence Tests
		{
			name:  "array and slice equivalence",
			left:  [2]string{"hello", "world"},
			right: []string{"hello", "world"},
		},
		{
			name:  "nil slice and empty slice equivalence",
			left:  []string(nil),
			right: []string{},
		},
		{
			name:  "different numeric types same value",
			left:  int32(42),
			right: int64(42),
		},
		{
			name:  "int and uint same value",
			left:  int32(42),
			right: uint32(42),
		},
		{
			name:  "nil map and empty map equivalence",
			left:  map[string]string(nil),
			right: map[string]string{},
		},
		{
			name:  "nil pointer and empty string equivalence",
			left:  (*string)(nil),
			right: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			leftHash := Hash(tt.left)
			rightHash := Hash(tt.right)
			assert.Equal(t, leftHash, rightHash)
		})
	}
}

func TestHash_EmbeddedMapComparison_PointerVsValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "pointer_embedded_map",
			testFunc: func(t *testing.T) {
				t.Helper()
				// Create a model with pointer embedded map
				model := &struct {
					*sequencedmap.Map[string, string]
					Name string
				}{
					Map:  sequencedmap.New[string, string](),
					Name: "test",
				}
				model.Set("key1", "value1")
				model.Set("key2", "value2")

				hash := Hash(model)
				assert.NotEmpty(t, hash)
				assert.Len(t, hash, 16) // Hash should be 16 characters
			},
		},
		{
			name: "value_embedded_map",
			testFunc: func(t *testing.T) {
				t.Helper()
				// Create a model with value embedded map
				model := &struct {
					sequencedmap.Map[string, string]
					Name string
				}{
					Map:  *sequencedmap.New[string, string](),
					Name: "test",
				}
				model.Set("key1", "value1")
				model.Set("key2", "value2")

				hash := Hash(model)
				assert.NotEmpty(t, hash)
				assert.Len(t, hash, 16) // Hash should be 16 characters
			},
		},
		{
			name: "both_produce_same_hash",
			testFunc: func(t *testing.T) {
				t.Helper()
				// Create models with same data but different embed types
				ptrModel := &struct {
					*sequencedmap.Map[string, string]
					Name string
				}{
					Map:  sequencedmap.New[string, string](),
					Name: "test",
				}
				ptrModel.Set("key1", "value1")
				ptrModel.Set("key2", "value2")

				valueModel := &struct {
					sequencedmap.Map[string, string]
					Name string
				}{
					Map:  *sequencedmap.New[string, string](),
					Name: "test",
				}
				valueModel.Set("key1", "value1")
				valueModel.Set("key2", "value2")

				ptrHash := Hash(ptrModel)
				valueHash := Hash(valueModel)

				// Both should produce the same hash since they have the same data
				assert.Equal(t, ptrHash, valueHash, "Pointer and value embedded maps with same data should produce same hash")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.testFunc(t)
		})
	}
}

// TestHash_JSONSchemaReferenceVsResolved tests that a JSONSchema with just a $ref
// and the same schema with the reference resolved produce the same hash.
func TestHash_JSONSchemaReferenceVsResolved(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		reference         references.Reference
		resolvedSchemaObj *oas3.Schema
		resolvedBool      *bool // For boolean schemas
	}{
		{
			name:      "simple string schema reference",
			reference: references.Reference("#/components/schemas/StringType"),
			resolvedSchemaObj: &oas3.Schema{
				Type: oas3.NewTypeFromString("string"),
			},
		},
		{
			name:      "object schema with properties reference",
			reference: references.Reference("#/components/schemas/User"),
			resolvedSchemaObj: &oas3.Schema{
				Type: oas3.NewTypeFromString("object"),
				Properties: sequencedmap.New(
					sequencedmap.NewElem("name", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
						Type: oas3.NewTypeFromString("string"),
					})),
					sequencedmap.NewElem("age", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
						Type: oas3.NewTypeFromString("integer"),
					})),
				),
			},
		},
		{
			name:      "schema with title and description reference",
			reference: references.Reference("#/definitions/Product"),
			resolvedSchemaObj: &oas3.Schema{
				Title:       pointer.From("Product"),
				Description: pointer.From("A product in the catalog"),
				Type:        oas3.NewTypeFromString("object"),
			},
		},
		{
			name:         "boolean schema reference",
			reference:    references.Reference("#/components/schemas/AlwaysFalse"),
			resolvedBool: pointer.From(false),
		},
		{
			name:      "array schema reference",
			reference: references.Reference("#/components/schemas/StringArray"),
			resolvedSchemaObj: &oas3.Schema{
				Type: oas3.NewTypeFromString("array"),
				Items: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type: oas3.NewTypeFromString("string"),
				}),
			},
		},
		{
			name:      "number schema with constraints",
			reference: references.Reference("#/components/schemas/Percentage"),
			resolvedSchemaObj: &oas3.Schema{
				Type:    oas3.NewTypeFromString("number"),
				Minimum: pointer.From(0.0),
				Maximum: pointer.From(100.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create an unresolved reference schema
			unresolvedRef := oas3.NewJSONSchemaFromReference(tt.reference)

			// Create a resolved reference schema using NewReferencedScheme
			var resolvedContent *oas3.JSONSchema[oas3.Concrete]
			if tt.resolvedBool != nil {
				resolvedContent = oas3.NewJSONSchemaFromBool(*tt.resolvedBool).GetResolvedSchema()
			} else {
				resolvedContent = oas3.NewJSONSchemaFromSchema[oas3.Concrete](tt.resolvedSchemaObj)
			}

			resolvedRef := oas3.NewReferencedScheme(
				t.Context(),
				tt.reference,
				resolvedContent,
			)

			// Hash both the unresolved and resolved references
			unresolvedHash := Hash(unresolvedRef)
			resolvedHash := Hash(resolvedRef)

			assert.Equal(t, unresolvedHash, resolvedHash,
				"Hash of unresolved reference should equal hash of resolved reference")
		})
	}
}

func TestHash_JSONSchemaFieldOrdering(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Two identical User schemas with different field ordering
	userSchema1YAML := `type: object
properties:
  id:
    type: string
    format: uuid
  name:
    type: string
    maxLength: 100
    minLength: 1
  email:
    type: string
    format: email
required:
  - id
  - name
  - email`

	userSchema2YAML := `type: object
required:
  - id
  - name
  - email
properties:
  id:
    type: string
    format: uuid
  name:
    type: string
    minLength: 1
    maxLength: 100
  email:
    type: string
    format: email`

	// Parse both schemas using marshaller.Unmarshal
	var schema1 oas3.Schema
	_, err := marshaller.Unmarshal(ctx, strings.NewReader(userSchema1YAML), &schema1)
	require.NoError(t, err, "should parse first schema")
	_ = schema1.Validate(ctx)

	var schema2 oas3.Schema
	_, err = marshaller.Unmarshal(ctx, strings.NewReader(userSchema2YAML), &schema2)
	require.NoError(t, err, "should parse second schema")

	// Create JSONSchema wrappers
	jsonSchema1 := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&schema1)
	jsonSchema2 := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&schema2)

	// Hash both schemas
	hash1 := Hash(jsonSchema1)
	hash2 := Hash(jsonSchema2)

	t.Logf("Schema 1 hash: %s", hash1)
	t.Logf("Schema 2 hash: %s", hash2)

	// They should be equal since they represent the same schema
	assert.Equal(t, hash1, hash2, "identical schemas should have the same hash regardless of field ordering")
}

func TestHash_YamlNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		node     *yaml.Node
		wantHash string
	}{
		{
			name:     "nil node",
			node:     nil,
			wantHash: "cbf29ce484222325", // empty string hash
		},
		{
			name: "scalar string node",
			node: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "hello",
			},
			wantHash: "049bf33d4f533213",
		},
		{
			name: "scalar int node",
			node: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!int",
				Value: "42",
			},
			wantHash: "f8aa31eb804642d5",
		},
		{
			name: "scalar without tag",
			node: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "test",
			},
			wantHash: "6a1b3c98d6cf2e16",
		},
		{
			name: "scalar without value",
			node: &yaml.Node{
				Kind: yaml.ScalarNode,
				Tag:  "!!null",
			},
			wantHash: "70dd4271b9b4bf98",
		},
		{
			name: "empty scalar node",
			node: &yaml.Node{
				Kind: yaml.ScalarNode,
			},
			wantHash: "535a299dfeb2aee1",
		},
		{
			name: "mapping node with children",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"},
				},
			},
			wantHash: "8f158417b793a189",
		},
		{
			name: "sequence node with children",
			node: &yaml.Node{
				Kind: yaml.SequenceNode,
				Tag:  "!!seq",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item1"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item2"},
				},
			},
			wantHash: "fac33aff0234a67f",
		},
		{
			name: "document node wrapping scalar",
			node: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "document content"},
				},
			},
			wantHash: "ae29e55a43581ec6",
		},
		{
			name: "deeply nested node",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "outer"},
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: "inner"},
							{Kind: yaml.ScalarNode, Value: "value"},
						},
					},
				},
			},
			wantHash: "4999ae359f57db96",
		},
		{
			name: "node with positional metadata should produce same hash as without",
			node: &yaml.Node{
				Kind:   yaml.ScalarNode,
				Tag:    "!!str",
				Value:  "hello",
				Line:   10,
				Column: 5,
				Style:  yaml.DoubleQuotedStyle,
			},
			wantHash: "049bf33d4f533213", // Same as basic scalar string "hello"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotHash := Hash(tt.node)
			assert.Equal(t, tt.wantHash, gotHash, "hash should match expected value")
		})
	}
}

func TestHash_YamlNodeStruct_Success(t *testing.T) {
	t.Parallel()

	// Test passing yaml.Node as value (not pointer) triggers the struct case
	tests := []struct {
		name     string
		node     yaml.Node
		wantHash string
	}{
		{
			name: "scalar string node as value",
			node: yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "hello",
			},
			wantHash: "049bf33d4f533213",
		},
		{
			name: "empty node as value",
			node: yaml.Node{
				Kind: yaml.ScalarNode,
			},
			wantHash: "535a299dfeb2aee1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotHash := Hash(tt.node)
			assert.Equal(t, tt.wantHash, gotHash, "hash should match expected value")
		})
	}
}

func TestHash_YamlNodeEquivalence_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  *yaml.Node
		right *yaml.Node
	}{
		{
			name: "positional metadata does not affect hash",
			left: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "hello",
			},
			right: &yaml.Node{
				Kind:   yaml.ScalarNode,
				Tag:    "!!str",
				Value:  "hello",
				Line:   100,
				Column: 50,
			},
		},
		{
			name: "style does not affect hash",
			left: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "test",
				Style: yaml.LiteralStyle,
			},
			right: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "test",
				Style: yaml.DoubleQuotedStyle,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			leftHash := Hash(tt.left)
			rightHash := Hash(tt.right)
			assert.Equal(t, leftHash, rightHash, "equivalent nodes should have same hash")
		})
	}
}
