package jsonpointer_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	legacyyaml "gopkg.in/yaml.v3"
)

func TestGetTarget_LegacyYAMLNode(t *testing.T) {
	t.Parallel()

	yamlData := `info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
`
	var root legacyyaml.Node
	err := legacyyaml.Unmarshal([]byte(yamlData), &root)
	require.NoError(t, err)

	ptr := jsonpointer.JSONPointer("/info/title")
	target, err := jsonpointer.GetTarget(&root, ptr)
	require.NoError(t, err)
	require.NotNil(t, target)
	targetNode, ok := target.(*legacyyaml.Node)
	require.True(t, ok, "expected *legacyyaml.Node from pointer input, got %T", target)
	assert.Equal(t, legacyyaml.ScalarNode, targetNode.Kind)
	assert.Equal(t, "Test API", targetNode.Value)

	// Also test value struct version
	target2, err := jsonpointer.GetTarget(root, ptr)
	require.NoError(t, err)
	require.NotNil(t, target2)
	targetNode2, ok := target2.(*legacyyaml.Node)
	require.True(t, ok, "expected *legacyyaml.Node from value input, got %T", target2)
	assert.Equal(t, legacyyaml.ScalarNode, targetNode2.Kind)
	assert.Equal(t, "Test API", targetNode2.Value)
	assert.Equal(t, target, target2)
}
