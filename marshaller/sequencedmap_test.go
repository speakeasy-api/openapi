package marshaller_test

import (
	"slices"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// SequencedMap test case for successful operations
type sequencedMapTestCase[K comparable, V any] struct {
	yamlData     string
	expectedKeys []K
	expectedVals []V
}

// SequencedMap error test case
type sequencedMapErrorTestCase[K comparable, V any] struct {
	yamlData string
}

// Helper to run SequencedMap success tests
func runSequencedMapTest[K comparable, V any](t *testing.T, testCase *sequencedMapTestCase[K, V]) {
	t.Helper()
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.yamlData), &yamlNode)
	require.NoError(t, err)

	var node marshaller.Node[*sequencedmap.Map[K, V]]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	require.NoError(t, err)
	require.Empty(t, validationErrors)

	assert.True(t, node.Present)
	require.NotNil(t, node.Value)

	// Verify order preservation and values
	assert.Equal(t, len(testCase.expectedKeys), node.Value.Len())

	for i, expectedKey := range testCase.expectedKeys {
		value, ok := node.Value.Get(expectedKey)
		require.True(t, ok, "key %v should be present", expectedKey)
		assert.Equal(t, testCase.expectedVals[i], value)
	}

	// Verify order is preserved
	keys := slices.Collect(node.Value.Keys())
	if keys == nil {
		keys = []K{} // Convert nil to empty slice for comparison
	}
	assert.Equal(t, testCase.expectedKeys, keys)
}

// Helper to run SequencedMap error tests
func runSequencedMapErrorTest[K comparable, V any](t *testing.T, testCase *sequencedMapErrorTestCase[K, V]) {
	t.Helper()
	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(testCase.yamlData), &yamlNode)
	if err != nil {
		// Malformed YAML is expected for some error cases
		return
	}

	var node marshaller.Node[*sequencedmap.Map[K, V]]
	validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
	if len(validationErrors) > 0 {
		require.NotEmpty(t, validationErrors)
	} else {
		require.Error(t, err)
	}
}

func TestSequencedMap_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testCase *sequencedMapTestCase[string, string]
	}{
		{
			name: "simple string map",
			testCase: &sequencedMapTestCase[string, string]{
				yamlData: `
key1: "value1"
key2: "value2"
key3: "value3"
`,
				expectedKeys: []string{"key1", "key2", "key3"},
				expectedVals: []string{"value1", "value2", "value3"},
			},
		},
		{
			name: "empty map",
			testCase: &sequencedMapTestCase[string, string]{
				yamlData:     `{}`,
				expectedKeys: []string{},
				expectedVals: []string{},
			},
		},
		{
			name: "single entry map",
			testCase: &sequencedMapTestCase[string, string]{
				yamlData: `
onlyKey: "onlyValue"
`,
				expectedKeys: []string{"onlyKey"},
				expectedVals: []string{"onlyValue"},
			},
		},
		{
			name: "order preservation test",
			testCase: &sequencedMapTestCase[string, string]{
				yamlData: `
zebra: "last"
alpha: "first"
beta: "middle"
`,
				expectedKeys: []string{"zebra", "alpha", "beta"},
				expectedVals: []string{"last", "first", "middle"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runSequencedMapTest(t, tt.testCase)
		})
	}
}

func TestSequencedMap_Unmarshal_IntValues_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testCase *sequencedMapTestCase[string, int]
	}{
		{
			name: "string to int map",
			testCase: &sequencedMapTestCase[string, int]{
				yamlData: `
first: 1
second: 2
third: 3
`,
				expectedKeys: []string{"first", "second", "third"},
				expectedVals: []int{1, 2, 3},
			},
		},
		{
			name: "mixed int values",
			testCase: &sequencedMapTestCase[string, int]{
				yamlData: `
positive: 42
zero: 0
negative: -10
`,
				expectedKeys: []string{"positive", "zero", "negative"},
				expectedVals: []int{42, 0, -10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runSequencedMapTest(t, tt.testCase)
		})
	}
}

func TestSequencedMap_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testCase *sequencedMapErrorTestCase[string, string]
	}{
		{
			name: "array instead of map",
			testCase: &sequencedMapErrorTestCase[string, string]{
				yamlData: `["not", "a", "map"]`,
			},
		},
		{
			name: "scalar instead of map",
			testCase: &sequencedMapErrorTestCase[string, string]{
				yamlData: `"not a map"`,
			},
		},
		{
			name: "malformed yaml",
			testCase: &sequencedMapErrorTestCase[string, string]{
				yamlData: `{invalid: yaml: content`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runSequencedMapErrorTest(t, tt.testCase)
		})
	}
}

func TestSequencedMap_Sync_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "sync map modifications",
			testFunc: func(t *testing.T) {
				t.Helper()
				yamlData := `
original1: "value1"
original2: "value2"
`
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(yamlData), &yamlNode)
				require.NoError(t, err)

				var node marshaller.Node[*sequencedmap.Map[string, string]]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				// Modify existing value
				node.Value.Set("original1", "modified1")

				// Add new value
				node.Value.Set("new", "newValue")

				// Sync the changes
				_, _, err = node.SyncValue(t.Context(), "", node.Value)
				require.NoError(t, err)

				// Verify the changes
				val1, ok := node.Value.Get("original1")
				require.True(t, ok)
				assert.Equal(t, "modified1", val1)

				val2, ok := node.Value.Get("original2")
				require.True(t, ok)
				assert.Equal(t, "value2", val2)

				newVal, ok := node.Value.Get("new")
				require.True(t, ok)
				assert.Equal(t, "newValue", newVal)

				// Verify order is preserved (original keys first, then new)
				keys := slices.Collect(node.Value.Keys())
				assert.Equal(t, []string{"original1", "original2", "new"}, keys)
			},
		},
		{
			name: "sync map reordering",
			testFunc: func(t *testing.T) {
				t.Helper()
				yamlData := `
third: "3"
first: "1"
second: "2"
`
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(yamlData), &yamlNode)
				require.NoError(t, err)

				var node marshaller.Node[*sequencedmap.Map[string, string]]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				// Verify original order is preserved
				originalKeys := slices.Collect(node.Value.Keys())
				assert.Equal(t, []string{"third", "first", "second"}, originalKeys)

				// Sync should preserve the original order
				_, _, err = node.SyncValue(t.Context(), "", node.Value)
				require.NoError(t, err)

				// Order should still be preserved
				keys := slices.Collect(node.Value.Keys())
				assert.Equal(t, []string{"third", "first", "second"}, keys)
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

func TestSequencedMap_Population_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "populate from core sequenced map",
			testFunc: func(t *testing.T) {
				t.Helper()
				// Create a sequenced map
				sm := sequencedmap.New[string, string]()
				sm.Set("first", "1")
				sm.Set("second", "2")
				sm.Set("third", "3")

				// Verify population worked correctly
				assert.Equal(t, 3, sm.Len())

				val1, ok := sm.Get("first")
				require.True(t, ok)
				assert.Equal(t, "1", val1)

				val2, ok := sm.Get("second")
				require.True(t, ok)
				assert.Equal(t, "2", val2)

				val3, ok := sm.Get("third")
				require.True(t, ok)
				assert.Equal(t, "3", val3)

				// Verify order
				keys := slices.Collect(sm.Keys())
				assert.Equal(t, []string{"first", "second", "third"}, keys)
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

func TestSequencedMap_WithExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "sequenced map with extension keys",
			testFunc: func(t *testing.T) {
				t.Helper()
				yamlData := `
normalKey: "normal value"
x-extension: "extension value"
anotherKey: "another value"
x-vendor: "vendor extension"
`
				var yamlNode yaml.Node
				err := yaml.Unmarshal([]byte(yamlData), &yamlNode)
				require.NoError(t, err)

				var node marshaller.Node[*sequencedmap.Map[string, string]]
				validationErrors, err := node.Unmarshal(t.Context(), "", nil, &yamlNode)
				require.NoError(t, err)
				require.Empty(t, validationErrors)

				// Verify all keys are present including extensions
				assert.Equal(t, 4, node.Value.Len())

				normalVal, ok := node.Value.Get("normalKey")
				require.True(t, ok)
				assert.Equal(t, "normal value", normalVal)

				extVal, ok := node.Value.Get("x-extension")
				require.True(t, ok)
				assert.Equal(t, "extension value", extVal)

				anotherVal, ok := node.Value.Get("anotherKey")
				require.True(t, ok)
				assert.Equal(t, "another value", anotherVal)

				vendorVal, ok := node.Value.Get("x-vendor")
				require.True(t, ok)
				assert.Equal(t, "vendor extension", vendorVal)

				// Verify order is preserved
				keys := slices.Collect(node.Value.Keys())
				assert.Equal(t, []string{"normalKey", "x-extension", "anotherKey", "x-vendor"}, keys)
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
