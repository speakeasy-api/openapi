package jsonpointer

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestModel represents a simple model for testing YAML fallback
type TestModel struct {
	marshaller.Model[TestModelCore]

	KnownField string
}

type TestModelCore struct {
	marshaller.CoreModel `model:"testModelCore"`

	KnownField marshaller.Node[string] `key:"knownField"`
}

func TestNavigateModel_YAMLFallback_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		yamlContent  string
		jsonPointer  string
		expectedType string
		expectedVal  interface{}
	}{
		{
			name: "unknown field in YAML root",
			yamlContent: `
knownField: "known value"
unknownField: "unknown value"
`,
			jsonPointer:  "/unknownField",
			expectedType: "*yaml.Node",
			expectedVal:  "unknown value",
		},
		{
			name: "nested unknown field",
			yamlContent: `
knownField: "known value"
unknownObject:
  nestedField: "nested value"
`,
			jsonPointer:  "/unknownObject/nestedField",
			expectedType: "*yaml.Node",
			expectedVal:  "nested value",
		},
		{
			name: "unknown array field",
			yamlContent: `
knownField: "known value"
unknownArray:
  - "item1"
  - "item2"
`,
			jsonPointer:  "/unknownArray/1",
			expectedType: "*yaml.Node",
			expectedVal:  "item2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Parse YAML content
			var rootNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &rootNode)
			require.NoError(t, err, "failed to parse YAML")

			// Create test model with YAML node
			model := &TestModel{}
			model.GetCore().SetRootNode(&rootNode)

			// Test navigation
			pointer := JSONPointer(tt.jsonPointer)
			result, err := GetTarget(model, pointer)

			require.NoError(t, err, "navigation should succeed")
			require.NotNil(t, result, "result should not be nil")

			// Verify result type
			assert.Equal(t, tt.expectedType, getTypeName(result), "result type should match expected")

			// For YAML nodes, check the value
			if yamlNode, ok := result.(*yaml.Node); ok {
				assert.Equal(t, tt.expectedVal, yamlNode.Value, "YAML node value should match expected")
			}
		})
	}
}

func TestNavigateModel_YAMLFallback_KnownFieldStillWorks(t *testing.T) {
	t.Parallel()
	yamlContent := `
knownField: "known value"
unknownField: "unknown value"
`

	// Parse YAML content
	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &rootNode)
	require.NoError(t, err, "failed to parse YAML")

	// Create test model with YAML node
	model := &TestModel{
		KnownField: "known value",
	}
	model.GetCore().SetRootNode(&rootNode)

	// Test navigation to known field (should use struct field, not YAML fallback)
	pointer := JSONPointer("/knownField")
	result, err := GetTarget(model, pointer)

	require.NoError(t, err, "navigation should succeed")
	require.NotNil(t, result, "result should not be nil")

	// Should return the string field from the high-level model, not a YAML node
	assert.Equal(t, "string", getTypeName(result), "should return struct field, not YAML fallback")
}

func TestNavigateModel_YAMLFallback_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		yamlContent string
		jsonPointer string
	}{
		{
			name: "field not found anywhere",
			yamlContent: `
knownField: "known value"
`,
			jsonPointer: "/nonExistentField",
		},
		{
			name: "nested field not found",
			yamlContent: `
knownField: "known value"
existingObject:
  someField: "value"
`,
			jsonPointer: "/existingObject/nonExistentField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Parse YAML content
			var rootNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &rootNode)
			require.NoError(t, err, "failed to parse YAML")

			// Create test model with YAML node
			model := &TestModel{}
			model.GetCore().SetRootNode(&rootNode)

			// Test navigation
			pointer := JSONPointer(tt.jsonPointer)
			result, err := GetTarget(model, pointer)

			require.Error(t, err, "navigation should fail")
			assert.Nil(t, result, "result should be nil")
			assert.Contains(t, err.Error(), "not found", "error should indicate field not found")
		})
	}
}

func TestNavigateModel_YAMLFallback_NoYAMLNode(t *testing.T) {
	t.Parallel()
	// Create test model without YAML node
	model := &TestModel{}

	// Test navigation to unknown field
	pointer := JSONPointer("/unknownField")
	result, err := GetTarget(model, pointer)

	require.Error(t, err, "navigation should fail when no YAML node available")
	assert.Nil(t, result, "result should be nil")
	assert.Contains(t, err.Error(), "not found", "error should indicate field not found")
}

// Helper function to get type name for assertions
func getTypeName(v interface{}) string {
	if v == nil {
		return "nil"
	}
	return reflect.TypeOf(v).String()
}
