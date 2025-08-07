package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowsStylePathJoining tests Windows-style path joining logic
// This test simulates the Windows path behavior to verify our fixes
func TestWindowsStylePathJoining_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		relative string
		expected string
	}{
		{
			name:     "windows path with simple relative file",
			base:     "C:\\path\\to\\schemas\\user.json",
			relative: "common.json",
			expected: "C:\\path\\to\\schemas\\common.json",
		},
		{
			name:     "windows path with relative directory navigation",
			base:     "C:\\path\\to\\schemas\\user.json",
			relative: "..\\base\\common.json",
			expected: "C:\\path\\to\\base\\common.json",
		},
		{
			name:     "windows path with dot relative path",
			base:     "C:\\path\\to\\schemas\\user.json",
			relative: ".\\common.json",
			expected: "C:\\path\\to\\schemas\\common.json",
		},
		{
			name:     "windows path with absolute relative path",
			base:     "C:\\path\\to\\schemas\\user.json",
			relative: "D:\\other\\path\\schema.json",
			expected: "D:\\other\\path\\schema.json",
		},
		{
			name:     "windows path with fragment",
			base:     "C:\\path\\to\\schema.json",
			relative: "#/definitions/User",
			expected: "C:\\path\\to\\schema.json#/definitions/User",
		},
		{
			name:     "windows path with unix-style dot relative path",
			base:     "C:\\path\\to\\schemas\\user.json",
			relative: "./common.json",
			expected: "C:\\path\\to\\schemas\\common.json",
		},
		{
			name:     "windows path with unix-style relative directory navigation",
			base:     "C:\\path\\to\\schemas\\user.json",
			relative: "../base/common.json",
			expected: "C:\\path\\to\\base\\common.json",
		},
		{
			name:     "windows path with unix-style complex relative path",
			base:     "D:\\a\\openapi\\openapi\\jsonschema\\oas3\\testdata\\resolve_test_main.yaml",
			relative: "./resolve_test_external.yaml",
			expected: "D:\\a\\openapi\\openapi\\jsonschema\\oas3\\testdata\\resolve_test_external.yaml",
		},
		{
			name:     "windows UNC path joining",
			base:     "\\\\server\\share\\path\\base.json",
			relative: "schema.json",
			expected: "\\\\server\\share\\path\\schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			classification, err := ClassifyReference(tt.base)
			require.NoError(t, err)
			require.NotNil(t, classification)
			require.True(t, classification.IsFile, "Base should be classified as file path")

			result, err := classification.JoinWith(tt.relative)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWindowsStylePathJoinReference_Success tests the convenience function
func TestWindowsStylePathJoinReference_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		relative string
		expected string
	}{
		{
			name:     "windows path joining via convenience function",
			base:     "C:\\path\\to\\base.json",
			relative: "schema.json",
			expected: "C:\\path\\to\\schema.json",
		},
		{
			name:     "windows UNC path joining",
			base:     "\\\\server\\share\\path\\base.json",
			relative: "schema.json",
			expected: "\\\\server\\share\\path\\schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := JoinReference(tt.base, tt.relative)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUnixStylePathJoining_Success tests that Unix-style paths still work correctly
func TestUnixStylePathJoining_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		relative string
		expected string
	}{
		{
			name:     "unix path with simple relative file",
			base:     "/path/to/schemas/user.json",
			relative: "common.json",
			expected: "/path/to/schemas/common.json",
		},
		{
			name:     "unix path with relative directory navigation",
			base:     "/path/to/schemas/user.json",
			relative: "../base/common.json",
			expected: "/path/to/base/common.json",
		},
		{
			name:     "unix path with dot relative path",
			base:     "/path/to/schemas/user.json",
			relative: "./common.json",
			expected: "/path/to/schemas/common.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			classification, err := ClassifyReference(tt.base)
			require.NoError(t, err)
			require.NotNil(t, classification)
			require.True(t, classification.IsFile, "Base should be classified as file path")

			result, err := classification.JoinWith(tt.relative)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
