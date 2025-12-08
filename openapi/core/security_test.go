package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSecurityRequirement_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns key node when key exists - first key",
			yaml: `
oauth2:
  - read:pets
  - write:pets
api_key: []
`,
			key: "oauth2",
		},
		{
			name: "returns key node when key exists - second key",
			yaml: `
oauth2:
  - read:pets
api_key: []
`,
			key: "api_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var secReq SecurityRequirement
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &secReq)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := secReq.GetRootNode()
			result := secReq.GetMapKeyNodeOrRoot(tt.key, rootNode)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tt.key, result.Value, "should return correct key node")
		})
	}
}

func TestSecurityRequirement_GetMapKeyNodeOrRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root when key not found",
			yaml: `
oauth2:
  - read:pets
`,
			key: "nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var secReq SecurityRequirement
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &secReq)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := secReq.GetRootNode()
			result := secReq.GetMapKeyNodeOrRoot(tt.key, rootNode)
			assert.Equal(t, rootNode, result, "should return root node when key not found")
		})
	}
}

func TestSecurityRequirement_GetMapKeyNodeOrRootLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedLine int
	}{
		{
			name: "returns line number when key exists - first key",
			yaml: `
oauth2:
  - read:pets
  - write:pets
api_key: []
`,
			key:          "oauth2",
			expectedLine: 2,
		},
		{
			name: "returns line number when key exists - second key",
			yaml: `
oauth2:
  - read:pets
api_key: []
`,
			key:          "api_key",
			expectedLine: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var secReq SecurityRequirement
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &secReq)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := secReq.GetRootNode()
			line := secReq.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, tt.expectedLine, line, "should return correct line number")
		})
	}
}

func TestSecurityRequirement_GetMapKeyNodeOrRootLine_ReturnsRootLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{
			name: "returns root line when key not found",
			yaml: `
oauth2:
  - read:pets
`,
			key: "nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var secReq SecurityRequirement
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &secReq)
			require.NoError(t, err, "unmarshal should succeed")

			rootNode := secReq.GetRootNode()
			line := secReq.GetMapKeyNodeOrRootLine(tt.key, rootNode)
			assert.Equal(t, rootNode.Line, line, "should return root node line")
		})
	}
}

// Helper function
func parseYAML(t *testing.T, yml string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)
	return &node
}
