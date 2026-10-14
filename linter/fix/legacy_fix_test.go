package fix_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
	legacyyaml "gopkg.in/yaml.v3"
)

type mockLegacyNodeFix struct {
	mockFix
	legacyNodeApplied bool
	receivedKind      legacyyaml.Kind
	receivedContent   []string
}

func (f *mockLegacyNodeFix) ApplyNode(rootNode *legacyyaml.Node) error {
	f.legacyNodeApplied = true
	if rootNode != nil {
		f.receivedKind = rootNode.Kind
		for _, child := range rootNode.Content {
			f.receivedContent = append(f.receivedContent, child.Value)
		}
	}
	return nil
}

func TestApplyNodeFix_WithLegacyNodeFix(t *testing.T) {
	t.Parallel()

	f := &mockLegacyNodeFix{mockFix: mockFix{description: "legacy node fix"}}
	rootNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "title"},
			{Kind: yaml.ScalarNode, Value: "Test API"},
		},
	}
	doc := &openapi.OpenAPI{}

	err := fix.ApplyNodeFix(f, doc, rootNode)
	require.NoError(t, err, "ApplyNodeFix should not fail")
	assert.True(t, f.legacyNodeApplied, "Legacy ApplyNode should be called")
	assert.Equal(t, legacyyaml.MappingNode, f.receivedKind)
	assert.Equal(t, []string{"title", "Test API"}, f.receivedContent)
	assert.False(t, f.applied, "Apply should not be called")
}
