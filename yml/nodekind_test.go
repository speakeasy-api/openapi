package yml_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNodeKindToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		kind     yaml.Kind
		expected string
	}{
		{
			name:     "document node",
			kind:     yaml.DocumentNode,
			expected: "document",
		},
		{
			name:     "sequence node",
			kind:     yaml.SequenceNode,
			expected: "sequence",
		},
		{
			name:     "mapping node",
			kind:     yaml.MappingNode,
			expected: "mapping",
		},
		{
			name:     "scalar node",
			kind:     yaml.ScalarNode,
			expected: "scalar",
		},
		{
			name:     "alias node",
			kind:     yaml.AliasNode,
			expected: "alias",
		},
		{
			name:     "unknown node",
			kind:     yaml.Kind(99), // Invalid kind
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.NodeKindToString(tt.kind)
			assert.Equal(t, tt.expected, result)
		})
	}
}
