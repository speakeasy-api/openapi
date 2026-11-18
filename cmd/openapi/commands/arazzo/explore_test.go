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

const exploreFixture = "../../../../arazzo/testdata/graph/workflows.arazzo.yaml"

// runExploreLoad creates the explorer model and returns its rendered view and stderr.
func runExploreLoad(t *testing.T, inputFile, stdin string) (string, string, error) {
	t.Helper()

	var stderr bytes.Buffer
	m, err := newExploreModel(t.Context(), inputFile, exploreIO{stdin: strings.NewReader(stdin), stderr: &stderr})
	if err != nil {
		return "", stderr.String(), err
	}
	return m.View().Content, stderr.String(), nil
}

func TestNewExploreModel_Success(t *testing.T) {
	t.Parallel()

	stdinDoc := `arazzo: 1.1.0
info:
  title: ""
  version: ""
sourceDescriptions:
  - name: api
    url: ./openapi.yaml
    type: openapi
workflows:
  - workflowId: piped
    steps:
      - stepId: only
        operationId: getThing
`

	tests := []struct {
		name           string
		inputFile      string
		stdin          string
		expectedView   []string
		expectedStderr string
	}{
		{
			name:         "file",
			inputFile:    exploreFixture,
			expectedView: []string{"adoptPet", "findPet", "registerOwner", "Pet adoption v1.0.0 · Arazzo 1.1.0 · 2 workflows"},
		},
		{
			name:         "stdin",
			inputFile:    "-",
			stdin:        stdinDoc,
			expectedView: []string{"piped", "only", "Arazzo vunknown · Arazzo 1.1.0 · 1 workflows"},
			expectedStderr: "Note: reading document from stdin; interactive TUI requires /dev/tty for keyboard input.\n" +
				"⚠️  Found 2 validation errors in document:\n" +
				"  1. [3:10] error validation-required-field `info.title` is required\n" +
				"  2. [4:12] error validation-required-field `info.version` is required\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			view, stderr, err := runExploreLoad(t, tt.inputFile, tt.stdin)
			require.NoError(t, err, "explorer should load")
			for _, expected := range tt.expectedView {
				assert.Contains(t, view, expected, "view should contain the expected content")
			}
			assert.Equal(t, tt.expectedStderr, stderr, "stderr should match exactly")
		})
	}
}

func TestNewExploreModel_ValidationErrors_Success(t *testing.T) {
	t.Parallel()

	// Seven steps without a target, each a validation error.
	var doc strings.Builder
	doc.WriteString("arazzo: 1.1.0\ninfo:\n  title: Invalid\n  version: 1.0.0\nsourceDescriptions:\n  - name: api\n    url: ./openapi.yaml\n    type: openapi\nworkflows:\n  - workflowId: broken\n    steps:\n")
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		doc.WriteString("      - stepId: " + id + "\n")
	}

	view, stderr, err := runExploreLoad(t, "-", doc.String())
	require.NoError(t, err, "an invalid document should still be explored")
	assert.Contains(t, view, "broken", "the workflow should be shown")
	assert.Contains(t, stderr, "Found 7 validation errors", "the errors should be counted")
	assert.Contains(t, stderr, "  5. ", "the first errors should be listed")
	assert.NotContains(t, stderr, "  6. ", "only the first five errors should be listed")
	assert.Contains(t, stderr, "... and 2 more", "the remaining errors should be counted")
}

func TestNewExploreModel_Error(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.yaml")
	require.NoError(t, os.WriteFile(invalid, []byte("arazzo: [unclosed"), 0o600), "fixture should be written")

	tests := []struct {
		name      string
		inputFile string
		stdin     string
		expected  string
	}{
		{name: "missing file", inputFile: filepath.Join(dir, "missing.yaml"), expected: "failed to open input file"},
		{name: "invalid YAML", inputFile: invalid, expected: "failed to unmarshal Arazzo document"},
		{name: "no workflows", inputFile: "-", stdin: "arazzo: 1.1.0\ninfo:\n  title: Empty\n  version: 1.0.0\nsourceDescriptions: []\nworkflows: []\n", expected: "no workflows found in the Arazzo document"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := runExploreLoad(t, tt.inputFile, tt.stdin)
			require.Error(t, err, "explorer should fail to load")
			assert.Contains(t, err.Error(), tt.expected, "error should explain the failure")
		})
	}
}

func TestGetExploreCommand_Success(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "explore <input-file>", GetExploreCommand().Use, "command should be returned")
}
