package arazzo

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

const (
	lintCleanFixture  = "../../../../arazzo/testdata/lint/clean.arazzo.yaml"
	lintIssuesFixture = "../../../../arazzo/testdata/lint/issues.arazzo.yaml"
)

// writeLintConfig writes a lint config file and returns its path.
func writeLintConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "arazzo-lint.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600), "config should be written")
	return path
}

// copyIssuesFixture copies the issues fixture to a temporary file so it can be fixed.
func copyIssuesFixture(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(lintIssuesFixture)
	require.NoError(t, err, "fixture should be read")
	path := filepath.Join(t.TempDir(), "issues.arazzo.yaml")
	require.NoError(t, os.WriteFile(path, data, 0o600), "fixture should be copied")
	return path
}

// runLint runs the lint command with an empty config and returns stdout and stderr.
func runLint(t *testing.T, file string, flags lintFlags, stdin string) (string, string, error) {
	t.Helper()

	if flags.configFile == "" {
		flags.configFile = writeLintConfig(t, "extends: all\n")
	}
	var stdout, stderr bytes.Buffer
	err := lintArazzo(t.Context(), file, flags, lintIO{stdin: strings.NewReader(stdin), stdout: &stdout, stderr: &stderr})
	return stdout.String(), stderr.String(), err
}

func TestLintArazzo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		file           string
		stdin          string
		flags          lintFlags
		expectedStdout []string
		expectedStderr string
	}{
		{
			name:           "clean document checked against its sources",
			file:           lintCleanFixture,
			expectedStdout: []string{filepath.Clean(lintCleanFixture)},
			expectedStderr: "Linting Arazzo document: " + filepath.Clean(lintCleanFixture),
		},
		{
			name:           "json output with summary",
			file:           lintCleanFixture,
			flags:          lintFlags{format: "json", summary: true},
			expectedStdout: []string{`"results"`},
		},
		{
			name: "stdin",
			file: "-",
			stdin: `arazzo: 1.1.0
info:
  title: Stdin
  description: Read from stdin.
  version: 1.0.0
sourceDescriptions: []
workflows: []
`,
			flags:          lintFlags{skipSources: true},
			expectedStdout: []string{"stdin"},
			expectedStderr: "Linting Arazzo document from stdin",
		},
		{
			name:           "disabled rules and recommended ruleset only report warnings",
			file:           lintIssuesFixture,
			flags:          lintFlags{ruleset: "recommended", skipSources: true, disableRules: []string{"semantic-dependency-cycle"}},
			expectedStdout: []string{"style-info-description", "security-source-description-https"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := runLint(t, tt.file, tt.flags, tt.stdin)
			require.NoError(t, err, "lint should succeed")
			for _, expected := range tt.expectedStdout {
				assert.Contains(t, stdout, expected, "stdout should contain expected output")
			}
			assert.Contains(t, stderr, tt.expectedStderr, "stderr should contain expected output")
		})
	}
}

func TestLintArazzo_JSONOutput_Success(t *testing.T) {
	t.Parallel()

	stdout, _, err := runLint(t, lintIssuesFixture, lintFlags{format: "json", skipSources: true, disableRules: []string{"semantic-dependency-cycle"}}, "")
	require.NoError(t, err, "lint should succeed without errors")

	var output map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &output), "stdout should be JSON")
	assert.Contains(t, output, "results", "output should contain results")
}

func TestLintArazzo_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		file          string
		stdin         string
		flags         lintFlags
		expectedError string
	}{
		{name: "fix flags are exclusive", file: lintCleanFixture, flags: lintFlags{fix: true, fixInteractive: true}, expectedError: "mutually exclusive"},
		{name: "dry run needs a fix flag", file: lintCleanFixture, flags: lintFlags{dryRun: true}, expectedError: "--dry-run requires"},
		{name: "fixes aren't supported for stdin", file: "-", flags: lintFlags{fix: true}, expectedError: "not supported when reading from stdin"},
		{name: "unknown ruleset", file: lintCleanFixture, flags: lintFlags{ruleset: "strict"}, expectedError: "unknown ruleset `strict`"},
		{name: "missing config file", file: lintCleanFixture, flags: lintFlags{configFile: "missing.yaml"}, expectedError: "missing.yaml"},
		{name: "missing file", file: "missing.arazzo.yaml", expectedError: "failed to open file"},
		{name: "invalid document", file: "-", stdin: "{", expectedError: "failed to unmarshal file"},
		{name: "lint errors", file: lintIssuesFixture, flags: lintFlags{skipSources: true, summary: true}, expectedError: "linting found 2 errors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := runLint(t, tt.file, tt.flags, tt.stdin)
			require.Error(t, err, "lint should fail")
			assert.Contains(t, err.Error(), tt.expectedError, "error should match")
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestLintArazzo_StdinReadError(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := lintArazzo(t.Context(), "-", lintFlags{configFile: writeLintConfig(t, "extends: all\n")}, lintIO{stdin: failingReader{}, stdout: &stdout, stderr: &stderr})
	require.Error(t, err, "lint should fail")
	assert.Contains(t, err.Error(), "failed to read stdin", "error should match")
}

func TestLintArazzo_Fix_Success(t *testing.T) {
	t.Parallel()

	file := copyIssuesFixture(t)
	_, stderr, err := runLint(t, file, lintFlags{fix: true, skipSources: true}, "")
	require.Error(t, err, "the dependency cycles can't be fixed")

	assert.Contains(t, stderr, "Applied 2 fix(es)", "unused components should be removed")
	assert.Contains(t, stderr, "(requires interactive input)", "interactive fixes should be skipped")

	data, err := os.ReadFile(file)
	require.NoError(t, err, "fixed file should be read")
	assert.NotContains(t, string(data), "unusedParam", "unused parameter should be removed")
	assert.NotContains(t, string(data), "components:", "empty components should be removed")
}

func TestLintArazzo_FixDryRun_Success(t *testing.T) {
	t.Parallel()

	file := copyIssuesFixture(t)
	before, err := os.ReadFile(file)
	require.NoError(t, err, "file should be read")

	_, stderr, err := runLint(t, file, lintFlags{fix: true, dryRun: true, skipSources: true}, "")
	require.Error(t, err, "the dependency cycles are still reported")
	assert.Contains(t, stderr, "[dry-run] Fixed:", "fixes should be previewed")

	after, err := os.ReadFile(file)
	require.NoError(t, err, "file should be read")
	assert.Equal(t, string(before), string(after), "dry run should not change the file")
}

func TestLintArazzo_FixInteractive_Success(t *testing.T) {
	t.Parallel()

	file := copyIssuesFixture(t)
	_, stderr, err := runLint(t, file, lintFlags{fixInteractive: true, skipSources: true}, "Lints pets.\n1\n")
	require.Error(t, err, "the dependency cycles can't be fixed")
	assert.Contains(t, stderr, "Applied 4 fix(es)", "all fixes should be applied")

	data, err := os.ReadFile(file)
	require.NoError(t, err, "fixed file should be read")

	var doc struct {
		Info struct {
			Description string `yaml:"description"`
		} `yaml:"info"`
		SourceDescriptions []struct {
			Type string `yaml:"type"`
		} `yaml:"sourceDescriptions"`
	}
	require.NoError(t, yaml.Unmarshal(data, &doc), "fixed file should parse")
	assert.Equal(t, "Lints pets.", doc.Info.Description, "description should be set")
	assert.Equal(t, "openapi", doc.SourceDescriptions[0].Type, "type should be set")
}

//nolint:paralleltest // modifies HOME to test the default config location
func TestLoadArazzoLintConfig_HomeConfig_Success(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	config, err := loadArazzoLintConfig(lintFlags{})
	require.NoError(t, err, "a missing home config should be ignored")
	assert.Equal(t, []string{"all"}, config.Extends, "default config should be used")

	require.NoError(t, os.MkdirAll(filepath.Join(home, ".openapi"), 0o700), "config dir should be created")
	require.NoError(t, os.WriteFile(filepath.Join(home, ".openapi", "arazzo-lint.yaml"), []byte("extends: recommended\n"), 0o600), "config should be written")

	config, err = loadArazzoLintConfig(lintFlags{format: "json"})
	require.NoError(t, err, "config should load")
	assert.Equal(t, []string{"recommended"}, config.Extends, "home config should be used")
	assert.Equal(t, "json", string(config.OutputFormat), "output format should be set")

	require.NoError(t, os.WriteFile(filepath.Join(home, ".openapi", "arazzo-lint.yaml"), []byte("extends: [\n"), 0o600), "config should be written")
	_, err = loadArazzoLintConfig(lintFlags{})
	require.Error(t, err, "an invalid home config should be reported")
}

func TestValidateArazzoLintFlags_Success(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateArazzoLintFlags(nil, []string{lintCleanFixture}), "default flags should be valid")
	assert.NotNil(t, GetLintCommand(), "lint command should be exposed")
}
