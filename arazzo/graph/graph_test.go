package graph_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseDoc parses an Arazzo document without validation.
func parseDoc(t *testing.T, doc string) *arazzo.Arazzo {
	t.Helper()

	a, _, err := arazzo.Unmarshal(t.Context(), bytes.NewBufferString(doc), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")
	return a
}

// loadFixture parses the graph fixture.
func loadFixture(t *testing.T) *arazzo.Arazzo {
	t.Helper()

	data, err := os.ReadFile("../testdata/graph/workflows.arazzo.yaml")
	require.NoError(t, err, "fixture should be read")
	return parseDoc(t, string(data))
}

// edges formats the edges of the graph as `<from> -<kind>-> <to> (<label>)`, using node labels.
func edges(g *graph.Graph) []string {
	kinds := map[graph.EdgeKind]string{
		graph.EdgeStart:       "start",
		graph.EdgeNext:        "next",
		graph.EdgeSuccessGoto: "success",
		graph.EdgeFailureGoto: "failure",
		graph.EdgeRetry:       "retry",
		graph.EdgeDependsOn:   "dependsOn",
		graph.EdgeCall:        "runs",
	}
	out := make([]string, 0, len(g.Edges))
	for _, e := range g.Edges {
		s := e.From.Label + " -" + kinds[e.Kind] + "-> " + e.To.Label
		if e.Label != "" {
			s += " (" + e.Label + ")"
		}
		out = append(out, s)
	}
	return out
}

func TestBuild_Success(t *testing.T) {
	t.Parallel()

	g, err := graph.Build(loadFixture(t), graph.Options{})
	require.NoError(t, err, "build should succeed")

	assert.Equal(t, graph.DirectionTopBottom, g.Direction, "direction should default to top to bottom")
	require.Len(t, g.Workflows, 2, "both workflows should be included")
	assert.Equal(t, "Find a pet and adopt it", g.Workflows[0].Title, "title should be the summary")
	assert.Equal(t, "registerOwner", g.Workflows[1].Title, "title should fall back to the workflowId")

	steps := g.Workflows[0].Steps
	require.Len(t, steps, 3, "all steps should be included")
	assert.Equal(t, "listPets", steps[0].Detail, "detail should be the operationId")
	assert.Equal(t, "{$sourceDescriptions.petstore.url}#/paths/~1pets~1{petId}/post", steps[1].Detail, "detail should be the operationPath")
	assert.Equal(t, "workflow registerOwner", steps[2].Detail, "detail should be the workflow")

	externals := []string{}
	for _, n := range g.External {
		assert.Equal(t, graph.NodeExternal, n.Kind, "external nodes should be external")
		externals = append(externals, n.Label)
	}
	assert.Equal(t, []string{"shared.audit", "shared.cleanup", "shared.audit.steps.log"}, externals, "references to other documents should be external")

	expected := []string{
		"adoptPet -dependsOn-> registerOwner (dependsOn)",
		"adoptPet -dependsOn-> shared.audit (dependsOn)",
		"adoptPet -start-> findPet",
		"findPet -success-> notify (tooMany)",
		"findPet -retry-> findPet (retryBusy)",
		"findPet -next-> adopt",
		"adopt -dependsOn-> register (dependsOn)",
		"adopt -failure-> shared.cleanup (giveUp)",
		"adopt -retry-> adopt (retryBusy)",
		"notify -runs-> registerOwner (runs)",
		"notify -failure-> findPet (backToStart)",
		"registerOwner -start-> register",
		"register -dependsOn-> shared.audit.steps.log (dependsOn)",
	}
	assert.Equal(t, expected, edges(g), "edges should match")
}

func TestBuild_WorkflowFilter_Success(t *testing.T) {
	t.Parallel()

	g, err := graph.Build(loadFixture(t), graph.Options{WorkflowID: "adoptPet", Direction: graph.DirectionLeftRight})
	require.NoError(t, err, "build should succeed")

	assert.Equal(t, graph.DirectionLeftRight, g.Direction, "direction should be set")
	require.Len(t, g.Workflows, 1, "only the selected workflow should be included")

	externals := []string{}
	for _, n := range g.External {
		externals = append(externals, n.Label)
	}
	assert.Equal(t, []string{"registerOwner", "shared.audit", "registerOwner.steps.register", "shared.cleanup"}, externals, "other workflows should be external")
}

func TestBuild_Unconditional_Success(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `arazzo: 1.1.0
info:
  title: Actions
  version: 1.0.0
sourceDescriptions: []
workflows:
  - workflowId: main
    steps:
      - stepId: first
        operationId: first
        onSuccess:
          - name: skip
            type: goto
            stepId: third
          - name: never
            type: goto
            stepId: second
        onFailure:
          - name: stop
            type: end
          - name: never
            type: retry
      - null
      - stepId: second
        operationId: b
        dependsOn:
          - first
          - missing
        onSuccess:
          - name: noTarget
            type: goto
      - stepId: third
        channelPath: $sourceDescriptions.events.channels.pets
      - stepId: noTarget
  - workflowId: main
    steps:
      - stepId: duplicate
        operationId: d
        onSuccess:
          - name: back
            type: goto
            stepId: first
`)

	g, err := graph.Build(doc, graph.Options{})
	require.NoError(t, err, "build should succeed")

	steps := g.Workflows[0].Steps
	require.Len(t, steps, 4, "null steps should be left out")
	assert.Empty(t, steps[0].Detail, "a target matching the stepId should not be repeated")
	assert.Equal(t, "$sourceDescriptions.events.channels.pets", steps[2].Detail, "detail should be the channelPath")
	assert.Empty(t, steps[3].Detail, "a step without a target has no detail")
	require.Len(t, g.Workflows, 2, "duplicate workflows should both be drawn")

	expected := []string{
		"main -start-> first",
		"first -success-> third (skip)",
		"second -dependsOn-> first (dependsOn)",
		"second -dependsOn-> main.steps.missing (dependsOn)",
		"second -success-> unknown target (noTarget)",
		"third -next-> noTarget",
		"main -start-> duplicate",
		"duplicate -success-> first (back)",
	}
	assert.Equal(t, expected, edges(g), "actions after an unconditional action should be left out")
}

func TestBuild_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      *arazzo.Arazzo
		opts     graph.Options
		expected string
	}{
		{name: "nil document", doc: nil, expected: "document is nil"},
		{name: "unknown direction", doc: &arazzo.Arazzo{}, opts: graph.Options{Direction: "RL"}, expected: "unknown direction `RL`, expected TB or LR"},
		{name: "unknown workflow", doc: &arazzo.Arazzo{}, opts: graph.Options{WorkflowID: "missing"}, expected: "workflow `missing` not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := graph.Build(tt.doc, tt.opts)
			require.Error(t, err, "build should fail")
			assert.Equal(t, tt.expected, err.Error(), "error should match")
		})
	}
}
