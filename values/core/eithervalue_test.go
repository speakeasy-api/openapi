package core

import (
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type TestEitherValue[L any, R any] struct {
	Left  *L
	Right *R
}

func TestEitherValue_SyncChanges_Success(t *testing.T) {
	ctx := context.Background()

	source := TestEitherValue[string, string]{
		Left: pointer.From("some-value"),
	}
	var target EitherValue[string, string]
	outNode, err := marshaller.SyncValue(ctx, source, &target, nil, false)
	require.NoError(t, err)
	assert.Equal(t, testutils.CreateStringYamlNode("some-value", 0, 0), outNode)
	assert.Equal(t, "some-value", target.Left.GetValue())
	assert.False(t, target.IsRight)
}

func TestEitherValue_Unmarshal_BooleanValue_Success(t *testing.T) {
	// Test case that reproduces the additionalProperties: false issue
	// This should unmarshal as a boolean (Right type) when Left type (complex object) fails with validation errors
	ctx := context.Background()

	// Create a simple struct for Left type that would fail validation on a boolean
	type ComplexType struct {
		marshaller.CoreModel
		Name marshaller.Node[string] `key:"name" required:"true"`
	}

	// YAML with just a boolean value
	testYaml := `false`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target EitherValue[ComplexType, bool]
	validationErrs, err := marshaller.UnmarshalCore(ctx, node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")

	// Should have chosen the Right type (bool)
	assert.True(t, target.IsRight, "Should have chosen Right type (bool)")
	assert.False(t, target.IsLeft, "Should not have chosen Left type (ComplexType)")
	assert.Equal(t, false, target.Right.Value, "Should have unmarshaled boolean value correctly")
}

// TestEitherValue_BothTypesFailValidation tests the case where both Left and Right types
// fail with validation errors (not unmarshalling errors). In this case, the EitherValue
// should return the combined validation errors instead of an unmarshalling error.
func TestEitherValue_BothTypesFailValidation(t *testing.T) {
	ctx := context.Background()

	// Test case that reproduces the items array issue from burgershop.openapi-modified.yaml
	// An array cannot be unmarshalled into either a string (expects scalar) or bool (expects scalar)
	testYaml := `
- $ref: '#/components/schemas/Drink'
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	// Create an EitherValue[string, bool] to test the logic
	// An array should fail validation for both string (expects scalar) and bool (expects scalar)
	var target EitherValue[string, bool]
	validationErrs, err := marshaller.UnmarshalCore(ctx, node.Content[0], &target)

	// Should NOT have an unmarshalling error - this is the key fix
	require.NoError(t, err, "Should not have unmarshalling errors when both types fail validation")

	// Should have validation errors instead
	require.NotEmpty(t, validationErrs, "Should have validation errors")

	// Should not have set either Left or Right as successful
	assert.False(t, target.IsLeft, "Should not have set Left as successful")
	assert.False(t, target.IsRight, "Should not have set Right as successful")

	// Validation errors should contain type mismatch information for both types
	foundScalarError := false
	for _, validationErr := range validationErrs {
		if strings.Contains(validationErr.Error(), "expected scalar") {
			foundScalarError = true
			break
		}
	}
	assert.True(t, foundScalarError, "Should contain type mismatch error for scalar types")
}
