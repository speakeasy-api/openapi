package overlay_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOverlay_Format_Success(t *testing.T) {
	t.Parallel()

	// Create a test overlay
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title:   "Test Overlay",
			Version: "1.0.0",
		},
		Actions: []overlay.Action{
			{
				Target: "$.info.title",
				Update: yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "New Title",
					Tag:   "!!str",
				},
			},
		},
	}

	var buf bytes.Buffer
	err := o.Format(&buf)

	require.NoError(t, err, "Format should not return error")
	assert.Contains(t, buf.String(), "overlay:", "output should contain overlay key")
	assert.Contains(t, buf.String(), "info:", "output should contain info key")
	assert.Contains(t, buf.String(), "actions:", "output should contain actions key")
}

func TestFormat_FileFormat_Success(t *testing.T) {
	t.Parallel()

	// Create a temp directory for the test
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-overlay.yaml")

	// Create a simple overlay file
	overlayContent := `overlay: "1.0.0"
info:
  title: Test
  version: "1.0"
actions:
  - target: $.info
    update:
      title: NewTitle
`
	err := os.WriteFile(testFile, []byte(overlayContent), 0o644)
	require.NoError(t, err, "setup: should create test file")

	// Test Format function
	err = overlay.Format(testFile)
	require.NoError(t, err, "Format should succeed")

	// Verify file was reformatted (exists and is valid)
	_, err = os.Stat(testFile)
	require.NoError(t, err, "file should still exist after formatting")
}

func TestFormat_InvalidPath_Error(t *testing.T) {
	t.Parallel()

	err := overlay.Format("/nonexistent/path/overlay.yaml")

	require.Error(t, err, "Format should return error for non-existent file")
}
