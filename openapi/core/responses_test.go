package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponses_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns key node when status code exists - first status",
			yaml: `
'200':
  description: Success response
'404':
  description: Not found
`,
			key: "200",
		},
		{
			name: "returns key node when status code exists - second status",
			yaml: `
'200':
  description: Success response
'404':
  description: Not found
`,
			key: "404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var responses Responses
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &responses)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := responses.GetRootNode()
			result := responses.GetMapKeyNodeOrRoot(tt.key, rootNode)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tt.key, result.Value, "should return correct key node")
		})
	}
}

func TestResponses_GetMapKeyNodeOrRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root when status code not found",
			yaml: `
'200':
  description: Success response
`,
			key: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var responses Responses
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &responses)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := responses.GetRootNode()
			result := responses.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, rootNode, result, "should return root node when key not found")
		})
	}
}

func TestResponses_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedLine int
	}{
		{
			name: "returns line number when status code exists - first status",
			yaml: `
'200':
  description: Success response
'404':
  description: Not found
`,
			key:          "200",
			expectedLine: 2,
		},
		{
			name: "returns line number when status code exists - second status",
			yaml: `
'200':
  description: Success response
'404':
  description: Not found
`,
			key:          "404",
			expectedLine: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var responses Responses
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &responses)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := responses.GetRootNode()
			line := responses.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, line, "should return correct line number")
		})
	}
}

func TestResponses_GetMapKeyNodeOrRootLine_ReturnsRootLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root line when status code not found",
			yaml: `
'200':
  description: Success response
`,
			key: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var responses Responses
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &responses)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := responses.GetRootNode()
			line := responses.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, rootNode.Line, line, "should return root node line")
		})
	}
}
