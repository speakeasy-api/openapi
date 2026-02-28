package jsonpointer

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
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
			expectedType: "*libyaml.Node",
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
			expectedType: "*libyaml.Node",
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
			expectedType: "*libyaml.Node",
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

// TestNavigateModel_YAMLFallback_KeyBasedStruct tests we can access the RootNode from Key based structs like our core models
// This validates the fix where GetTarget can navigate from a core model to additional YAML properties
func TestNavigateModel_YAMLFallback_KeyBasedStruct(t *testing.T) {
	t.Parallel()

	yamlContent := `
knownField: "test"
definitions:
  attachments:
    title: Attachments
    description: "Array of attachments"
    type: array
    items:
      type: object
`

	// Parse YAML content
	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &rootNode)
	require.NoError(t, err, "failed to parse YAML")

	// Create test model with YAML node
	model := &TestModel{}
	model.GetCore().SetRootNode(&rootNode)

	// Test navigation from the CORE MODEL (this was the bug - GetTarget couldn't find YAML properties from core model)
	coreModel := model.GetCore()

	// Test navigation to definitions key (not in core model)
	result, err := GetTarget(coreModel, "/definitions")
	require.NoError(t, err, "should find definitions in YAML from core model")
	require.NotNil(t, result, "result should not be nil")

	yamlNode, ok := result.(*yaml.Node)
	require.True(t, ok, "should return yaml.Node for definitions")
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind, "definitions should be a mapping node")

	// Test navigation deeper into definitions structure from core model
	result, err = GetTarget(coreModel, "/definitions/attachments/title")
	require.NoError(t, err, "should navigate through YAML structure from core model")
	require.NotNil(t, result, "result should not be nil")

	yamlNode, ok = result.(*yaml.Node)
	require.True(t, ok, "should return yaml.Node")
	assert.Equal(t, "Attachments", yamlNode.Value, "title value should match")
}
