package arazzo

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeArazzo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		input             string
		expected          string
		expectedStderr    []string
		notExpectedStderr []string
	}{
		{
			name: "upgrades 1.0 document",
			input: `arazzo: 1.0.1
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        successCriteria:
          - context: $response.body
            condition: $[?@.status == 'available']
            type: jsonpath
`,
			expected: `arazzo: 1.1.0
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        successCriteria:
          - context: $response.body
            condition: $[?@.status == 'available']
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
`,
			expectedStderr:    []string{"Successfully upgraded from 1.0.1 to 1.1.0"},
			notExpectedStderr: []string{"validation errors"},
		},
		{
			name: "reports expressions that need manual changes",
			input: `arazzo: 1.0.1
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        outputs:
          location: $response.headers.Location
`,
			expected: `arazzo: 1.1.0
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        outputs:
          location: $response.headers.Location
`,
			expectedStderr: []string{
				"Successfully upgraded from 1.0.1 to 1.1.0",
				"Found 1 validation errors in the upgraded document that need to be fixed manually",
				"$response.headers.Location",
			},
		},
		{
			name:           "no upgrade needed",
			input:          "arazzo: 1.1.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expected:       "arazzo: 1.1.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expectedStderr: []string{"No upgrade needed - document is already at version 1.1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)

			err := upgradeArazzo(t.Context(), bytes.NewBufferString(tt.input), out, stderr)
			require.NoError(t, err, "upgrade should succeed")
			assert.Equal(t, tt.expected, out.String(), "upgraded document should match")
			for _, expected := range tt.expectedStderr {
				assert.Contains(t, stderr.String(), expected, "stderr should contain expected output")
			}
			for _, notExpected := range tt.notExpectedStderr {
				assert.NotContains(t, stderr.String(), notExpected, "stderr should not contain unexpected output")
			}
		})
	}
}

func TestUpgradeArazzo_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "unsupported version",
			input:         "arazzo: 2.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expectedError: "only Arazzo versions `1.0.x`, `1.1.x` are supported",
		},
		{
			name:          "invalid yaml",
			input:         "arazzo: [\n",
			expectedError: "failed to unmarshal Arazzo document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := upgradeArazzo(t.Context(), bytes.NewBufferString(tt.input), bytes.NewBuffer(nil), bytes.NewBuffer(nil))
			require.Error(t, err, "upgrade should fail")
			assert.Contains(t, err.Error(), tt.expectedError, "error should match")
		})
	}
}

const upgradeCommandInput = "arazzo: 1.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n"

func TestRunUpgradeWithIO_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		flags           upgradeFlags
		expectedVersion string
	}{
		{name: "defaults to latest version", flags: upgradeFlags{}, expectedVersion: "arazzo: 1.1.0\n"},
		{name: "target version", flags: upgradeFlags{targetVersion: "1.0.1"}, expectedVersion: "arazzo: 1.0.1\n"},
		{name: "minor only", flags: upgradeFlags{minorOnly: true}, expectedVersion: "arazzo: 1.1.0\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" from stdin to stdout", func(t *testing.T) {
			t.Parallel()

			stdout := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)
			err := runUpgradeWithIO(t.Context(), nil, tt.flags, upgradeIO{stdin: bytes.NewBufferString(upgradeCommandInput), stdout: stdout, stderr: stderr})
			require.NoError(t, err, "upgrade should succeed")
			assert.Contains(t, stdout.String(), tt.expectedVersion, "stdout should contain the upgraded document")
			assert.Contains(t, stderr.String(), "Processing Arazzo document from stdin", "stderr should report stdin input")
		})
	}
}

func TestRunUpgradeWithIO_OutputFile_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.arazzo.yaml")
	outputFile := filepath.Join(dir, "output.arazzo.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(upgradeCommandInput), 0o600), "write input should succeed")

	stderr := bytes.NewBuffer(nil)
	err := runUpgradeWithIO(t.Context(), []string{inputFile, outputFile}, upgradeFlags{}, upgradeIO{stdout: bytes.NewBuffer(nil), stderr: stderr})
	require.NoError(t, err, "upgrade should succeed")

	output, err := os.ReadFile(outputFile)
	require.NoError(t, err, "read output should succeed")
	assert.Contains(t, string(output), "arazzo: 1.1.0", "output file should contain the upgraded document")
	assert.Contains(t, stderr.String(), "Document written to: "+outputFile, "stderr should report the output file")
}

func TestRunUpgradeWithIO_WriteInPlace_Success(t *testing.T) {
	t.Parallel()

	inputFile := filepath.Join(t.TempDir(), "input.arazzo.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(upgradeCommandInput), 0o600), "write input should succeed")

	err := runUpgradeWithIO(t.Context(), []string{inputFile}, upgradeFlags{writeInPlace: true}, upgradeIO{stdout: bytes.NewBuffer(nil), stderr: bytes.NewBuffer(nil)})
	require.NoError(t, err, "upgrade should succeed")

	output, err := os.ReadFile(inputFile)
	require.NoError(t, err, "read input should succeed")
	assert.Contains(t, string(output), "arazzo: 1.1.0", "input file should be upgraded in place")
}

func TestRunUpgradeWithIO_Error(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.arazzo.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(upgradeCommandInput), 0o600), "write input should succeed")
	unsupportedFile := filepath.Join(dir, "unsupported.arazzo.yaml")
	require.NoError(t, os.WriteFile(unsupportedFile, []byte("arazzo: 2.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n"), 0o600), "write input should succeed")

	tests := []struct {
		name          string
		args          []string
		flags         upgradeFlags
		expectedError string
	}{
		{name: "version and minor only", args: []string{inputFile}, flags: upgradeFlags{targetVersion: "1.1.0", minorOnly: true}, expectedError: "--version and --minor-only are mutually exclusive"},
		{name: "write with stdin", args: nil, flags: upgradeFlags{writeInPlace: true}, expectedError: "cannot use --write flag when reading from stdin"},
		{name: "write with output file", args: []string{inputFile, filepath.Join(dir, "out.yaml")}, flags: upgradeFlags{writeInPlace: true}, expectedError: "cannot specify output file when using --write flag"},
		{name: "missing input file", args: []string{filepath.Join(dir, "missing.yaml")}, expectedError: "failed to open input file"},
		{name: "unsupported version", args: []string{unsupportedFile}, expectedError: "failed to upgrade Arazzo document"},
		{name: "unwritable output file", args: []string{inputFile, filepath.Join(dir, "missing", "out.yaml")}, expectedError: "failed to write output file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runUpgradeWithIO(t.Context(), tt.args, tt.flags, upgradeIO{stdin: bytes.NewBuffer(nil), stdout: bytes.NewBuffer(nil), stderr: bytes.NewBuffer(nil)})
			require.Error(t, err, "upgrade should fail")
			assert.Contains(t, err.Error(), tt.expectedError, "error should match")
		})
	}
}

func TestApply_Success(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "arazzo"}
	Apply(root)

	names := []string{}
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Name())
	}
	assert.ElementsMatch(t, []string{"validate", "upgrade", "lint", "list-rules"}, names, "arazzo commands should be registered")
	assert.Equal(t, upgradeCmd, GetUpgradeCommand(), "should return the upgrade command")
}
