package arazzo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const graphFixture = "../../../../arazzo/testdata/graph/workflows.arazzo.yaml"

// runGraphCommand runs the graph command and returns stdout and stderr.
func runGraphCommand(t *testing.T, args []string, flags graphFlags, stdin string) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := graphArazzo(t.Context(), args, flags, graphIO{stdin: strings.NewReader(stdin), stdout: &stdout, stderr: &stderr})
	return stdout.String(), stderr.String(), err
}

func TestGraphArazzo_Success(t *testing.T) {
	t.Parallel()

	stdinDoc := `arazzo: 1.1.0
info:
  title: Stdin
  version: 1.0.0
sourceDescriptions: []
workflows:
  - workflowId: main
    steps:
      - stepId: only
        operationId: getPet
`

	tests := []struct {
		name           string
		args           []string
		stdin          string
		flags          graphFlags
		expectedStdout []string
		expectedStderr string
	}{
		{
			name:           "mermaid by default",
			args:           []string{graphFixture},
			expectedStdout: []string{"flowchart TB\n", `subgraph wf0 ["Find a pet and adopt it"]`},
			expectedStderr: "Rendering Arazzo document: " + filepath.Clean(graphFixture),
		},
		{
			name:           "dot",
			args:           []string{graphFixture},
			flags:          graphFlags{format: "DOT", direction: "lr"},
			expectedStdout: []string{"digraph arazzo {\n", "rankdir=LR;"},
		},
		{
			name:           "single workflow",
			args:           []string{graphFixture},
			flags:          graphFlags{workflow: "registerOwner"},
			expectedStdout: []string{`subgraph wf0 ["registerOwner"]`},
		},
		{
			name:           "stdin",
			args:           []string{"-"},
			stdin:          stdinDoc,
			expectedStdout: []string{`w0s0["only<br/>getPet"]`},
			expectedStderr: "Rendering Arazzo document from stdin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := runGraphCommand(t, tt.args, tt.flags, tt.stdin)
			require.NoError(t, err, "graph should succeed")
			for _, expected := range tt.expectedStdout {
				assert.Contains(t, stdout, expected, "stdout should contain expected output")
			}
			assert.Contains(t, stderr, tt.expectedStderr, "stderr should contain expected output")
		})
	}
}

func TestGraphArazzo_OutputFile_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     string
		flags    graphFlags
		expected string
	}{
		{name: "dot extension", file: "workflows.dot", expected: "digraph arazzo {"},
		{name: "gv extension", file: "workflows.GV", expected: "digraph arazzo {"},
		{name: "mermaid extension", file: "workflows.mmd", expected: "flowchart TB"},
		{name: "format overrides the extension", file: "workflows.dot", flags: graphFlags{format: "mermaid"}, expected: "flowchart TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := filepath.Join(t.TempDir(), tt.file)
			stdout, stderr, err := runGraphCommand(t, []string{graphFixture, output}, tt.flags, "")
			require.NoError(t, err, "graph should succeed")
			assert.Empty(t, stdout, "nothing should be written to stdout")
			assert.Contains(t, stderr, "Diagram written to: "+output, "stderr should report the output file")

			data, err := os.ReadFile(output)
			require.NoError(t, err, "output file should be written")
			assert.True(t, strings.HasPrefix(string(data), tt.expected), "output should be in the expected format")
		})
	}
}

func TestGraphArazzo_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		stdin         string
		flags         graphFlags
		expectedError string
	}{
		{name: "unknown format", args: []string{graphFixture}, flags: graphFlags{format: "svg"}, expectedError: "unknown format `svg`, expected mermaid or dot"},
		{name: "missing file", args: []string{"missing.arazzo.yaml"}, expectedError: "failed to open input file"},
		{name: "invalid document", args: []string{"-"}, stdin: "{", expectedError: "failed to unmarshal Arazzo document"},
		{name: "unknown workflow", args: []string{graphFixture}, flags: graphFlags{workflow: "missing"}, expectedError: "workflow `missing` not found"},
		{name: "unknown direction", args: []string{graphFixture}, flags: graphFlags{direction: "RL"}, expectedError: "unknown direction `RL`"},
		{name: "unwritable output", args: []string{graphFixture, filepath.Join("missing", "dir", "out.mmd")}, expectedError: "failed to write output file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := runGraphCommand(t, tt.args, tt.flags, tt.stdin)
			require.Error(t, err, "graph should fail")
			assert.Contains(t, err.Error(), tt.expectedError, "error should match")
		})
	}
}

func TestGraphArazzo_StdinReadError(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := graphArazzo(t.Context(), []string{"-"}, graphFlags{}, graphIO{stdin: failingReader{}, stdout: &stdout, stderr: &stderr})
	require.Error(t, err, "graph should fail")
	assert.Contains(t, err.Error(), "failed to read stdin", "error should match")
	assert.NotNil(t, GetGraphCommand(), "graph command should be exposed")
}
