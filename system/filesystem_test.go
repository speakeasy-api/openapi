package system

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSystem_Open_Success(t *testing.T) {
	t.Parallel()

	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")
	err := os.WriteFile(testFile, testContent, 0o644)
	require.NoError(t, err, "should create test file")

	fsys := &FileSystem{}
	file, err := fsys.Open(testFile)

	require.NoError(t, err, "should open file successfully")
	require.NotNil(t, file, "should return non-nil file")
	defer file.Close()

	// Verify file can be read
	content := make([]byte, len(testContent))
	n, err := file.Read(content)
	require.NoError(t, err, "should read file content")
	assert.Equal(t, len(testContent), n, "should read correct number of bytes")
	assert.Equal(t, testContent, content, "should read correct content")
}

func TestFileSystem_Open_Error(t *testing.T) {
	t.Parallel()

	fsys := &FileSystem{}
	file, err := fsys.Open("nonexistent-file.txt")

	assert.Error(t, err, "should return error for nonexistent file")
	assert.Nil(t, file, "should return nil file on error")
}

func TestFileSystem_WriteFile_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "test.txt")
	testContent := []byte("test content")

	fsys := &FileSystem{}
	err := fsys.WriteFile(testFile, testContent, 0o644)

	require.NoError(t, err, "should write file successfully")

	// Verify file was written
	content, err := os.ReadFile(testFile)
	require.NoError(t, err, "should read written file")
	assert.Equal(t, testContent, content, "should have correct content")

	// Verify file permissions
	info, err := os.Stat(testFile)
	require.NoError(t, err, "should stat file")
	assert.Equal(t, fs.FileMode(0o644), info.Mode().Perm(), "should have correct permissions")
}

func TestFileSystem_WriteFile_CreatesDirectories(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "level1", "level2", "level3", "test.txt")
	testContent := []byte("test content")

	fsys := &FileSystem{}
	err := fsys.WriteFile(testFile, testContent, 0o644)

	require.NoError(t, err, "should write file and create directories")

	// Verify directories were created
	dirInfo, err := os.Stat(filepath.Dir(testFile))
	require.NoError(t, err, "should stat created directory")
	assert.True(t, dirInfo.IsDir(), "should be a directory")

	// Verify file was written
	content, err := os.ReadFile(testFile)
	require.NoError(t, err, "should read written file")
	assert.Equal(t, testContent, content, "should have correct content")
}

func TestFileSystem_WriteFile_OverwritesExisting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write initial content
	initialContent := []byte("initial content")
	err := os.WriteFile(testFile, initialContent, 0o644)
	require.NoError(t, err, "should create initial file")

	// Overwrite with new content
	newContent := []byte("new content")
	fsys := &FileSystem{}
	err = fsys.WriteFile(testFile, newContent, 0o644)

	require.NoError(t, err, "should overwrite file successfully")

	// Verify new content
	content, err := os.ReadFile(testFile)
	require.NoError(t, err, "should read file")
	assert.Equal(t, newContent, content, "should have new content")
}

func TestFileSystem_MkdirAll_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "level1", "level2", "level3")

	fsys := &FileSystem{}
	err := fsys.MkdirAll(testPath, 0o755)

	require.NoError(t, err, "should create directories successfully")

	// Verify directories were created
	info, err := os.Stat(testPath)
	require.NoError(t, err, "should stat created directory")
	assert.True(t, info.IsDir(), "should be a directory")
	assert.Equal(t, fs.FileMode(0o755), info.Mode().Perm(), "should have correct permissions")
}

func TestFileSystem_MkdirAll_ExistingDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "existing")

	// Create directory first
	err := os.MkdirAll(testPath, 0o755)
	require.NoError(t, err, "should create initial directory")

	// Call MkdirAll on existing directory
	fsys := &FileSystem{}
	err = fsys.MkdirAll(testPath, 0o755)

	require.NoError(t, err, "should succeed for existing directory")

	// Verify directory still exists
	info, err := os.Stat(testPath)
	require.NoError(t, err, "should stat directory")
	assert.True(t, info.IsDir(), "should be a directory")
}

func TestFileSystem_ImplementsInterfaces(t *testing.T) {
	t.Parallel()

	fsys := &FileSystem{}

	// Test VirtualFS interface
	var _ VirtualFS = fsys
	var _ fs.FS = fsys

	// Test WritableVirtualFS interface
	var _ WritableVirtualFS = fsys
}
