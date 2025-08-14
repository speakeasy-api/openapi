package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestJSONSchema_Unmarshal_BooleanValue_Success(t *testing.T) {
	t.Parallel()

	// Test case that reproduces the additionalProperties: false issue
	// This should unmarshal as a boolean (Right type) when Left type (Schema) fails with validation errors
	ctx := t.Context()

	// YAML with just a boolean value (like additionalProperties: false)
	testYaml := `false`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	// Test the exact JSONSchema type structure
	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")

	// Should have chosen the Right type (bool)
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsRight, "JSONSchema should be Right type (bool)")
	assert.False(t, target.IsLeft, "JSONSchema should not be Left type (Schema)")
	assert.False(t, target.Right.Value, "JSONSchema should have unmarshaled boolean value correctly")
}

func TestJSONSchema_Unmarshal_SchemaObject_Success(t *testing.T) {
	t.Parallel()

	// Test case that ensures schema objects still work correctly
	ctx := t.Context()

	// YAML with a schema object
	testYaml := `
type: string
minLength: 1
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	// Test the exact JSONSchema type structure
	var target JSONSchema
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")

	// Should have chosen the Left type (Schema)
	require.NotNil(t, target, "JSONSchema should not be nil")
	assert.True(t, target.IsLeft, "JSONSchema should be Left type (Schema)")
	assert.False(t, target.IsRight, "JSONSchema should not be Right type (bool)")

	// Verify the schema was unmarshaled correctly
	require.NotNil(t, target.Left.Value.Type.Value, "Schema type should be set")
	assert.True(t, target.Left.Value.Type.Value.IsRight, "Type should be Right type (string)")
	assert.Equal(t, "string", target.Left.Value.Type.Value.Right.Value, "Type should be 'string'")
}
