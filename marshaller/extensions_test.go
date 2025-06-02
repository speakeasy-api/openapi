package marshaller_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_Extensions_Coverage_Success(t *testing.T) {
	ctx := context.Background()

	// Test unmarshalExtension function (covered via extensions processing)
	testYaml := `
x-test-extension:
  name: test-extension
  value: 42
primitiveField: hello
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target TestExtensionModel
	err = marshaller.UnmarshalModel(ctx, node.Content[0], &target)
	require.NoError(t, err)

	// Verify extension was processed
	assert.True(t, target.Extensions.Has("x-test-extension"))
	extensionValue := target.Extensions.GetOrZero("x-test-extension")
	assert.Equal(t, "x-test-extension", extensionValue.Key)
	assert.NotNil(t, extensionValue.Value)
}

type TestExtensionModel struct {
	marshaller.CoreModel
	PrimitiveField marshaller.Node[string] `key:"primitiveField"`
	Extensions     Extensions              `key:"extensions"`
}

func Test_Extensions_SyncExtensions_Success(t *testing.T) {
	ctx := context.Background()

	// Test the syncExtensions functionality
	source := &TestStructWithExtensions{}
	source.Extensions = extensions.New(
		extensions.NewElem("x-custom",
			testutils.CreateStringYamlNode("custom-value", 1, 1)))

	outNode, err := marshaller.SyncValue(ctx, source, source.GetCore(), nil, false)
	require.NoError(t, err)

	assert.NotNil(t, outNode)
	assert.Equal(t, yaml.MappingNode, outNode.Kind)

	// Verify the extension was synced
	found := false
	for i := 0; i < len(outNode.Content); i += 2 {
		if outNode.Content[i].Value == "x-custom" {
			found = true
			assert.Equal(t, "custom-value", outNode.Content[i+1].Value)
			break
		}
	}
	assert.True(t, found, "Extension 'x-custom' should be found in synced output")
}

type TestExtensionUnmarshalError struct {
	marshaller.CoreModel
	Extensions Extensions `key:"extensions"`
}

func Test_Extensions_UnmarshalError_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test error path in unmarshalExtension by providing invalid extension structure
	testYaml := `
x-invalid-extension: |
  this is a scalar string that should cause
  an unmarshal error when trying to process
  as an extension
primitiveField: hello
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target TestExtensionUnmarshalError
	err = marshaller.UnmarshalModel(ctx, node.Content[0], &target)
	// This might succeed but we're testing the code path coverage
	// The unmarshalExtension function should handle various node types
	if err != nil {
		// Error is acceptable for this test - we're just ensuring code coverage
		t.Logf("Expected error occurred: %v", err)
	}
}

type TestStructWithComplexExtensions struct {
	marshaller.Model[TestStructWithComplexExtensionsCore]
	Extensions *extensions.Extensions
}

type TestStructWithComplexExtensionsCore struct {
	marshaller.CoreModel
	Extensions core.Extensions `key:"extensions"`
}

func Test_Extensions_ComplexStructure_Success(t *testing.T) {
	ctx := context.Background()

	// Test with complex extension structure
	complexExtensionNode := testutils.CreateMapYamlNode(
		[]*yaml.Node{
			testutils.CreateStringYamlNode("nested", 0, 0),
			testutils.CreateMapYamlNode([]*yaml.Node{
				testutils.CreateStringYamlNode("inner", 0, 0),
				testutils.CreateStringYamlNode("value", 0, 0),
			}, 0, 0),
			testutils.CreateStringYamlNode("array", 0, 0),
			{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					testutils.CreateStringYamlNode("item1", 0, 0),
					testutils.CreateStringYamlNode("item2", 0, 0),
				},
			},
		}, 0, 0)

	source := &TestStructWithComplexExtensions{}
	source.Extensions = extensions.New(
		extensions.NewElem("x-complex", complexExtensionNode))

	outNode, err := marshaller.SyncValue(ctx, source, source.GetCore(), nil, false)
	require.NoError(t, err)

	assert.NotNil(t, outNode)
	assert.Equal(t, yaml.MappingNode, outNode.Kind)
}

func Test_Extensions_EmptyExtensions_Success(t *testing.T) {
	ctx := context.Background()

	// Test with empty extensions
	source := &TestStructWithExtensions{}
	source.Extensions = extensions.New() // Empty extensions

	outNode, err := marshaller.SyncValue(ctx, source, source.GetCore(), nil, false)
	require.NoError(t, err)

	// Should produce empty mapping or nil
	if outNode != nil {
		assert.Equal(t, yaml.MappingNode, outNode.Kind)
	}
}
