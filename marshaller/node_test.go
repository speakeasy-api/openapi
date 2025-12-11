package marshaller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testCase[T any] struct {
	yamlData string
	expected T
}

func runNodeTest[T any](t *testing.T, testCase *testCase[T]) {
	t.Helper()
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.yamlData), &yamlNode)
	require.NoError(t, err)

	var node Node[T]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	assert.Equal(t, testCase.expected, node.Value)
	assert.True(t, node.Present)
}

func TestNode_Unmarshal_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testCase *testCase[string]
	}{
		{
			name: "basic string",
			testCase: &testCase[string]{
				yamlData: `"hello world"`,
				expected: "hello world",
			},
		},
		{
			name: "empty string",
			testCase: &testCase[string]{
				yamlData: `""`,
				expected: "",
			},
		},
		{
			name: "multiline string",
			testCase: &testCase[string]{
				yamlData: `"line1\nline2"`,
				expected: "line1\nline2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runNodeTest(t, tt.testCase)
		})
	}
}

func TestNode_Unmarshal_StringPtr_Success(t *testing.T) {
	t.Parallel()

	hello := "hello"
	tests := []struct {
		name     string
		testCase *testCase[*string]
	}{
		{
			name: "non-null string pointer",
			testCase: &testCase[*string]{
				yamlData: `"hello"`,
				expected: &hello,
			},
		},
		{
			name: "null string pointer",
			testCase: &testCase[*string]{
				yamlData: `null`,
				expected: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runNodeTest(t, tt.testCase)
		})
	}
}

func TestNode_Unmarshal_Bool_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testCase *testCase[bool]
	}{
		{
			name: "true value",
			testCase: &testCase[bool]{
				yamlData: `true`,
				expected: true,
			},
		},
		{
			name: "false value",
			testCase: &testCase[bool]{
				yamlData: `false`,
				expected: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runNodeTest(t, tt.testCase)
		})
	}
}

type errorTestCase[T any] struct {
	yamlData              string
	expectValidationError bool
}

func runNodeErrorTest[T any](t *testing.T, testCase *errorTestCase[T]) {
	t.Helper()
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.yamlData), &yamlNode)
	if !testCase.expectValidationError {
		require.Error(t, err) // Expect YAML parsing to fail
		return
	}
	require.NoError(t, err)

	var node Node[T]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	if testCase.expectValidationError {
		require.NoError(t, err)
		require.NotEmpty(t, validationErrors)
	} else {
		require.Error(t, err)
	}
}

func TestNode_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "string node with array value",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeErrorTest(t, &errorTestCase[string]{
					yamlData:              `["not", "a", "string"]`,
					expectValidationError: true,
				})
			},
		},
		{
			name: "int node with string value",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeErrorTest(t, &errorTestCase[int]{
					yamlData:              `"hello"`,
					expectValidationError: true,
				})
			},
		},
		{
			name: "bool node with string value",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeErrorTest(t, &errorTestCase[bool]{
					yamlData:              `"true"`,
					expectValidationError: true,
				})
			},
		},
		{
			name: "malformed yaml",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeErrorTest(t, &errorTestCase[string]{
					yamlData:              `{invalid: yaml: content`,
					expectValidationError: false,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.testFunc(t)
		})
	}
}

type syncTestCase[T any] struct {
	initialYAML  string
	newValue     T
	expectedYAML string
}

func runNodeSyncTest[T any](t *testing.T, testCase *syncTestCase[T]) {
	t.Helper()
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.initialYAML), &yamlNode)
	require.NoError(t, err)

	var node Node[T]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	// Sync new value
	node.Value = testCase.newValue
	_, _, err = node.SyncValue(t.Context(), "", testCase.newValue)
	require.NoError(t, err)

	// Verify sync worked
	assert.Equal(t, testCase.newValue, node.Value)
	assert.Equal(t, testCase.expectedYAML, node.ValueNode.Value)
}

func TestNode_SyncValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "sync string value",
			testFunc: func(t *testing.T) {
				t.Helper()
				t.Helper()
				runNodeSyncTest(t, &syncTestCase[string]{
					initialYAML:  `"old value"`,
					newValue:     "new value",
					expectedYAML: "new value",
				})
			},
		},
		{
			name: "sync int value",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeSyncTest(t, &syncTestCase[int]{
					initialYAML:  `42`,
					newValue:     100,
					expectedYAML: "100",
				})
			},
		},
		{
			name: "sync bool value true to false",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeSyncTest(t, &syncTestCase[bool]{
					initialYAML:  `true`,
					newValue:     false,
					expectedYAML: "false",
				})
			},
		},
		{
			name: "sync bool value false to true",
			testFunc: func(t *testing.T) {
				t.Helper()
				runNodeSyncTest(t, &syncTestCase[bool]{
					initialYAML:  `false`,
					newValue:     true,
					expectedYAML: "true",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.testFunc(t)
		})
	}
}

func TestNode_GetValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "get string value",
			testFunc: func(t *testing.T) {
				t.Helper()
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`"hello"`), &yamlNode)
				require.NoError(t, err)

				var node Node[string]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Equal(t, "hello", node.GetValue())
			},
		},
		{
			name: "get string pointer value",
			testFunc: func(t *testing.T) {
				t.Helper()
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`"hello"`), &yamlNode)
				require.NoError(t, err)

				var node Node[*string]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				// GetValue() returns the actual pointer, not the dereferenced value
				value := node.GetValue()
				require.NotNil(t, value)
				stringPtr, ok := value.(*string)
				require.True(t, ok)
				assert.Equal(t, "hello", *stringPtr)
			},
		},
		{
			name: "get null string pointer value",
			testFunc: func(t *testing.T) {
				t.Helper()
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`null`), &yamlNode)
				require.NoError(t, err)

				var node Node[*string]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Nil(t, node.GetValue())
			},
		},
		{
			name: "get int value",
			testFunc: func(t *testing.T) {
				t.Helper()
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`42`), &yamlNode)
				require.NoError(t, err)

				var node Node[int]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Equal(t, 42, node.GetValue())
			},
		},
		{
			name: "get bool value",
			testFunc: func(t *testing.T) {
				t.Helper()
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`true`), &yamlNode)
				require.NoError(t, err)

				var node Node[bool]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Equal(t, true, node.GetValue())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.testFunc(t)
		})
	}
}

func TestNode_NodeAccessor_Success(t *testing.T) {
	t.Parallel()

	// Use the same pattern as the working tests
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(`"test value"`), &yamlNode)
	require.NoError(t, err)

	var node Node[string]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	// Verify the node was populated correctly
	assert.True(t, node.Present)
	assert.Equal(t, "test value", node.Value)

	// Test NodeAccessor interface
	var accessor NodeAccessor = &node

	// Test GetValue
	value := accessor.GetValue()
	assert.Equal(t, "test value", value)

	// Test GetValueType
	valueType := accessor.GetValueType()
	assert.Equal(t, "string", valueType.String())

	// Test direct ValueNode access
	require.NotNil(t, node.ValueNode)
	// Note: ValueNode stores the document node, which may not have the parsed value directly
	// The parsed value is correctly stored in node.Value and accessible via GetValue()
	// This is expected behavior - syncing will handle updating the appropriate content nodes
}

func TestNode_NodeMutator_Success(t *testing.T) {
	t.Parallel()

	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(`"original value"`), &yamlNode)
	require.NoError(t, err)

	var node Node[string]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	// Test NodeMutator interface methods
	var mutator NodeMutator = &node

	// Test SetPresent
	mutator.SetPresent(false)
	assert.False(t, node.Present)
	mutator.SetPresent(true)
	assert.True(t, node.Present)

	// Test SyncValue
	_, _, err = mutator.SyncValue(t.Context(), "testKey", "new value")
	require.NoError(t, err)

	// Verify the change
	assert.Equal(t, "new value", node.ValueNode.Value)
	assert.Equal(t, "testKey", node.Key)
}

func TestNode_Unmarshal_Int_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testCase *testCase[int]
	}{
		{
			name: "positive int",
			testCase: &testCase[int]{
				yamlData: `42`,
				expected: 42,
			},
		},
		{
			name: "negative int",
			testCase: &testCase[int]{
				yamlData: `-10`,
				expected: -10,
			},
		},
		{
			name: "zero",
			testCase: &testCase[int]{
				yamlData: `0`,
				expected: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runNodeTest(t, tt.testCase)
		})
	}
}

func TestNode_GetKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}

	tests := []struct {
		name     string
		node     Node[string]
		expected *yaml.Node
	}{
		{
			name:     "not present returns root node",
			node:     Node[string]{Present: false, KeyNode: &yaml.Node{Line: 5}},
			expected: rootNode,
		},
		{
			name:     "present but nil key node returns root node",
			node:     Node[string]{Present: true, KeyNode: nil},
			expected: rootNode,
		},
		{
			name:     "present with key node returns key node",
			node:     Node[string]{Present: true, KeyNode: &yaml.Node{Line: 10}},
			expected: &yaml.Node{Line: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.node.GetKeyNodeOrRoot(rootNode)
			assert.Equal(t, tt.expected.Line, result.Line)
		})
	}
}

func TestNode_GetKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}

	tests := []struct {
		name     string
		node     Node[string]
		expected int
	}{
		{
			name:     "not present returns root line",
			node:     Node[string]{Present: false, KeyNode: &yaml.Node{Line: 5}},
			expected: 1,
		},
		{
			name:     "present with key node returns key line",
			node:     Node[string]{Present: true, KeyNode: &yaml.Node{Line: 10}},
			expected: 10,
		},
		{
			name:     "nil root node returns -1",
			node:     Node[string]{Present: false},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rn *yaml.Node
			if tt.name != "nil root node returns -1" {
				rn = rootNode
			}
			result := tt.node.GetKeyNodeOrRootLine(rn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNode_GetValueNode_Success(t *testing.T) {
	t.Parallel()

	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	node := Node[string]{ValueNode: valueNode}

	assert.Equal(t, valueNode, node.GetValueNode())
}

func TestNode_GetValueNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 5}

	tests := []struct {
		name         string
		node         Node[string]
		expectedLine int
	}{
		{
			name:         "not present returns root node",
			node:         Node[string]{Present: false, ValueNode: valueNode},
			expectedLine: 1,
		},
		{
			name:         "present but nil value node returns root node",
			node:         Node[string]{Present: true, ValueNode: nil},
			expectedLine: 1,
		},
		{
			name:         "present with value node returns value node",
			node:         Node[string]{Present: true, ValueNode: valueNode},
			expectedLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.node.GetValueNodeOrRoot(rootNode)
			assert.Equal(t, tt.expectedLine, result.Line)
		})
	}
}

func TestNode_GetValueNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 5}

	tests := []struct {
		name     string
		node     Node[string]
		expected int
	}{
		{
			name:     "not present returns root line",
			node:     Node[string]{Present: false, ValueNode: valueNode},
			expected: 1,
		},
		{
			name:     "present with value node returns value line",
			node:     Node[string]{Present: true, ValueNode: valueNode},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.node.GetValueNodeOrRootLine(rootNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNode_GetSliceValueNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}
	contentNode0 := &yaml.Node{Kind: yaml.ScalarNode, Line: 10, Value: "item0"}
	contentNode1 := &yaml.Node{Kind: yaml.ScalarNode, Line: 11, Value: "item1"}
	seqNode := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Line:    5,
		Content: []*yaml.Node{contentNode0, contentNode1},
	}

	tests := []struct {
		name         string
		node         Node[[]string]
		idx          int
		expectedLine int
	}{
		{
			name:         "not present returns root node",
			node:         Node[[]string]{Present: false, ValueNode: seqNode},
			idx:          0,
			expectedLine: 1,
		},
		{
			name:         "valid index returns content node",
			node:         Node[[]string]{Present: true, ValueNode: seqNode},
			idx:          0,
			expectedLine: 10,
		},
		{
			name:         "valid index 1 returns content node",
			node:         Node[[]string]{Present: true, ValueNode: seqNode},
			idx:          1,
			expectedLine: 11,
		},
		{
			name:         "negative index returns value node",
			node:         Node[[]string]{Present: true, ValueNode: seqNode},
			idx:          -1,
			expectedLine: 5,
		},
		{
			name:         "out of bounds index returns value node",
			node:         Node[[]string]{Present: true, ValueNode: seqNode},
			idx:          10,
			expectedLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.node.GetSliceValueNodeOrRoot(tt.idx, rootNode)
			assert.Equal(t, tt.expectedLine, result.Line)
		})
	}
}

func TestNode_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 10, Value: "key1"}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 10, Value: "val1"}
	mapNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Line:    5,
		Content: []*yaml.Node{keyNode, valNode},
	}

	tests := []struct {
		name         string
		node         Node[map[string]string]
		key          string
		expectedLine int
	}{
		{
			name:         "not present returns root node",
			node:         Node[map[string]string]{Present: false, ValueNode: mapNode},
			key:          "key1",
			expectedLine: 1,
		},
		{
			name:         "key found returns key node",
			node:         Node[map[string]string]{Present: true, ValueNode: mapNode},
			key:          "key1",
			expectedLine: 10,
		},
		{
			name:         "key not found returns value node",
			node:         Node[map[string]string]{Present: true, ValueNode: mapNode},
			key:          "nonexistent",
			expectedLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.node.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, result.Line)
		})
	}
}

func TestNode_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 10, Value: "key1"}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 10, Value: "val1"}
	mapNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Line:    5,
		Content: []*yaml.Node{keyNode, valNode},
	}

	node := Node[map[string]string]{Present: true, ValueNode: mapNode}

	assert.Equal(t, 10, node.GetMapKeyNodeOrRootLine("key1", rootNode))
	assert.Equal(t, 5, node.GetMapKeyNodeOrRootLine("nonexistent", rootNode))
}

func TestNode_GetMapValueNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	rootNode := &yaml.Node{Kind: yaml.DocumentNode, Line: 1}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 10, Value: "key1"}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Line: 11, Value: "val1"}
	mapNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Line:    5,
		Content: []*yaml.Node{keyNode, valNode},
	}

	tests := []struct {
		name         string
		node         Node[map[string]string]
		key          string
		expectedLine int
	}{
		{
			name:         "not present returns root node",
			node:         Node[map[string]string]{Present: false, ValueNode: mapNode},
			key:          "key1",
			expectedLine: 1,
		},
		{
			name:         "key found returns value node",
			node:         Node[map[string]string]{Present: true, ValueNode: mapNode},
			key:          "key1",
			expectedLine: 11,
		},
		{
			name:         "key not found returns value node",
			node:         Node[map[string]string]{Present: true, ValueNode: mapNode},
			key:          "nonexistent",
			expectedLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.node.GetMapValueNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, result.Line)
		})
	}
}

func TestNode_GetNavigableNode_Success(t *testing.T) {
	t.Parallel()

	node := Node[string]{Value: "test value"}
	result, err := node.GetNavigableNode()
	require.NoError(t, err)
	assert.Equal(t, "test value", result)
}
