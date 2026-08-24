package overlay

import (
	"testing"

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
		stabilizeFoldedScalars(&node)
	}

	out, err := yaml.Marshal(&node)
	require.NoError(t, err)

	return string(out)
}

func applyRoundTrip(t *testing.T, o *Overlay, doc string) string {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &node))
	require.NoError(t, o.ApplyTo(&node))

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

func TestApplyToSurvivesRepeatedApplies(t *testing.T) {
	t.Parallel()

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{
				Target: "$.title",
				Update: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "Updated"},
			},
		},
	}

	doc := "title: Original\n" + foldedWithMoreIndentedLine
	want := decodeDescription(t, doc)

	for i := range 30 {
		doc = applyRoundTrip(t, o, doc)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d applies", i+1)
	}
}

func TestApplyToStrictSurvivesRepeatedApplies(t *testing.T) {
	t.Parallel()

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{
				Target: "$.title",
				Update: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "Updated"},
			},
		},
	}

	doc := "title: Original\n" + foldedWithMoreIndentedLine
	want := decodeDescription(t, doc)

	for i := range 30 {
		var node yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(doc), &node))
		_, err := o.ApplyToStrict(&node)
		require.NoError(t, err)

		out, err := yaml.Marshal(&node)
		require.NoError(t, err)

		doc = string(out)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d applies", i+1)
	}
}

func TestStabilizeFoldedScalarsHandlesExplicitTags(t *testing.T) {
	t.Parallel()

	doc := "description: !!str >-\n  a\n   b\n"
	want := decodeDescription(t, doc)

	for i := range 5 {
		doc = roundTrip(t, doc, true)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d round trips", i+1)
	}
	assert.Contains(t, doc, "!!str", "explicit tag should be preserved")
}

func TestStabilizeFoldedScalarsSurvivesRepeatedRoundTrips(t *testing.T) {
	t.Parallel()

	want := decodeDescription(t, foldedWithMoreIndentedLine)

	doc := foldedWithMoreIndentedLine
	for i := range 30 {
		doc = roundTrip(t, doc, true)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d round trips", i+1)
	}
}

// Fails once gopkg.in/yaml.v3 fixes the emitter, at which point stabilizeFoldedScalars can go.
func TestFoldedScalarGrowsWithoutStabilizer(t *testing.T) {
	t.Parallel()

	before := decodeDescription(t, foldedWithMoreIndentedLine)
	after := decodeDescription(t, roundTrip(t, foldedWithMoreIndentedLine, false))

	assert.NotEqual(t, before, after)
}

func TestStabilizeFoldedScalarsLeavesStableStylesAlone(t *testing.T) {
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

func TestStabilizeFoldedScalarsToleratesNilNodes(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() { stabilizeFoldedScalars(nil) })

	// A hand-built tree may carry nil children; recursion must not panic.
	assert.NotPanics(t, func() {
		stabilizeFoldedScalars(&yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{nil, nil},
		})
	})
}

func TestHasMoreIndentedLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "no indentation", value: "one\ntwo", want: false},
		{name: "space indented line", value: "one\n two", want: true},
		{name: "tab indented line", value: "one\n\ttwo", want: true},
		{name: "blank lines only", value: "one\n\ntwo", want: false},
		{name: "empty", value: "", want: false},
		{name: "first line indented", value: " one\ntwo", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, hasMoreIndentedLine(tt.value))
		})
	}
}
