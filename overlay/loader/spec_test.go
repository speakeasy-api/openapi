package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetOverlayExtendsPath_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overlay   *overlay.Overlay
		wantValue string
	}{
		{
			name: "returns path from file URL",
			overlay: &overlay.Overlay{
				Extends: "file:///path/to/spec.yaml",
			},
			wantValue: "/path/to/spec.yaml",
		},
		{
			name: "returns path from file URL with host",
			overlay: &overlay.Overlay{
				Extends: "file://localhost/path/to/spec.yaml",
			},
			wantValue: "/path/to/spec.yaml",
		},
		{
			name: "returns decoded path from file URL",
			overlay: &overlay.Overlay{
				Extends: "file:///path/to/my%20spec.yaml",
			},
			wantValue: "/path/to/my spec.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetOverlayExtendsPath(tt.overlay)

			require.NoError(t, err, "should get path successfully")
			assert.Equal(t, tt.wantValue, result, "should return correct path")
		})
	}
}

func TestGetOverlayExtendsPath_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		overlay      *overlay.Overlay
		wantErrorMsg string
	}{
		{
			name: "returns error when extends is empty",
			overlay: &overlay.Overlay{
				Extends: "",
			},
			wantErrorMsg: "overlay does not specify an extends URL",
		},
		{
			name: "returns error for http URL",
			overlay: &overlay.Overlay{
				Extends: "http://example.com/spec.yaml",
			},
			wantErrorMsg: "only file:// extends URLs are supported",
		},
		{
			name: "returns error for https URL",
			overlay: &overlay.Overlay{
				Extends: "https://example.com/spec.yaml",
			},
			wantErrorMsg: "only file:// extends URLs are supported",
		},
		{
			name: "returns error for invalid URL",
			overlay: &overlay.Overlay{
				Extends: "://invalid-url",
			},
			wantErrorMsg: "failed to parse URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetOverlayExtendsPath(tt.overlay)

			assert.Error(t, err, "should return error")
			assert.Empty(t, result, "should return empty path on error")
			assert.Contains(t, err.Error(), tt.wantErrorMsg, "error should contain expected message")
		})
	}
}

func TestLoadSpecification_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	testFile := filepath.Join(tmpDir, "spec.yaml")
	err := os.WriteFile(testFile, []byte(yamlContent), 0o644)
	require.NoError(t, err, "should create test file")

	result, err := LoadSpecification(testFile)

	require.NoError(t, err, "should load specification successfully")
	require.NotNil(t, result, "should return non-nil node")
	assert.Equal(t, yaml.DocumentNode, result.Kind, "should be a document node")
}

func TestLoadSpecification_JSONFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonContent := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {}
}`
	testFile := filepath.Join(tmpDir, "spec.json")
	err := os.WriteFile(testFile, []byte(jsonContent), 0o644)
	require.NoError(t, err, "should create test file")

	result, err := LoadSpecification(testFile)

	require.NoError(t, err, "should load JSON specification successfully")
	require.NotNil(t, result, "should return non-nil node")
	assert.Equal(t, yaml.DocumentNode, result.Kind, "should be a document node")
}

func TestLoadSpecification_Error_FileNotFound(t *testing.T) {
	t.Parallel()

	result, err := LoadSpecification("nonexistent-file.yaml")

	assert.Error(t, err, "should return error for nonexistent file")
	assert.Nil(t, result, "should return nil node on error")
	assert.Contains(t, err.Error(), "failed to open schema", "error should mention opening failure")
}

func TestLoadSpecification_Error_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(testFile, []byte("invalid: yaml: [content"), 0o644)
	require.NoError(t, err, "should create test file")

	result, err := LoadSpecification(testFile)

	assert.Error(t, err, "should return error for invalid YAML")
	assert.Nil(t, result, "should return nil node on error")
	assert.Contains(t, err.Error(), "failed to parse schema", "error should mention parsing failure")
}

func TestLoadExtendsSpecification_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	testFile := filepath.Join(tmpDir, "spec.yaml")
	err := os.WriteFile(testFile, []byte(yamlContent), 0o644)
	require.NoError(t, err, "should create test file")

	o := &overlay.Overlay{
		Extends: "file://" + testFile,
	}

	result, err := LoadExtendsSpecification(o)

	require.NoError(t, err, "should load extends specification successfully")
	require.NotNil(t, result, "should return non-nil node")
	assert.Equal(t, yaml.DocumentNode, result.Kind, "should be a document node")
}

func TestLoadExtendsSpecification_Error_NoExtends(t *testing.T) {
	t.Parallel()

	o := &overlay.Overlay{
		Extends: "",
	}

	result, err := LoadExtendsSpecification(o)

	assert.Error(t, err, "should return error when extends is empty")
	assert.Nil(t, result, "should return nil node on error")
	assert.Contains(t, err.Error(), "overlay does not specify an extends URL", "error should mention missing extends")
}

func TestLoadExtendsSpecification_Error_InvalidURL(t *testing.T) {
	t.Parallel()

	o := &overlay.Overlay{
		Extends: "http://example.com/spec.yaml",
	}

	result, err := LoadExtendsSpecification(o)

	assert.Error(t, err, "should return error for non-file URL")
	assert.Nil(t, result, "should return nil node on error")
	assert.Contains(t, err.Error(), "only file:// extends URLs are supported", "error should mention unsupported URL scheme")
}

func TestLoadExtendsSpecification_Error_FileNotFound(t *testing.T) {
	t.Parallel()

	o := &overlay.Overlay{
		Extends: "file:///nonexistent/spec.yaml",
	}

	result, err := LoadExtendsSpecification(o)

	assert.Error(t, err, "should return error for nonexistent file")
	assert.Nil(t, result, "should return nil node on error")
	assert.Contains(t, err.Error(), "failed to open schema", "error should mention file opening failure")
}

func TestLoadEitherSpecification_WithPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	testFile := filepath.Join(tmpDir, "spec.yaml")
	err := os.WriteFile(testFile, []byte(yamlContent), 0o644)
	require.NoError(t, err, "should create test file")

	o := &overlay.Overlay{
		Extends: "file:///some/other/path.yaml",
	}

	result, name, err := LoadEitherSpecification(testFile, o)

	require.NoError(t, err, "should load specification from provided path")
	require.NotNil(t, result, "should return non-nil node")
	assert.Equal(t, testFile, name, "should return provided path as name")
	assert.Equal(t, yaml.DocumentNode, result.Kind, "should be a document node")
}

func TestLoadEitherSpecification_WithoutPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	testFile := filepath.Join(tmpDir, "spec.yaml")
	err := os.WriteFile(testFile, []byte(yamlContent), 0o644)
	require.NoError(t, err, "should create test file")

	o := &overlay.Overlay{
		Extends: "file://" + testFile,
	}

	result, name, err := LoadEitherSpecification("", o)

	require.NoError(t, err, "should load specification from extends URL")
	require.NotNil(t, result, "should return non-nil node")
	assert.Equal(t, testFile, name, "should return extends path as name")
	assert.Equal(t, yaml.DocumentNode, result.Kind, "should be a document node")
}

func TestLoadEitherSpecification_Error_NoPathNoExtends(t *testing.T) {
	t.Parallel()

	o := &overlay.Overlay{
		Extends: "",
	}

	result, name, err := LoadEitherSpecification("", o)

	assert.Error(t, err, "should return error when neither path nor extends provided")
	assert.Nil(t, result, "should return nil node on error")
	assert.Empty(t, name, "should return empty name on error")
}

func TestLoadEitherSpecification_Error_InvalidPath(t *testing.T) {
	t.Parallel()

	o := &overlay.Overlay{}

	result, name, err := LoadEitherSpecification("nonexistent-file.yaml", o)

	assert.Error(t, err, "should return error for invalid path")
	assert.Nil(t, result, "should return nil node on error")
	assert.Equal(t, "nonexistent-file.yaml", name, "should return attempted path as name")
}
