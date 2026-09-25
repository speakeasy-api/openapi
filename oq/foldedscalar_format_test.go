package oq_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The trailing table row is indented one space further than the rows above it, which
// makes yaml.v3 emit an extra line break before it on every encode. The break lands
// inside the scalar, so it changes the decoded value rather than just the layout.
const foldedScalarSpec = `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Widget:
      type: object
      description: >-
        ### Widgets

        | Name | Kind |
        | ---- | ---- |
         | acme | ` + "`petstore`" + ` |
`

func loadFoldedScalarGraph(t *testing.T) *graph.SchemaGraph {
	t.Helper()

	ctx := t.Context()

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(foldedScalarSpec), openapi.WithSkipValidation())
	require.NoError(t, err)
	require.NotNil(t, doc)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "spec.yaml",
	})

	return graph.Build(ctx, idx)
}

// formattedDescription pulls the description back out of a `key:\n  <schema>` wrapper.
func formattedDescription(t *testing.T, formatted string) string {
	t.Helper()

	var decoded map[string]struct {
		Description string `yaml:"description"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(formatted), &decoded))
	require.Len(t, decoded, 1)

	for _, schema := range decoded {
		return schema.Description
	}

	return ""
}

// `openapi spec query --format yaml` marshals graph nodes with its own encoder, so it
// needs the same stabilization as every other encode boundary.
func TestFormatYAML_FoldedScalarKeepsItsValue(t *testing.T) {
	t.Parallel()

	var want struct {
		Components struct {
			Schemas map[string]struct {
				Description string `yaml:"description"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(foldedScalarSpec), &want))
	wantDescription := want.Components.Schemas["Widget"].Description
	require.Contains(t, wantDescription, "| acme |")

	g := loadFoldedScalarGraph(t)

	result, err := oq.Execute(`schemas | where(name == "Widget")`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	formatted := oq.FormatYAML(result, g)
	require.NotEmpty(t, formatted)

	assert.Equal(t, wantDescription, formattedDescription(t, formatted))
	assert.NotContains(t, formatted, ">-", "affected folded scalars should be emitted as literal blocks")
}

// Formatting the same result twice must not drift, since FormatYAML restyles graph
// nodes in place.
func TestFormatYAML_FoldedScalarIsAFixedPoint(t *testing.T) {
	t.Parallel()

	g := loadFoldedScalarGraph(t)

	result, err := oq.Execute(`schemas | where(name == "Widget")`, g)
	require.NoError(t, err)

	first := oq.FormatYAML(result, g)
	for i := range 3 {
		assert.Equal(t, first, oq.FormatYAML(result, g), "output changed on format %d", i+2)
	}
}
