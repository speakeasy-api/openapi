package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestPaths_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedLine int
	}{
		{
			name: "returns line number when path exists - first path",
			yaml: `
/pets:
  get:
    summary: List all pets
/users:
  get:
    summary: List all users
`,
			key:          "/pets",
			expectedLine: 2,
		},
		{
			name: "returns line number when path exists - second path",
			yaml: `
/pets:
  get:
    summary: List all pets
/users:
  get:
    summary: List all users
`,
			key:          "/users",
			expectedLine: 5,
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

func TestPaths_GetMapKeyNodeOrRootLine_ReturnsRootLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root line when path not found",
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
			line := paths.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, rootNode.Line, line, "should return root node line")
		})
	}
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
			name: "returns line number when method exists - first method",
			yaml: `
get:
  summary: Get operation
post:
  summary: Post operation
`,
			key:          "get",
			expectedLine: 2,
		},
		{
			name: "returns line number when method exists - second method",
			yaml: `
get:
  summary: Get operation
post:
  summary: Post operation
`,
			key:          "post",
			expectedLine: 4,
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

func TestPathItem_GetMapKeyNodeOrRootLine_ReturnsRootLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root line when method not found",
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
			line := pathItem.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, rootNode.Line, line, "should return root node line")
		})
	}
}
