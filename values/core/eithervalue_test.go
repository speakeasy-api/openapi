package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type TestEitherValue[L any, R any] struct {
	Left  *L
	Right *R
}

func TestEitherValue_SyncChanges_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

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
	t.Parallel()

	// Test case that reproduces the additionalProperties: false issue
	// This should unmarshal as a boolean (Right type) when Left type (complex object) fails with validation errors
	ctx := t.Context()

	// Create a simple struct for Left type that would fail validation on a boolean
	type ComplexType struct {
		marshaller.CoreModel `model:"complexType"`

		Name marshaller.Node[string] `key:"name" required:"true"`
	}

	// YAML with just a boolean value
	testYaml := `false`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target EitherValue[ComplexType, bool]
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err, "Should not have syntax errors")
	require.Empty(t, validationErrs, "Should not have validation errors")

	// Should have chosen the Right type (bool)
	assert.True(t, target.IsRight, "Should have chosen Right type (bool)")
	assert.False(t, target.IsLeft, "Should not have chosen Left type (ComplexType)")
	assert.False(t, target.Right.Value, "Should have unmarshaled boolean value correctly")
}

// TestEitherValue_BothTypesFailValidation tests the case where both Left and Right types
// fail with validation errors (not unmarshalling errors). In this case, the EitherValue
// should return the combined validation errors instead of an unmarshalling error.
func TestEitherValue_BothTypesFailValidation(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

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
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should NOT have an unmarshalling error - this is the key fix
	require.NoError(t, err, "Should not have unmarshalling errors when both types fail validation")

	// Should have validation errors instead
	require.NotEmpty(t, validationErrs, "Should have validation errors")

	// Should not have set either Left or Right as successful
	assert.False(t, target.IsLeft, "Should not have set Left as successful")
	assert.False(t, target.IsRight, "Should not have set Right as successful")

	// Validation errors should contain type mismatch information for both types
	foundTypeMismatchError := false
	for _, validationErr := range validationErrs {
		errStr := validationErr.Error()
		// Check for type mismatch patterns like "expected X, got Y"
		if (strings.Contains(errStr, "expected string") || strings.Contains(errStr, "expected bool")) &&
			strings.Contains(errStr, "got sequence") {
			foundTypeMismatchError = true
			break
		}
	}
	assert.True(t, foundTypeMismatchError, "Should contain type mismatch error for string/bool types")
}

func TestEitherValue_GetNavigableNode_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setup    func() *EitherValue[string, int]
		expected any
	}{
		{
			name: "left value set",
			setup: func() *EitherValue[string, int] {
				ev := &EitherValue[string, int]{
					IsLeft: true,
					Left:   marshaller.Node[string]{Value: "test-value"},
				}
				return ev
			},
			expected: marshaller.Node[string]{Value: "test-value"},
		},
		{
			name: "right value set",
			setup: func() *EitherValue[string, int] {
				ev := &EitherValue[string, int]{
					IsRight: true,
					Right:   marshaller.Node[int]{Value: 42},
				}
				return ev
			},
			expected: marshaller.Node[int]{Value: 42},
		},
		{
			name: "neither value set - returns right by default",
			setup: func() *EitherValue[string, int] {
				ev := &EitherValue[string, int]{
					IsLeft:  false,
					IsRight: false,
				}
				return ev
			},
			expected: marshaller.Node[int]{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ev := tt.setup()
			result, err := ev.GetNavigableNode()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEitherValue_Unmarshal_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "both types fail with unmarshal errors",
			yamlContent: `invalid: yaml: content`,
			expectError: true,
			errorMsg:    "unable to marshal into either",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &node)
			if err != nil {
				// Skip test if YAML itself is invalid
				t.Skip("Invalid YAML for this test")
				return
			}

			var target EitherValue[string, int]
			_, err = marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEitherValue_SyncChanges_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		model       any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "non-struct model",
			model:       "not a struct",
			expectError: true,
			errorMsg:    "expected struct, got string",
		},
		{
			name:        "both left and right nil",
			model:       TestEitherValue[string, int]{},
			expectError: true,
			errorMsg:    "EitherValue has neither Left nor Right set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			ev := &EitherValue[string, int]{}

			_, err := ev.SyncChanges(ctx, tt.model, nil)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEitherValue_SyncChanges_SideSwitching_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		initialSide    string
		newModel       TestEitherValue[string, int]
		expectedIsLeft bool
		expectedValue  any
	}{
		{
			name:        "switch from right to left",
			initialSide: "right",
			newModel: TestEitherValue[string, int]{
				Left: pointer.From("new-left-value"),
			},
			expectedIsLeft: true,
			expectedValue:  "new-left-value",
		},
		{
			name:        "switch from left to right",
			initialSide: "left",
			newModel: TestEitherValue[string, int]{
				Right: pointer.From(123),
			},
			expectedIsLeft: false,
			expectedValue:  123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			// Setup initial state
			ev := &EitherValue[string, int]{}
			if tt.initialSide == "left" {
				ev.IsLeft = true
				ev.Left = marshaller.Node[string]{Value: "initial-left"}
			} else {
				ev.IsRight = true
				ev.Right = marshaller.Node[int]{Value: 999}
			}

			// Sync with new model
			outNode, err := ev.SyncChanges(ctx, tt.newModel, nil)
			require.NoError(t, err)
			require.NotNil(t, outNode)

			// Verify the side switch
			assert.Equal(t, tt.expectedIsLeft, ev.IsLeft)
			assert.Equal(t, !tt.expectedIsLeft, ev.IsRight)

			// Verify the value
			if tt.expectedIsLeft {
				assert.Equal(t, tt.expectedValue, ev.Left.GetValue())
			} else {
				assert.Equal(t, tt.expectedValue, ev.Right.GetValue())
			}
		})
	}
}

func TestEitherValue_SyncChanges_RightSide_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	source := TestEitherValue[string, int]{
		Right: pointer.From(42),
	}
	var target EitherValue[string, int]
	outNode, err := marshaller.SyncValue(ctx, source, &target, nil, false)
	require.NoError(t, err)
	assert.NotNil(t, outNode)
	assert.Equal(t, 42, target.Right.GetValue())
	assert.True(t, target.IsRight)
	assert.False(t, target.IsLeft)
}

func TestHasTypeMismatchErrors_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		errors   []error
		expected bool
	}{
		{
			name:     "empty errors",
			errors:   []error{},
			expected: false,
		},
		{
			name:     "nil errors",
			errors:   nil,
			expected: false,
		},
		{
			name: "contains type mismatch error",
			errors: []error{
				validation.NewTypeMismatchError("", "expected string but got number"),
			},
			expected: true,
		},
		{
			name: "contains type mismatch error with parent name",
			errors: []error{
				validation.NewTypeMismatchError("", "expected object but received array"),
			},
			expected: true,
		},
		{
			name: "no type mismatch errors",
			errors: []error{
				errors.New("some other validation error"),
				errors.New("missing required field"),
			},
			expected: false,
		},
		{
			name: "plain errors without validation wrapper",
			errors: []error{
				errors.New("some error"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := hasTypeMismatchErrors("", tt.errors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEitherValue_Unmarshal_LeftUnmarshalError_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a scenario where Left type fails with unmarshal error but Right succeeds
	type FailingType struct {
		marshaller.CoreModel `model:"failingType"`
		// This will cause unmarshal errors with certain inputs
	}

	testYaml := `"simple string"`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target EitherValue[FailingType, string]
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Should have chosen the Right type (string)
	assert.True(t, target.IsRight)
	assert.False(t, target.IsLeft)
	assert.Equal(t, "simple string", target.Right.Value)
}

func TestEitherValue_Unmarshal_RightUnmarshalError_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a scenario where Right type fails with unmarshal error but Left succeeds
	type FailingType struct {
		marshaller.CoreModel `model:"failingType"`
		// This will cause unmarshal errors with certain inputs
	}

	testYaml := `"simple string"`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(testYaml), &node)
	require.NoError(t, err)

	var target EitherValue[string, FailingType]
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", node.Content[0], &target)

	// Should succeed without syntax errors
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Should have chosen the Left type (string)
	assert.True(t, target.IsLeft)
	assert.False(t, target.IsRight)
	assert.Equal(t, "simple string", target.Left.Value)
}
