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
}

func (f *mockLegacyNodeFix) ApplyNode(rootNode *legacyyaml.Node) error {
	f.legacyNodeApplied = true
	return nil
}

func TestApplyNodeFix_WithLegacyNodeFix(t *testing.T) {
	t.Parallel()

	f := &mockLegacyNodeFix{mockFix: mockFix{description: "legacy node fix"}}
	rootNode := &yaml.Node{Kind: yaml.MappingNode}
	doc := &openapi.OpenAPI{}

	err := fix.ApplyNodeFix(f, doc, rootNode)
	require.NoError(t, err, "ApplyNodeFix should not fail")
	assert.True(t, f.legacyNodeApplied, "Legacy ApplyNode should be called")
	assert.False(t, f.applied, "Apply should not be called")
}
