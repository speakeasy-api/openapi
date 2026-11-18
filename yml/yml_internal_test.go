package yml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
)

func TestSameContent_Success(t *testing.T) {
	t.Parallel()

	target := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "t", Anchor: "t"}
	other := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "t", Anchor: "o"}
	scalar := func(value string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value} }
	alias := func(to *yaml.Node) *yaml.Node { return &yaml.Node{Kind: yaml.AliasNode, Value: to.Anchor, Alias: to} }
	mapping := func(content ...*yaml.Node) *yaml.Node {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: content}
	}

	tests := []struct {
		name     string
		a, b     *yaml.Node
		expected bool
	}{
		{name: "both nil", expected: true},
		{name: "one nil", a: scalar("a"), expected: false},
		{name: "equal scalars", a: scalar("a"), b: scalar("a"), expected: true},
		{name: "different scalars", a: scalar("a"), b: scalar("b"), expected: false},
		{name: "aliases to the same node", a: alias(target), b: alias(target), expected: true},
		{name: "aliases to different nodes with equal values", a: alias(target), b: alias(other), expected: false},
		{name: "alias and scalar with the alias's value", a: alias(target), b: scalar("t"), expected: false},
		{name: "equal mappings", a: mapping(scalar("k"), scalar("v")), b: mapping(scalar("k"), scalar("v")), expected: true},
		{name: "mappings with different values", a: mapping(scalar("k"), scalar("v")), b: mapping(scalar("k"), scalar("w")), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, sameContent(tt.a, tt.b), "comparison should match")
		})
	}
}
