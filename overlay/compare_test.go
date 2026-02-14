package overlay_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCompare(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)
	node2, err := loader.LoadSpecification("testdata/openapi-overlayed.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay-generated.yaml")
	require.NoError(t, err)

	o2, err := overlay.Compare("Drinks Overlay", node, *node2)
	require.NoError(t, err)

	o1s, err := o.ToString()
	require.NoError(t, err)
	o2s, err := o2.ToString()
	require.NoError(t, err)

	// Uncomment this if we've improved the output
	// os.WriteFile("testdata/overlay-generated.yaml", []byte(o2s), 0644)
	assert.Equal(t, o1s, o2s)

	// round trip it
	err = o.ApplyTo(node)
	require.NoError(t, err)
	NodeMatchesFile(t, node, "testdata/openapi-overlayed.yaml")

}

func TestCompare_ArrayAppendMultiple(t *testing.T) {
	t.Parallel()

	original := `items:
  - name: a
  - name: b
`
	target := `items:
  - name: a
  - name: b
  - name: c
  - name: d
`
	var origNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(original), &origNode))
	var targetNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(target), &targetNode))

	o, err := overlay.Compare("test", &origNode, targetNode)
	require.NoError(t, err)

	// Should produce a single update action (no remove), appending the new elements
	require.Len(t, o.Actions, 1)
	assert.Equal(t, `$["items"]`, o.Actions[0].Target)
	assert.False(t, o.Actions[0].Remove)
	assert.Equal(t, yaml.SequenceNode, o.Actions[0].Update.Kind)
	require.Len(t, o.Actions[0].Update.Content, 2)
	assert.Equal(t, "name: c\n", nodeToString(t, o.Actions[0].Update.Content[0]))
	assert.Equal(t, "name: d\n", nodeToString(t, o.Actions[0].Update.Content[1]))

	// Round-trip: applying the overlay should produce the target
	err = o.ApplyTo(&origNode)
	require.NoError(t, err)
	var roundTripped yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(target), &roundTripped))

	origStr := nodeToString(t, &origNode)
	targetStr := nodeToString(t, &roundTripped)
	assert.Equal(t, targetStr, origStr)
}

func nodeToString(t *testing.T, node *yaml.Node) string {
	t.Helper()
	out, err := yaml.Marshal(node)
	require.NoError(t, err)
	return string(out)
}
