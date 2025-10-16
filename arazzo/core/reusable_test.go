package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReusable_Unmarshal_WithReference_Success(t *testing.T) {
	t.Parallel()

	yamlContent := `reference: '#/components/parameters/userId'`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	require.NoError(t, err, "unmarshal should succeed")

	var reusable Reusable[*Parameter]
	validationErrs, err := reusable.Unmarshal(t.Context(), "test", node.Content[0])
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "validation errors should be empty")
	assert.True(t, reusable.GetValid(), "reusable should be valid")
	assert.True(t, reusable.Reference.Present, "reference should be present")
	assert.NotNil(t, reusable.Reference.Value, "reference value should not be nil")
}

func TestReusable_Unmarshal_NonMappingNode_Error(t *testing.T) {
	t.Parallel()

	yamlContent := "- item1\n- item2"

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	require.NoError(t, err, "unmarshal should succeed")

	var reusable Reusable[*Parameter]
	validationErrs, err := reusable.Unmarshal(t.Context(), "test", node.Content[0])
	require.NoError(t, err, "unmarshal error should be nil")
	require.NotEmpty(t, validationErrs, "validation errors should not be empty")
	assert.Contains(t, validationErrs[0].Error(), "reusable expected object", "error message should match")
	assert.False(t, reusable.GetValid(), "reusable should not be valid")
}

func TestReusable_SyncChanges_NonStruct_Error(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	err := yaml.Unmarshal([]byte(`reference: '#/test'`), &node)
	require.NoError(t, err, "unmarshal should succeed")

	reusable := Reusable[*Parameter]{}
	_, err = reusable.SyncChanges(t.Context(), "not a struct", node.Content[0])
	require.Error(t, err, "SyncChanges should fail")
	assert.Contains(t, err.Error(), "Reusable.SyncChanges expected a struct, got string", "error message should match")
}
