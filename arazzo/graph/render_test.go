package graph_test

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraph_Write_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format graph.Format
		golden string
	}{
		{name: "mermaid", format: graph.FormatMermaid, golden: "../testdata/graph/workflows.mmd"},
		{name: "dot", format: graph.FormatDOT, golden: "../testdata/graph/workflows.dot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g, err := graph.Build(loadFixture(t), graph.Options{})
			require.NoError(t, err, "build should succeed")

			var buf bytes.Buffer
			require.NoError(t, g.Write(&buf, tt.format), "write should succeed")

			expected, err := os.ReadFile(tt.golden)
			require.NoError(t, err, "golden file should be read")
			assert.Equal(t, string(expected), buf.String(), "output should match the golden file")
		})
	}
}

func TestGraph_Write_Escaping_Success(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `arazzo: 1.1.0
info:
  title: Escaping
  version: 1.0.0
sourceDescriptions: []
workflows:
  - workflowId: main
    summary: "Say \"hi\" <b>#1</b>\\now"
    steps: []
`)
	g, err := graph.Build(doc, graph.Options{})
	require.NoError(t, err, "build should succeed")

	var mermaid bytes.Buffer
	require.NoError(t, g.WriteMermaid(&mermaid), "write should succeed")
	assert.Equal(t, `flowchart TB
  subgraph wf0 ["Say #quot;hi#quot; #lt;b#gt;#35;1#lt;/b#gt;\now"]
    w0(["main"])
  end
`, mermaid.String(), "mermaid labels should be escaped")

	var dot bytes.Buffer
	require.NoError(t, g.WriteDOT(&dot), "write should succeed")
	assert.Contains(t, dot.String(), `label="Say \"hi\" <b>#1</b>\\now";`, "dot labels should be escaped")
	assert.NotContains(t, dot.String(), " -> ", "a workflow without steps should have no edges")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestGraph_Write_Error(t *testing.T) {
	t.Parallel()

	g, err := graph.Build(loadFixture(t), graph.Options{})
	require.NoError(t, err, "build should succeed")

	err = g.Write(&bytes.Buffer{}, "svg")
	require.Error(t, err, "unknown formats should be rejected")
	assert.Equal(t, "unknown format `svg`, expected mermaid or dot", err.Error(), "error should match")

	require.Error(t, g.Write(failingWriter{}, graph.FormatMermaid), "write errors should be returned")
	require.Error(t, g.Write(failingWriter{}, graph.FormatDOT), "write errors should be returned")
}
