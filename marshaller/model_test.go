package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestModel_GetPropertyLine_Success tests the GetPropertyLine method with valid inputs
func TestModel_GetPropertyLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *marshaller.Model[core.TestPrimitiveModel]
		prop     string
		expected int
	}{
		{
			name: "property with key node returns line number",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				keyNode := &yaml.Node{Line: 42}
				coreModel := core.TestPrimitiveModel{
					StringField: marshaller.Node[string]{
						KeyNode: keyNode,
						Key:     "stringField",
						Value:   "testValue",
						Present: true,
					},
				}
				model := &marshaller.Model[core.TestPrimitiveModel]{
					Valid: true,
				}
				model.SetCore(&coreModel)
				return model
			},
			prop:     "StringField",
			expected: 42,
		},
		{
			name: "property with nil key node returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				coreModel := core.TestPrimitiveModel{
					StringField: marshaller.Node[string]{
						KeyNode: nil,
						Key:     "stringField",
						Value:   "testValue",
						Present: true,
					},
				}
				model := &marshaller.Model[core.TestPrimitiveModel]{
					Valid: true,
				}
				model.SetCore(&coreModel)
				return model
			},
			prop:     "StringField",
			expected: -1,
		},
		{
			name: "bool field with key node returns line number",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				keyNode := &yaml.Node{Line: 15}
				coreModel := core.TestPrimitiveModel{
					BoolField: marshaller.Node[bool]{
						KeyNode: keyNode,
						Key:     "boolField",
						Value:   true,
						Present: true,
					},
				}
				model := &marshaller.Model[core.TestPrimitiveModel]{
					Valid: true,
				}
				model.SetCore(&coreModel)
				return model
			},
			prop:     "BoolField",
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model := tt.setup()
			actual := model.GetPropertyLine(tt.prop)
			assert.Equal(t, tt.expected, actual, "line number should match expected value")
		})
	}
}

// TestModel_GetPropertyLine_Error tests the GetPropertyLine method with error conditions
func TestModel_GetPropertyLine_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *marshaller.Model[core.TestPrimitiveModel]
		prop     string
		expected int
	}{
		{
			name: "nil model returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return nil
			},
			prop:     "StringField",
			expected: -1,
		},
		{
			name: "non-existent property returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return &marshaller.Model[core.TestPrimitiveModel]{}
			},
			prop:     "NonExistentField",
			expected: -1,
		},
		{
			name: "property that is not a Node returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				coreModel := core.TestPrimitiveModel{
					CoreModel: marshaller.CoreModel{}, // This field doesn't implement GetKeyNode
				}
				model := &marshaller.Model[core.TestPrimitiveModel]{
					Valid: true,
				}
				model.SetCore(&coreModel)
				return model
			},
			prop:     "CoreModel",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model := tt.setup()
			actual := model.GetPropertyLine(tt.prop)
			assert.Equal(t, tt.expected, actual, "should return -1 for error conditions")
		})
	}
}

// TestModel_GetPropertyLine_ComplexModel tests with complex model types
func TestModel_GetPropertyLine_ComplexModel_Success(t *testing.T) {
	t.Parallel()

	keyNode := &yaml.Node{Line: 25}
	coreModel := core.TestComplexModel{
		ArrayField: marshaller.Node[[]string]{
			KeyNode: keyNode,
			Key:     "arrayField",
			Value:   []string{"item1", "item2"},
			Present: true,
		},
	}

	model := &marshaller.Model[core.TestComplexModel]{
		Valid: true,
	}
	model.SetCore(&coreModel)

	actual := model.GetPropertyLine("ArrayField")
	assert.Equal(t, 25, actual, "should return line number for array field")
}
