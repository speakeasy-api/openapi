package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOverlay_Success(t *testing.T) {
	t.Parallel()

	// Use existing test data from overlay/testdata
	testDataPath := filepath.Join("..", "testdata", "overlay-generated.yaml")

	result, err := LoadOverlay(testDataPath)

	require.NoError(t, err, "should load overlay successfully")
	require.NotNil(t, result, "should return non-nil overlay")
	assert.NotEmpty(t, result.Version, "overlay should have version")
}

func TestLoadOverlay_Error_InvalidPath(t *testing.T) {
	t.Parallel()

	result, err := LoadOverlay("nonexistent-file.yaml")

	require.Error(t, err, "should return error for nonexistent file")
	assert.Nil(t, result, "should return nil overlay on error")
	assert.Contains(t, err.Error(), "failed to parse overlay", "error should mention parsing failure")
}

func TestLoadOverlay_Error_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0o644)
	require.NoError(t, err, "should create test file")

	result, err := LoadOverlay(invalidFile)

	require.Error(t, err, "should return error for invalid YAML")
	assert.Nil(t, result, "should return nil overlay on error")
}

func TestLoadOverlay_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(emptyFile, []byte(""), 0o644)
	require.NoError(t, err, "should create test file")

	result, err := LoadOverlay(emptyFile)

	require.NoError(t, err, "should handle empty file")
	require.NotNil(t, result, "should return non-nil overlay for empty file")
	assert.Empty(t, result.Version, "empty file should have empty version")
}
