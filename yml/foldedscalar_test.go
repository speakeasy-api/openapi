package yml_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The trailing table row is indented one space further than the rows above it,
// which is what triggers the yaml.v3 emitter defect this file guards against.
const foldedWithMoreIndentedLine = `description: >-
  ### Widgets

  | Name | Kind |
  | ---- | ---- |
   | acme | ` + "`petstore`" + ` |
`

func roundTrip(t *testing.T, doc string, stabilize bool) string {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &node))

	if stabilize {
		yml.StabilizeFoldedScalars(&node)
	}

	out, err := yaml.Marshal(&node)
	require.NoError(t, err)

	return string(out)
}

func decodeDescription(t *testing.T, doc string) string {
	t.Helper()

	var decoded struct {
		Description string `yaml:"description"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(doc), &decoded))

	return decoded.Description
}

func TestStabilizeFoldedScalars_SurvivesRepeatedRoundTrips(t *testing.T) {
	t.Parallel()

	want := decodeDescription(t, foldedWithMoreIndentedLine)

	doc := foldedWithMoreIndentedLine
	for i := range 30 {
		doc = roundTrip(t, doc, true)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d round trips", i+1)
	}
}

func TestStabilizeFoldedScalars_HandlesExplicitTags(t *testing.T) {
	t.Parallel()

	doc := "description: !!str >-\n  a\n   b\n"
	want := decodeDescription(t, doc)

	for i := range 5 {
		doc = roundTrip(t, doc, true)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d round trips", i+1)
	}
	assert.Contains(t, doc, "!!str", "explicit tag should be preserved")
}

func TestStabilizeFoldedScalars_LeavesStableStylesAlone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		doc   string
		style string
	}{
		{
			name:  "folded scalar with no more-indented line",
			doc:   "description: >-\n  one line\n  another line\n",
			style: ">-",
		},
		{
			name:  "literal scalar with more-indented line",
			doc:   "description: |-\n  one line\n   more indented\n",
			style: "|-",
		},
		{
			name: "plain scalar",
			doc:  "description: just a string\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := decodeDescription(t, tt.doc)

			doc := roundTrip(t, tt.doc, true)
			assert.Equal(t, want, decodeDescription(t, doc))
			if tt.style != "" {
				assert.Contains(t, doc, tt.style, "original style should be preserved")
			}

			for range 5 {
				next := roundTrip(t, doc, true)
				assert.Equal(t, doc, next, "representation should be a fixed point")
				assert.Equal(t, want, decodeDescription(t, next))
				doc = next
			}
		})
	}
}

func TestStabilizeFoldedScalars_DetectsMoreIndentedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		restyled bool
	}{
		{name: "no indentation", value: "one\ntwo", restyled: false},
		{name: "space indented line", value: "one\n two", restyled: true},
		{name: "tab indented line", value: "one\n\ttwo", restyled: true},
		{name: "blank lines only", value: "one\n\ntwo", restyled: false},
		{name: "empty", value: "", restyled: false},
		{name: "first line indented", value: " one\ntwo", restyled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Style: yaml.FoldedStyle,
				Value: tt.value,
			}

			yml.StabilizeFoldedScalars(node)

			want := yaml.FoldedStyle
			if tt.restyled {
				want = yaml.LiteralStyle
			}
			assert.Equal(t, want, node.Style)
			assert.Equal(t, tt.value, node.Value, "value must not be rewritten")
		})
	}
}

func TestStabilizeFoldedScalars_ToleratesNilNodes(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() { yml.StabilizeFoldedScalars(nil) })

	// A hand-built tree may carry nil children; recursion must not panic.
	assert.NotPanics(t, func() {
		yml.StabilizeFoldedScalars(&yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{nil, nil},
		})
	})
}

// An anchored scalar is an ordinary node in the tree, so aliases pointing at it need
// no special handling: stabilizing the anchor definition covers every reference to it.
func TestStabilizeFoldedScalars_HandlesAnchoredScalars(t *testing.T) {
	t.Parallel()

	doc := "a: &anchor >-\n  one\n   more indented\nb: *anchor\n"

	var want struct {
		A string `yaml:"a"`
		B string `yaml:"b"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(doc), &want))

	for i := range 30 {
		doc = roundTrip(t, doc, true)

		var got struct {
			A string `yaml:"a"`
			B string `yaml:"b"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(doc), &got))
		assert.Equal(t, want, got, "value changed after %d round trips", i+1)
		assert.Contains(t, doc, "*anchor", "alias should be preserved")
	}
}

// Fails once gopkg.in/yaml.v3 fixes the emitter, at which point StabilizeFoldedScalars can go.
func TestFoldedScalarGrowsWithoutStabilizer(t *testing.T) {
	t.Parallel()

	before := decodeDescription(t, foldedWithMoreIndentedLine)
	after := decodeDescription(t, roundTrip(t, foldedWithMoreIndentedLine, false))

	assert.NotEqual(t, before, after)
}
