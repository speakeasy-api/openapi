package yml_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestResolveOrphanedAliases_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		edit     func(t *testing.T, root *yaml.Node)
		expected string
	}{
		{
			name:     "aliases to anchors earlier in the document are kept",
			input:    "a: &x foo\nb: *x\nc: &m\n  k: v\nd: *m\n",
			edit:     func(t *testing.T, root *yaml.Node) { t.Helper() },
			expected: "a: &x foo\nb: *x\nc: &m\n    k: v\nd: *m\n",
		},
		{
			name:  "alias to a removed anchor becomes a copy",
			input: "a: &x foo\nb: *x\n",
			edit: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				root.Content = root.Content[2:]
			},
			expected: "b: foo\n",
		},
		{
			name:  "alias to a replaced anchor becomes a copy of the old value",
			input: "a: &x foo\nb: *x\n",
			edit: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				root.Content[1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bar", Anchor: "x"}
			},
			expected: "a: &x bar\nb: foo\n",
		},
		{
			name:  "alias moved before its anchor becomes a copy",
			input: "a: &x foo\nb: *x\n",
			edit: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				root.Content = []*yaml.Node{root.Content[2], root.Content[3], root.Content[0], root.Content[1]}
			},
			expected: "b: foo\na: &x foo\n",
		},
		{
			name:  "copied mapping drops nested anchors and copies aliases to them",
			input: "a: &m\n  k: &n v\nb: *m\nc: *n\n",
			edit: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				root.Content = root.Content[2:]
			},
			expected: "b:\n    k: v\nc: v\n",
		},
		{
			name:  "copy of a recursive node points its alias at the copy",
			input: "a: &r\n  - x\n  - *r\nb: *r\n",
			edit: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				root.Content = root.Content[2:]
			},
			expected: "b: &r\n    - x\n    - *r\n",
		},
		{
			name:  "comments on a replaced alias are kept",
			input: "a: &x foo\n# head\nb: *x # line\n",
			edit: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				root.Content = root.Content[2:]
			},
			expected: "# head\nb: foo # line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tt.input), &doc), "input should parse")
			tt.edit(t, doc.Content[0])

			yml.ResolveOrphanedAliases(&doc)

			out, err := yaml.Marshal(&doc)
			require.NoError(t, err, "document should marshal")
			assert.Equal(t, tt.expected, string(out), "output should match")

			var reparsed yaml.Node
			require.NoError(t, yaml.Unmarshal(out, &reparsed), "output should be valid YAML")
		})
	}
}

func TestResolveOrphanedAliases_Nil_Success(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() { yml.ResolveOrphanedAliases(nil) }, "a nil node should be ignored")
}
