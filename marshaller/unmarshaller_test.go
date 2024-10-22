package marshaller

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type Extensions = *sequencedmap.Map[string, Node[Extension]]

type TestCoreModel struct {
	PrimitiveField              Node[string]                                     `key:"primitiveField"`
	NestedModelField            Node[TestNestedModel]                            `key:"nestedModelField"`
	NestedModelOptionalField    Node[*TestNestedModel]                           `key:"nestedModelOptionalField"`
	SliceNestedModelField       Node[[]TestNestedModel]                          `key:"sliceNestedModelField"`
	MapRequiredNestedModelField Node[*sequencedmap.Map[string, TestNestedModel]] `key:"mapRequiredNestedModelField" required:"true"`
	Extensions                  Extensions                                       `key:"extensions"`
}

type TestNestedModel struct {
	PrimitiveOptionalField      Node[*string]                        `key:"primitiveOptionalField"`
	SlicePrimitiveField         Node[[]string]                       `key:"slicePrimitiveField"`
	SliceRequiredPrimitiveField Node[[]string]                       `key:"sliceRequiredPrimitiveField" required:"true"`
	MapPrimitiveField           Node[*sequencedmap.Map[string, int]] `key:"mapPrimitiveField"`
	Extensions                  Extensions                           `key:"extensions"`
}

func (t *TestNestedModel) Unmarshal(ctx context.Context, node *yaml.Node) error {
	return UnmarshalStruct(ctx, node, t)
}

func TestUnmarshal_Success(t *testing.T) {
	testYaml := `primitiveField: "hello world"
nestedModelField:
  primitiveOptionalField: "guess who"
  slicePrimitiveField: ["where", "are", "you"]
  sliceRequiredPrimitiveField: ["I", "am", "here"]
  mapPrimitiveField:
    a: 1
    b: 2
  x-test: some-value
nestedModelOptionalField:
  slicePrimitiveField: ["some", "other", "values"]
  sliceRequiredPrimitiveField: ["a", "b", "c"]
sliceNestedModelField:
  - slicePrimitiveField: ["d", "e", "f"]
    sliceRequiredPrimitiveField: ["g", "h", "i"]
  - slicePrimitiveField: ["j", "k", "l"]
    sliceRequiredPrimitiveField: ["m", "n", "o"]
mapRequiredNestedModelField:
  z:
    slicePrimitiveField: ["p", "q", "r"]
    sliceRequiredPrimitiveField: ["s", "t", "u"]
    x-test: some-value
  x: 
    slicePrimitiveField: ["w", "x", "y"]
    sliceRequiredPrimitiveField: ["1", "2", "3"]
x-test-2: some-value-2
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var out TestCoreModel
	err = Unmarshal(context.Background(), &node, &out)
	require.NoError(t, err)

	assertNodeField(t, "primitiveField", 1, "hello world", 1, out.PrimitiveField)

	assertModelNodeField(t, "nestedModelField", 2, 3, out.NestedModelField)
	assertNodeField(t, "primitiveOptionalField", 3, pointer.From("guess who"), 3, out.NestedModelField.Value.PrimitiveOptionalField)
	assertNodeField(t, "slicePrimitiveField", 4, []string{"where", "are", "you"}, 4, out.NestedModelField.Value.SlicePrimitiveField)
	assertNodeField(t, "sliceRequiredPrimitiveField", 5, []string{"I", "am", "here"}, 5, out.NestedModelField.Value.SliceRequiredPrimitiveField)
	assertNodeField(t, "mapPrimitiveField", 6, sequencedmap.New(sequencedmap.NewElem("a", 1), sequencedmap.NewElem("b", 2)), 7, out.NestedModelField.Value.MapPrimitiveField)
	xTestExtensionNodeNestedModelField := testutils.CreateStringYamlNode("some-value", 9, 11)
	assert.Equal(t, sequencedmap.New(sequencedmap.NewElem("x-test", Node[Extension]{
		Key:       "x-test",
		KeyNode:   testutils.CreateStringYamlNode("x-test", 9, 3),
		Value:     xTestExtensionNodeNestedModelField,
		ValueNode: xTestExtensionNodeNestedModelField,
	})), out.NestedModelField.Value.Extensions)

	assertModelNodeField(t, "nestedModelOptionalField", 10, 11, out.NestedModelOptionalField)
	assertNodeField(t, "slicePrimitiveField", 11, []string{"some", "other", "values"}, 11, out.NestedModelOptionalField.Value.SlicePrimitiveField)
	assertNodeField(t, "sliceRequiredPrimitiveField", 12, []string{"a", "b", "c"}, 12, out.NestedModelOptionalField.Value.SliceRequiredPrimitiveField)

	assertModelNodeField(t, "sliceNestedModelField", 13, 14, out.SliceNestedModelField)
	assertNodeField(t, "slicePrimitiveField", 14, []string{"d", "e", "f"}, 14, out.SliceNestedModelField.Value[0].SlicePrimitiveField)
	assertNodeField(t, "sliceRequiredPrimitiveField", 15, []string{"g", "h", "i"}, 15, out.SliceNestedModelField.Value[0].SliceRequiredPrimitiveField)
	assertNodeField(t, "slicePrimitiveField", 16, []string{"j", "k", "l"}, 16, out.SliceNestedModelField.Value[1].SlicePrimitiveField)
	assertNodeField(t, "sliceRequiredPrimitiveField", 17, []string{"m", "n", "o"}, 17, out.SliceNestedModelField.Value[1].SliceRequiredPrimitiveField)

	assertModelNodeField(t, "mapRequiredNestedModelField", 18, 19, out.MapRequiredNestedModelField)
	assertNodeField(t, "slicePrimitiveField", 20, []string{"p", "q", "r"}, 20, out.MapRequiredNestedModelField.Value.GetOrZero("z").SlicePrimitiveField)
	assertNodeField(t, "sliceRequiredPrimitiveField", 21, []string{"s", "t", "u"}, 21, out.MapRequiredNestedModelField.Value.GetOrZero("z").SliceRequiredPrimitiveField)
	xTestExtensionNodeMapRequiredNestedModelField := testutils.CreateStringYamlNode("some-value", 22, 13)
	assert.Equal(t, sequencedmap.New(sequencedmap.NewElem("x-test", Node[Extension]{
		Key:       "x-test",
		KeyNode:   testutils.CreateStringYamlNode("x-test", 22, 5),
		Value:     xTestExtensionNodeMapRequiredNestedModelField,
		ValueNode: xTestExtensionNodeMapRequiredNestedModelField,
	})), out.MapRequiredNestedModelField.Value.GetOrZero("z").Extensions)
	assertNodeField(t, "slicePrimitiveField", 24, []string{"w", "x", "y"}, 24, out.MapRequiredNestedModelField.Value.GetOrZero("x").SlicePrimitiveField)
	assertNodeField(t, "sliceRequiredPrimitiveField", 25, []string{"1", "2", "3"}, 25, out.MapRequiredNestedModelField.Value.GetOrZero("x").SliceRequiredPrimitiveField)

	xTestExtensionNode := testutils.CreateStringYamlNode("some-value-2", 26, 11)
	assert.Equal(t, sequencedmap.New(sequencedmap.NewElem("x-test-2", Node[Extension]{
		Key:       "x-test-2",
		KeyNode:   testutils.CreateStringYamlNode("x-test-2", 26, 1),
		Value:     xTestExtensionNode,
		ValueNode: xTestExtensionNode,
	})), out.Extensions)
}

func assertNodeField[T any](t *testing.T, expectedKey string, expectedKeyLine int, expectedValue any, expectedValueLine int, actual Node[T]) {
	assert.Equal(t, expectedKey, actual.Key)
	assert.Equal(t, expectedKeyLine, actual.KeyNode.Line)
	assert.Equal(t, expectedValue, actual.Value)
	assert.Equal(t, expectedValueLine, actual.ValueNode.Line)
}

func assertModelNodeField[T any](t *testing.T, expectedKey string, expectedKeyLine int, expectedValueLine int, actual Node[T]) {
	assert.Equal(t, expectedKey, actual.Key)
	assert.Equal(t, expectedKeyLine, actual.KeyNode.Line)
	assert.Equal(t, expectedValueLine, actual.ValueNode.Line)
}
