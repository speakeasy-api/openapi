package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
