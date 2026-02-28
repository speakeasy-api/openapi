package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestModel_GetPropertyNode_Success tests the GetPropertyNode method with valid inputs
func TestModel_GetPropertyNode_Success(t *testing.T) {
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
			actual := model.GetPropertyNode(tt.prop)
			line := -1
			if actual != nil {
				line = actual.Line
			}
			assert.Equal(t, tt.expected, line, "line number should match expected value")
		})
	}
}

// TestModel_GetPropertyNode_Error tests the GetPropertyNode method with error conditions
func TestModel_GetPropertyNode_Error(t *testing.T) {
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
			actual := model.GetPropertyNode(tt.prop)
			if actual == nil {
				assert.Equal(t, tt.expected, -1, "should return -1 for error conditions")
			} else {
				assert.Equal(t, tt.expected, actual.Line, "line number should match expected value")
			}
		})
	}
}

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

func TestModel_GetCoreAny_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func() *marshaller.Model[core.TestPrimitiveModel]
		isNil bool
	}{
		{
			name: "non-nil model returns core",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return &marshaller.Model[core.TestPrimitiveModel]{}
			},
			isNil: false,
		},
		{
			name: "nil model returns nil",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return nil
			},
			isNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := tt.setup()
			result := model.GetCoreAny()
			if tt.isNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestModel_GetRootNodeLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *marshaller.Model[core.TestPrimitiveModel]
		expected int
	}{
		{
			name: "nil model returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return nil
			},
			expected: -1,
		},
		{
			name: "model with no root node returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return &marshaller.Model[core.TestPrimitiveModel]{}
			},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := tt.setup()
			result := model.GetRootNodeLine()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModel_GetRootNodeColumn_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *marshaller.Model[core.TestPrimitiveModel]
		expected int
	}{
		{
			name: "nil model returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return nil
			},
			expected: -1,
		},
		{
			name: "model with no root node returns -1",
			setup: func() *marshaller.Model[core.TestPrimitiveModel] {
				return &marshaller.Model[core.TestPrimitiveModel]{}
			},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := tt.setup()
			result := model.GetRootNodeColumn()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModel_Cache_Success(t *testing.T) {
	t.Parallel()

	t.Run("nil model GetCachedReferencedObject returns nil false", func(t *testing.T) {
		t.Parallel()
		var model *marshaller.Model[core.TestPrimitiveModel]
		result, ok := model.GetCachedReferencedObject("key")
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("nil model GetCachedReferenceDocument returns nil false", func(t *testing.T) {
		t.Parallel()
		var model *marshaller.Model[core.TestPrimitiveModel]
		result, ok := model.GetCachedReferenceDocument("key")
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("model with uninitialized cache returns nil false", func(t *testing.T) {
		t.Parallel()
		model := &marshaller.Model[core.TestPrimitiveModel]{}
		result, ok := model.GetCachedReferencedObject("key")
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("InitCache creates caches that work correctly", func(t *testing.T) {
		t.Parallel()
		model := &marshaller.Model[core.TestPrimitiveModel]{}
		model.InitCache()

		// Store and retrieve object
		model.StoreReferencedObjectInCache("objKey", "testObject")
		result, ok := model.GetCachedReferencedObject("objKey")
		assert.True(t, ok)
		assert.Equal(t, "testObject", result)

		// Store and retrieve document
		model.StoreReferenceDocumentInCache("docKey", []byte("testDoc"))
		doc, ok := model.GetCachedReferenceDocument("docKey")
		assert.True(t, ok)
		assert.Equal(t, []byte("testDoc"), doc)
	})
}
