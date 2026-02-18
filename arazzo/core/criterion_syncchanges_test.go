package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestCriterionTypeUnion_SyncChanges_WithStringType_Success(t *testing.T) {
	t.Parallel()

	yamlContent := "simple"
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	require.NoError(t, err, "unmarshal should succeed")

	var ctu CriterionTypeUnion
	validationErrs, err := ctu.Unmarshal(t.Context(), "test", node.Content[0])
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "validation errors should be empty")

	model := CriterionTypeUnion{
		Type: pointer.From("simple"),
	}

	resultNode, err := ctu.SyncChanges(t.Context(), model, node.Content[0])
	require.NoError(t, err, "SyncChanges should succeed")
	assert.NotNil(t, resultNode, "result node should not be nil")
}

func TestCriterionTypeUnion_SyncChanges_NonStruct_Error(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	err := yaml.Unmarshal([]byte("simple"), &node)
	require.NoError(t, err, "unmarshal should succeed")

	ctu := CriterionTypeUnion{}
	_, err = ctu.SyncChanges(t.Context(), "not a struct", node.Content[0])
	require.Error(t, err, "SyncChanges should fail")
	assert.Contains(t, err.Error(), "CriterionTypeUnion.SyncChanges expected a struct, got `string`", "error message should match")
}
