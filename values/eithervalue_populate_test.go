package values

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModel represents a simple model for testing (similar to Schema)
type TestModel struct {
	marshaller.Model[TestCoreModel]

	Name        *string
	Description *string
}

type TestCoreModel struct {
	marshaller.CoreModel `model:"testCoreModel"`

	Name        marshaller.Node[*string] `key:"name"`
	Description marshaller.Node[*string] `key:"description"`
}

// TestEitherValue_UnmarshalAndPopulate_RootNodePropagation tests the complete flow
// from YAML unmarshalling through population with a model on left and primitive on right
func TestEitherValue_UnmarshalAndPopulate_RootNodePropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		expectLeft   bool
		expectedLine int
	}{
		{
			name: "Model value (left side) with RootNode",
			yaml: `name: "test-model"
description: "A test model for validation"`,
			expectLeft:   true,
			expectedLine: 1,
		},
		{
			name:         "Boolean primitive (right side) with RootNode",
			yaml:         `true`,
			expectLeft:   false,
			expectedLine: 1,
		},
		{
			name: "Model with comment and specific line",
			yaml: `# This is a test model
name: "commented-model"
description: "Model with comments"`,
			expectLeft:   true,
			expectedLine: 2,
		},
		{
			name: "Boolean false with comment and specific line",
			yaml: `# Comment line 1
# Comment line 2
false`,
			expectLeft:   false,
			expectedLine: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use the complete marshaller.Unmarshal flow (UnmarshalCore + Populate)
			highLevelEither := &EitherValue[TestModel, TestCoreModel, bool, bool]{}

			validationErrs, err := marshaller.Unmarshal(t.Context(),
				strings.NewReader(tt.yaml), highLevelEither)

			require.NoError(t, err, "Unmarshal should not return an error")
			require.Empty(t, validationErrs, "Unmarshal should not return validation errors")

			// Verify RootNode propagation after complete unmarshal
			highLevelCore := highLevelEither.GetCore()
			require.NotNil(t, highLevelCore, "High-level model should have core set")

			actualRootNode := highLevelCore.GetRootNode()
			require.NotNil(t, actualRootNode, "High-level model should have RootNode set after unmarshal")
			assert.Equal(t, tt.expectedLine, actualRootNode.Line, "RootNode should have correct line number")

			// Verify the correct side is populated with correct values
			if tt.expectLeft {
				assert.True(t, highLevelCore.IsLeft, "Should be left side (model)")
				assert.False(t, highLevelCore.IsRight, "Should not be right side")
				assert.NotNil(t, highLevelEither.Left, "Left value (model) should be populated")
				assert.Nil(t, highLevelEither.Right, "Right value should be nil")

				// Verify the model has its own RootNode set
				model := highLevelEither.Left
				require.NotNil(t, model.GetRootNode(), "Model should have its own RootNode")
				assert.Equal(t, tt.expectedLine, model.GetRootNode().Line, "Model RootNode should have correct line")

				// Verify model fields are populated correctly
				if strings.Contains(tt.yaml, "test-model") {
					require.NotNil(t, model.Name, "Model Name should be set")
					assert.Equal(t, "test-model", *model.Name, "Model Name should match")
					require.NotNil(t, model.Description, "Model Description should be set")
					assert.Equal(t, "A test model for validation", *model.Description, "Model Description should match")
				} else if strings.Contains(tt.yaml, "commented-model") {
					require.NotNil(t, model.Name, "Model Name should be set")
					assert.Equal(t, "commented-model", *model.Name, "Model Name should match")
				}
			} else {
				assert.False(t, highLevelCore.IsLeft, "Should not be left side")
				assert.True(t, highLevelCore.IsRight, "Should be right side (primitive)")
				assert.NotNil(t, highLevelEither.Right, "Right value (primitive) should be populated")
				assert.Nil(t, highLevelEither.Left, "Left value should be nil")

				// Verify the actual boolean value
				expectedValue := parseExpectedBoolValue(tt.yaml)
				assert.Equal(t, expectedValue, *highLevelEither.Right, "Right value should match expected")
			}
		})
	}
}

// Helper function

func parseExpectedBoolValue(yamlStr string) bool {
	lines := strings.Split(strings.TrimSpace(yamlStr), "\n")

	// Find the last non-comment line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "#") && len(line) > 0 {
			return line == "true"
		}
	}
	return false
}
