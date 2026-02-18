package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestNewCallback_Success(t *testing.T) {
	t.Parallel()

	callback := NewCallback()
	require.NotNil(t, callback, "NewCallback should return a non-nil callback")
	require.NotNil(t, callback.Map, "callback.Map should be initialized")
	assert.Equal(t, 0, callback.Len(), "newly created callback should be empty")
}

func TestCallback_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns key node when expression exists - first expression",
			yaml: `
'{$request.body#/callbackUrl}':
  post:
    summary: Callback payload
'{$request.body#/statusUrl}':
  post:
    summary: Status callback
`,
			key: "{$request.body#/callbackUrl}",
		},
		{
			name: "returns key node when expression exists - second expression",
			yaml: `
'{$request.body#/callbackUrl}':
  post:
    summary: Callback payload
'{$request.body#/statusUrl}':
  post:
    summary: Status callback
`,
			key: "{$request.body#/statusUrl}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var callback Callback
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &callback)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := callback.GetRootNode()
			result := callback.GetMapKeyNodeOrRoot(tt.key, rootNode)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tt.key, result.Value, "should return correct key node")
		})
	}
}

func TestCallback_GetMapKeyNodeOrRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root when expression not found",
			yaml: `
'{$request.body#/callbackUrl}':
  post:
    summary: Callback payload
`,
			key: "{$request.body#/nonexistent}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var callback Callback
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &callback)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := callback.GetRootNode()
			result := callback.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, rootNode, result, "should return root node when key not found")
		})
	}
}

func TestCallback_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedLine int
	}{
		{
			name: "returns line number when expression exists - first expression",
			yaml: `
'{$request.body#/callbackUrl}':
  post:
    summary: Callback payload
'{$request.body#/statusUrl}':
  post:
    summary: Status callback
`,
			key:          "{$request.body#/callbackUrl}",
			expectedLine: 2,
		},
		{
			name: "returns line number when expression exists - second expression",
			yaml: `
'{$request.body#/callbackUrl}':
  post:
    summary: Callback payload
'{$request.body#/statusUrl}':
  post:
    summary: Status callback
`,
			key:          "{$request.body#/statusUrl}",
			expectedLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var callback Callback
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &callback)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := callback.GetRootNode()
			line := callback.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, line, "should return correct line number")
		})
	}
}

func TestCallback_GetMapKeyNodeOrRootLine_ReturnsRootLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root line when expression not found",
			yaml: `
'{$request.body#/callbackUrl}':
  post:
    summary: Callback payload
`,
			key: "{$request.body#/nonexistent}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var callback Callback
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &callback)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := callback.GetRootNode()
			line := callback.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, rootNode.Line, line, "should return root node line")
		})
	}
}

func TestCallback_GetMapKeyNodeOrRoot_Uninitialized(t *testing.T) {
	t.Parallel()

	t.Run("returns root when callback is not initialized", func(t *testing.T) {
		t.Parallel()
		var callback Callback
		// Don't unmarshal - leave uninitialized
		rootNode := &yaml.Node{Kind: yaml.MappingNode, Line: 1}
		result := callback.GetMapKeyNodeOrRoot("anykey", rootNode)
		assert.Equal(t, rootNode, result, "should return root node when not initialized")
	})

	t.Run("returns root when RootNode is nil", func(t *testing.T) {
		t.Parallel()
		callback := &Callback{}
		callback.SetValid(true, true) // Mark as initialized but no RootNode
		rootNode := &yaml.Node{Kind: yaml.MappingNode, Line: 1}
		result := callback.GetMapKeyNodeOrRoot("anykey", rootNode)
		assert.Equal(t, rootNode, result, "should return root node when RootNode is nil")
	})
}

func TestCallback_GetMapKeyNodeOrRootLine_NilNode(t *testing.T) {
	t.Parallel()

	t.Run("returns -1 when GetMapKeyNodeOrRoot returns nil", func(t *testing.T) {
		t.Parallel()
		var callback Callback
		// Pass nil as rootNode - when not initialized, nil is returned
		line := callback.GetMapKeyNodeOrRootLine("anykey", nil)
		assert.Equal(t, -1, line, "should return -1 when node is nil")
	})
}
