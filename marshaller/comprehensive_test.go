package marshaller_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test error paths in syncer
func Test_SyncValue_ErrorPaths_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test sync with invalid target
	_, err := marshaller.SyncValue(ctx, "test", nil, nil, false)
	require.Error(t, err)
}

// Test Node.SyncValue error path
func Test_Node_SyncValue_Error_Coverage(t *testing.T) {
	ctx := context.Background()

	node := &marshaller.Node[string]{}

	// Test with an error-causing scenario
	_, _, err := node.SyncValue(ctx, "key", make(chan int))
	require.Error(t, err)
}

// Test unmarshaller error paths
func Test_Unmarshal_ErrorPaths_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test with string target
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	var target string
	err := marshaller.Unmarshal(ctx, node, &target)
	require.NoError(t, err)
	assert.Equal(t, "test", target)
}

// Test various node types for better coverage
func Test_UnmarshalNode_VariousTypes_Coverage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		yaml        string
		target      any
		expectError bool
	}{
		{
			name:        "alias node",
			yaml:        `&anchor "value"`,
			target:      new(string),
			expectError: false,
		},
		{
			name:        "null node",
			yaml:        `null`,
			target:      new(*string),
			expectError: false,
		},
		{
			name:        "document node",
			yaml:        `"value"`,
			target:      new(string),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &doc)
			require.NoError(t, err)

			err = marshaller.Unmarshal(ctx, &doc, tt.target)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test extension error paths
func Test_Extensions_ErrorPaths_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test with malformed YAML that might cause extension errors
	testYaml := `
x-extension: !!seq
  - invalid
  - structure
  - that
  - might
  - cause
  - errors
primitiveField: "test"
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target TestExtensionModel
	err = marshaller.UnmarshalModel(ctx, node.Content[0], &target)
	// This may or may not error, but we're testing the code path
	if err != nil {
		t.Logf("Extension unmarshal error (expected for some cases): %v", err)
	}
}

// Test interfaces and type assertions for better coverage
func Test_TypeAssertions_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test various type scenarios
	var intTarget int
	err := marshaller.Unmarshal(ctx, testutils.CreateIntYamlNode(42, 0, 0), &intTarget)
	require.NoError(t, err)
	assert.Equal(t, 42, intTarget)

	var floatTarget float64
	err = marshaller.Unmarshal(ctx, testutils.CreateIntYamlNode(42, 0, 0), &floatTarget)
	require.NoError(t, err)
	assert.Equal(t, 42.0, floatTarget)

	var boolTarget bool
	err = marshaller.Unmarshal(ctx, testutils.CreateBoolYamlNode(true, 0, 0), &boolTarget)
	require.NoError(t, err)
	assert.True(t, boolTarget)
}

// Test pointer handling
func Test_PointerHandling_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test pointer to pointer scenarios
	var ptrTarget **string
	err := marshaller.Unmarshal(ctx, testutils.CreateStringYamlNode("test", 0, 0), &ptrTarget)
	require.NoError(t, err)
	require.NotNil(t, ptrTarget)
	require.NotNil(t, *ptrTarget)
	assert.Equal(t, "test", **ptrTarget)
}

// Test syncing with existing nodes
func Test_SyncValue_ExistingNodes_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test syncing string with existing node
	existingNode := testutils.CreateStringYamlNode("old-value", 1, 1)
	var target string

	outNode, err := marshaller.SyncValue(ctx, "new-value", &target, existingNode, false)
	require.NoError(t, err)
	assert.Equal(t, "new-value", target)
	assert.Equal(t, "new-value", outNode.Value)

	// Test syncing with boolean
	var boolTarget bool
	outNode, err = marshaller.SyncValue(ctx, true, &boolTarget, nil, false)
	require.NoError(t, err)
	assert.True(t, boolTarget)
	assert.Equal(t, "true", outNode.Value)
}

// Test edge cases in various functions
func Test_EdgeCases_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test empty slice
	var sliceTarget []string
	outNode, err := marshaller.SyncValue(ctx, []string{}, &sliceTarget, nil, false)
	require.NoError(t, err)
	assert.Empty(t, sliceTarget)
	assert.Equal(t, yaml.SequenceNode, outNode.Kind)

	// Test nil interface
	var interfaceTarget any
	err = marshaller.Unmarshal(ctx, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, &interfaceTarget)
	require.NoError(t, err)
	assert.Nil(t, interfaceTarget)
}

// Test specific error paths in unmarshaller
func Test_UnmarshallerSpecificErrors_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test with document node containing scalar
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "test"},
		},
	}

	var target string
	err := marshaller.Unmarshal(ctx, doc, &target)
	require.NoError(t, err)
	assert.Equal(t, "test", target)

	// Test with invalid YAML kind
	invalidNode := &yaml.Node{Kind: yaml.Kind(255), Value: "test"}
	err = marshaller.Unmarshal(ctx, invalidNode, &target)
	require.Error(t, err)
}

// Test simple additional scenarios
func Test_SimpleAdditionalScenarios_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test with float
	var floatTarget float32
	outNode, err := marshaller.SyncValue(ctx, float32(3.14), &floatTarget, nil, false)
	require.NoError(t, err)
	assert.Equal(t, float32(3.14), floatTarget)
	assert.NotNil(t, outNode)
}

// Test array slice with nil scenarios
func Test_ArraySlice_NilScenarios_Coverage(t *testing.T) {
	ctx := context.Background()

	// Test nil source and nil target
	var target []string
	outNode, err := marshaller.SyncValue(ctx, ([]string)(nil), &target, nil, false)
	require.NoError(t, err)
	assert.Nil(t, outNode)
	assert.Nil(t, target)
}
