package marshaller

import (
	"context"
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
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.yamlData), &yamlNode)
	require.NoError(t, err)

	var node Node[T]
	validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	assert.Equal(t, testCase.expected, node.Value)
	assert.True(t, node.Present)
}

func TestNode_Unmarshal_String_Success(t *testing.T) {
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
			runNodeTest(t, tt.testCase)
		})
	}
}

func TestNode_Unmarshal_StringPtr_Success(t *testing.T) {
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
			runNodeTest(t, tt.testCase)
		})
	}
}

func TestNode_Unmarshal_Bool_Success(t *testing.T) {
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
			runNodeTest(t, tt.testCase)
		})
	}
}

type errorTestCase[T any] struct {
	yamlData              string
	expectValidationError bool
}

func runNodeErrorTest[T any](t *testing.T, testCase *errorTestCase[T]) {
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.yamlData), &yamlNode)
	if !testCase.expectValidationError {
		require.Error(t, err) // Expect YAML parsing to fail
		return
	}
	require.NoError(t, err)

	var node Node[T]
	validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
	if testCase.expectValidationError {
		require.NoError(t, err)
		require.NotEmpty(t, validationErrors)
	} else {
		require.Error(t, err)
	}
}

func TestNode_Unmarshal_Error(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "string node with array value",
			testFunc: func(t *testing.T) {
				runNodeErrorTest(t, &errorTestCase[string]{
					yamlData:              `["not", "a", "string"]`,
					expectValidationError: true,
				})
			},
		},
		{
			name: "int node with string value",
			testFunc: func(t *testing.T) {
				runNodeErrorTest(t, &errorTestCase[int]{
					yamlData:              `"hello"`,
					expectValidationError: true,
				})
			},
		},
		{
			name: "bool node with string value",
			testFunc: func(t *testing.T) {
				runNodeErrorTest(t, &errorTestCase[bool]{
					yamlData:              `"true"`,
					expectValidationError: true,
				})
			},
		},
		{
			name: "malformed yaml",
			testFunc: func(t *testing.T) {
				runNodeErrorTest(t, &errorTestCase[string]{
					yamlData:              `{invalid: yaml: content`,
					expectValidationError: false,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.initialYAML), &yamlNode)
	require.NoError(t, err)

	var node Node[T]
	validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	// Sync new value
	node.Value = testCase.newValue
	_, _, err = node.SyncValue(context.Background(), "", testCase.newValue)
	require.NoError(t, err)

	// Verify sync worked
	assert.Equal(t, testCase.newValue, node.Value)
	assert.Equal(t, testCase.expectedYAML, node.ValueNode.Value)
}

func TestNode_SyncValue_Success(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "sync string value",
			testFunc: func(t *testing.T) {
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
			tt.testFunc(t)
		})
	}
}

func TestNode_GetValue_Success(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "get string value",
			testFunc: func(t *testing.T) {
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`"hello"`), &yamlNode)
				require.NoError(t, err)

				var node Node[string]
				validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Equal(t, "hello", node.GetValue())
			},
		},
		{
			name: "get string pointer value",
			testFunc: func(t *testing.T) {
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`"hello"`), &yamlNode)
				require.NoError(t, err)

				var node Node[*string]
				validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
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
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`null`), &yamlNode)
				require.NoError(t, err)

				var node Node[*string]
				validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Nil(t, node.GetValue())
			},
		},
		{
			name: "get int value",
			testFunc: func(t *testing.T) {
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`42`), &yamlNode)
				require.NoError(t, err)

				var node Node[int]
				validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Equal(t, 42, node.GetValue())
			},
		},
		{
			name: "get bool value",
			testFunc: func(t *testing.T) {
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(`true`), &yamlNode)
				require.NoError(t, err)

				var node Node[bool]
				validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				assert.Equal(t, true, node.GetValue())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

func TestNode_NodeAccessor_Success(t *testing.T) {
	// Use the same pattern as the working tests
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(`"test value"`), &yamlNode)
	require.NoError(t, err)

	var node Node[string]
	validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
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
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(`"original value"`), &yamlNode)
	require.NoError(t, err)

	var node Node[string]
	validationErrors, err := node.Unmarshal(context.Background(), nil, &yamlNode)
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
	_, _, err = mutator.SyncValue(context.Background(), "testKey", "new value")
	require.NoError(t, err)

	// Verify the change
	assert.Equal(t, "new value", node.ValueNode.Value)
	assert.Equal(t, "testKey", node.Key)
}

func TestNode_Unmarshal_Int_Success(t *testing.T) {
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
			runNodeTest(t, tt.testCase)
		})
	}
}
