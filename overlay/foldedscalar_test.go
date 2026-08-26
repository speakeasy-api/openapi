package overlay_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The trailing table row is indented one space further than the rows above it,
// which is what triggers the yaml.v3 emitter defect these tests guard against.
const foldedWithMoreIndentedLine = `description: >-
  ### Widgets

  | Name | Kind |
  | ---- | ---- |
   | acme | ` + "`petstore`" + ` |
`

func testOverlay() *overlay.Overlay {
	return &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Test", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.title",
				Update: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "Updated"},
			},
		},
	}
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

	o := testOverlay()

	doc := "title: Original\n" + foldedWithMoreIndentedLine
	want := decodeDescription(t, doc)

	for i := range 3 {
		var node yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(doc), &node))
		require.NoError(t, o.ApplyTo(&node))

		out, err := yaml.Marshal(&node)
		require.NoError(t, err)

		doc = string(out)
		assert.Equal(t, want, decodeDescription(t, doc), "value changed after %d applies", i+1)
	}
}

func TestApplyToStrictSurvivesRepeatedApplies(t *testing.T) {
	t.Parallel()

	o := testOverlay()

	doc := "title: Original\n" + foldedWithMoreIndentedLine
	want := decodeDescription(t, doc)

	for i := range 3 {
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

// An overlay carries folded scalars of its own, in the update payloads it applies.
// Format and ToString are separate entry points and each stabilizes independently.
func TestFormatSurvivesRepeatedRoundTrips(t *testing.T) {
	t.Parallel()

	src := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions:
  - target: $.info
    update:
` + indent(foldedWithMoreIndentedLine, "      ")

	serializers := map[string]func(*overlay.Overlay) (string, error){
		"ToString": func(o *overlay.Overlay) (string, error) {
			return o.ToString()
		},
		"Format": func(o *overlay.Overlay) (string, error) {
			var buf bytes.Buffer
			if err := o.Format(&buf); err != nil {
				return "", err
			}

			return buf.String(), nil
		},
	}

	for name, serialize := range serializers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			want := updateDescription(t, src)

			doc := src
			for i := range 3 {
				o, err := overlay.ParseReader(strings.NewReader(doc))
				require.NoError(t, err)

				formatted, err := serialize(o)
				require.NoError(t, err)

				doc = formatted
				assert.Equal(t, want, updateDescription(t, doc), "value changed after %d round trips", i+1)
			}
		})
	}
}

// A nil overlay serializes as "null"; stabilizing must not change that.
func TestFormatToleratesNilOverlay(t *testing.T) {
	t.Parallel()

	var o *overlay.Overlay

	formatted, err := o.ToString()
	require.NoError(t, err)
	assert.Equal(t, "null\n", formatted)

	var buf bytes.Buffer
	require.NoError(t, o.Format(&buf))
	assert.Equal(t, "null\n", buf.String())
}

func indent(doc string, prefix string) string {
	lines := strings.Split(strings.TrimSuffix(doc, "\n"), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func updateDescription(t *testing.T, doc string) string {
	t.Helper()

	o, err := overlay.ParseReader(strings.NewReader(doc))
	require.NoError(t, err)
	require.Len(t, o.Actions, 1)

	var decoded struct {
		Description string `yaml:"description"`
	}
	require.NoError(t, o.Actions[0].Update.Decode(&decoded))

	return decoded.Description
}
