package arazzo

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bootstrapFixture = "../../../../arazzo/testdata/bootstrap/petstore.openapi.yaml"

// runBootstrapCommand runs the bootstrap command and returns stdout and stderr.
func runBootstrapCommand(t *testing.T, args []string, flags bootstrapFlags) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := bootstrapArazzo(t.Context(), args, flags, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

func TestBootstrapArazzo_Stdout_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		flags          bootstrapFlags
		expectedStdout []string
		expectedStderr string
	}{
		{
			name:           "sample",
			expectedStdout: []string{"title: Pet adoption workflows\n", "workflowId: adoptPet\n"},
		},
		{
			name:  "from OpenAPI",
			flags: bootstrapFlags{from: bootstrapFixture},
			expectedStdout: []string{
				"url: " + filepath.ToSlash(filepath.Clean(bootstrapFixture)) + "\n",
				"workflowId: pets\n",
				"workflowId: adoption\n",
				"workflowId: default\n",
			},
			expectedStderr: "Generating Arazzo workflows from: " + filepath.Clean(bootstrapFixture),
		},
		{
			name:           "tags",
			flags:          bootstrapFlags{from: bootstrapFixture, tags: []string{"adoption"}, sourceName: "store"},
			expectedStdout: []string{"name: store\n", "workflowId: adoption\n"},
		},
		{
			name:           "operations",
			flags:          bootstrapFlags{from: bootstrapFixture, operations: []string{"listPets", "getPet"}, workflowID: "findPet"},
			expectedStdout: []string{"workflowId: findPet\n", "value: $steps.listPets.outputs.petId\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := runBootstrapCommand(t, nil, tt.flags)
			require.NoError(t, err, "bootstrap should succeed")
			for _, expected := range tt.expectedStdout {
				assert.Contains(t, stdout, expected, "stdout should contain the expected output")
			}
			assert.Contains(t, stderr, tt.expectedStderr, "stderr should contain the expected output")
		})
	}
}

func TestBootstrapArazzo_File_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	from := filepath.Join(dir, "specs", "petstore.openapi.yaml")
	data, err := os.ReadFile(bootstrapFixture)
	require.NoError(t, err, "fixture should open")
	require.NoError(t, os.MkdirAll(filepath.Dir(from), 0o755), "specs directory should be created")
	require.NoError(t, os.WriteFile(from, data, 0o600), "fixture should be copied")

	tests := []struct {
		name        string
		output      string
		expectedURL string
	}{
		{name: "same directory", output: filepath.Join(dir, "specs", "workflows.arazzo.yaml"), expectedURL: "./petstore.openapi.yaml"},
		{name: "sibling directory", output: filepath.Join(dir, "arazzo", "workflows.arazzo.yaml"), expectedURL: "../specs/petstore.openapi.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, os.MkdirAll(filepath.Dir(tt.output), 0o755), "output directory should be created")

			stdout, stderr, err := runBootstrapCommand(t, []string{tt.output}, bootstrapFlags{from: from})
			require.NoError(t, err, "bootstrap should succeed")
			assert.Empty(t, stdout, "nothing should be written to stdout")
			assert.Contains(t, stderr, "📄 Arazzo document written to: "+filepath.Clean(tt.output), "stderr should report the output file")

			written, err := os.ReadFile(tt.output)
			require.NoError(t, err, "output file should be written")
			assert.Contains(t, string(written), "url: "+tt.expectedURL+"\n", "source url should be relative to the output file")
		})
	}
}

func TestBootstrapArazzo_UnresolvedReferences_Success(t *testing.T) {
	t.Parallel()

	from := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, os.WriteFile(from, []byte(`openapi: 3.1.0
info:
  title: Broken
  version: 1.0.0
paths:
  /things:
    get:
      operationId: getThings
      parameters:
        - $ref: "#/components/parameters/Missing"
      responses:
        "200":
          description: OK
`), 0o600), "fixture should be written")

	stdout, stderr, err := runBootstrapCommand(t, nil, bootstrapFlags{from: from})
	require.NoError(t, err, "bootstrap should succeed")
	assert.Contains(t, stdout, "operationId: getThings\n", "the operation should still be generated")
	assert.Contains(t, stderr, "Some references could not be resolved", "unresolved references should be reported")
}

func TestBootstrapArazzo_Error(t *testing.T) {
	t.Parallel()

	invalid := filepath.Join(t.TempDir(), "invalid.yaml")
	require.NoError(t, os.WriteFile(invalid, []byte("openapi: [unclosed"), 0o600), "fixture should be written")

	tests := []struct {
		name     string
		args     []string
		flags    bootstrapFlags
		expected string
	}{
		{name: "flags without from", flags: bootstrapFlags{tags: []string{"pets"}}, expected: "--tag, --operations, --workflow-id and --source-name require --from"},
		{name: "missing OpenAPI document", flags: bootstrapFlags{from: "missing.yaml"}, expected: "failed to open OpenAPI document"},
		{name: "invalid OpenAPI document", flags: bootstrapFlags{from: invalid}, expected: "failed to unmarshal OpenAPI document"},
		{name: "unknown tag", flags: bootstrapFlags{from: bootstrapFixture, tags: []string{"nope"}}, expected: "no operations found for tag `nope`"},
		{name: "unwritable output", args: []string{filepath.Join(t.TempDir(), "missing", "out.yaml")}, expected: "failed to write output file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := runBootstrapCommand(t, tt.args, tt.flags)
			require.Error(t, err, "bootstrap should fail")
			assert.Contains(t, err.Error(), tt.expected, "error should explain the failure")
		})
	}
}

func TestGetBootstrapCommand_Success(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "bootstrap [output-file]", GetBootstrapCommand().Use, "command should be returned")
}
