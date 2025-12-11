package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func parseYAML(t *testing.T, yml string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)
	return &node
}

func TestNewPaths_Success(t *testing.T) {
	t.Parallel()

	paths := NewPaths()
	require.NotNil(t, paths, "NewPaths should return a non-nil paths")
	require.NotNil(t, paths.Map, "paths.Map should be initialized")
	assert.Equal(t, 0, paths.Len(), "newly created paths should be empty")
}

func TestNewPathItem_Success(t *testing.T) {
	t.Parallel()

	pathItem := NewPathItem()
	require.NotNil(t, pathItem, "NewPathItem should return a non-nil path item")
	require.NotNil(t, pathItem.Map, "pathItem.Map should be initialized")
	assert.Equal(t, 0, pathItem.Len(), "newly created path item should be empty")
}

func TestPaths_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns key node when path exists - first path",
			yaml: `
/pets:
  get:
    summary: List all pets
/users:
  get:
    summary: List all users
`,
			key: "/pets",
		},
		{
			name: "returns key node when path exists - second path",
			yaml: `
/pets:
  get:
    summary: List all pets
/users:
  get:
    summary: List all users
`,
			key: "/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var paths Paths
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &paths)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := paths.GetRootNode()
			result := paths.GetMapKeyNodeOrRoot(tt.key, rootNode)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tt.key, result.Value, "should return correct key node")
		})
	}
}

func TestPaths_GetMapKeyNodeOrRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root when path not found",
			yaml: `
/pets:
  get:
    summary: List all pets
`,
			key: "/nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var paths Paths
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &paths)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := paths.GetRootNode()
			result := paths.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, rootNode, result, "should return root node when key not found")
		})
	}
}

func TestPaths_GetMapKeyNodeOrRoot_Uninitialized(t *testing.T) {
	t.Parallel()

	t.Run("returns root when paths is not initialized", func(t *testing.T) {
		t.Parallel()
		var paths Paths
		rootNode := &yaml.Node{Kind: yaml.MappingNode, Line: 1}
		result := paths.GetMapKeyNodeOrRoot("/pets", rootNode)
		assert.Equal(t, rootNode, result, "should return root node when not initialized")
	})

	t.Run("returns root when RootNode is nil", func(t *testing.T) {
		t.Parallel()
		paths := &Paths{}
		paths.SetValid(true, true)
		rootNode := &yaml.Node{Kind: yaml.MappingNode, Line: 1}
		result := paths.GetMapKeyNodeOrRoot("/pets", rootNode)
		assert.Equal(t, rootNode, result, "should return root node when RootNode is nil")
	})
}

func TestPaths_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedLine int
	}{
		{
			name: "returns line number when path exists",
			yaml: `
/pets:
  get:
    summary: List all pets
`,
			key:          "/pets",
			expectedLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var paths Paths
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &paths)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := paths.GetRootNode()
			line := paths.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, line, "should return correct line number")
		})
	}
}

func TestPaths_GetMapKeyNodeOrRootLine_NilNode(t *testing.T) {
	t.Parallel()

	t.Run("returns -1 when GetMapKeyNodeOrRoot returns nil", func(t *testing.T) {
		t.Parallel()
		var paths Paths
		line := paths.GetMapKeyNodeOrRootLine("/pets", nil)
		assert.Equal(t, -1, line, "should return -1 when node is nil")
	})
}

func TestPathItem_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns key node when method exists - first method",
			yaml: `
get:
  summary: Get operation
post:
  summary: Post operation
`,
			key: "get",
		},
		{
			name: "returns key node when method exists - second method",
			yaml: `
get:
  summary: Get operation
post:
  summary: Post operation
`,
			key: "post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var pathItem PathItem
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &pathItem)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := pathItem.GetRootNode()
			result := pathItem.GetMapKeyNodeOrRoot(tt.key, rootNode)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tt.key, result.Value, "should return correct key node")
		})
	}
}

func TestPathItem_GetMapKeyNodeOrRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root when method not found",
			yaml: `
get:
  summary: Get operation
`,
			key: "delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var pathItem PathItem
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &pathItem)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := pathItem.GetRootNode()
			result := pathItem.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, rootNode, result, "should return root node when key not found")
		})
	}
}

func TestPathItem_GetMapKeyNodeOrRoot_Uninitialized(t *testing.T) {
	t.Parallel()

	t.Run("returns root when pathItem is not initialized", func(t *testing.T) {
		t.Parallel()
		var pathItem PathItem
		rootNode := &yaml.Node{Kind: yaml.MappingNode, Line: 1}
		result := pathItem.GetMapKeyNodeOrRoot("get", rootNode)
		assert.Equal(t, rootNode, result, "should return root node when not initialized")
	})

	t.Run("returns root when RootNode is nil", func(t *testing.T) {
		t.Parallel()
		pathItem := &PathItem{}
		pathItem.SetValid(true, true)
		rootNode := &yaml.Node{Kind: yaml.MappingNode, Line: 1}
		result := pathItem.GetMapKeyNodeOrRoot("get", rootNode)
		assert.Equal(t, rootNode, result, "should return root node when RootNode is nil")
	})
}

func TestPathItem_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedLine int
	}{
		{
			name: "returns line number when method exists",
			yaml: `
get:
  summary: Get operation
`,
			key:          "get",
			expectedLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var pathItem PathItem
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &pathItem)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := pathItem.GetRootNode()
			line := pathItem.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, line, "should return correct line number")
		})
	}
}

func TestPathItem_GetMapKeyNodeOrRootLine_NilNode(t *testing.T) {
	t.Parallel()

	t.Run("returns -1 when GetMapKeyNodeOrRoot returns nil", func(t *testing.T) {
		t.Parallel()
		var pathItem PathItem
		line := pathItem.GetMapKeyNodeOrRootLine("get", nil)
		assert.Equal(t, -1, line, "should return -1 when node is nil")
	})
}
