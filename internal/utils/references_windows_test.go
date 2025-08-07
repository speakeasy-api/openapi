//go:build windows

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsPathClassification_Success(t *testing.T) {
	tests := []struct {
		name               string
		windowsPath        string
		expectedType       ReferenceType
		expectedIsURL      bool
		expectedIsFile     bool
		expectedIsFragment bool
	}{
		{
			name:               "absolute windows path with drive letter",
			windowsPath:        "C:\\path\\to\\schemas\\user.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "windows path with different drive",
			windowsPath:        "D:\\projects\\api\\schema.yaml",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "windows UNC path",
			windowsPath:        "\\\\server\\share\\path\\file.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classification, err := ClassifyReference(tt.windowsPath)
			require.NoError(t, err)
			require.NotNil(t, classification)

			assert.Equal(t, tt.expectedType, classification.Type)
			assert.Equal(t, tt.expectedIsURL, classification.IsURL)
			assert.Equal(t, tt.expectedIsFile, classification.IsFile)
			assert.Equal(t, tt.expectedIsFragment, classification.IsFragment)
			assert.Nil(t, classification.ParsedURL, "Windows paths should not have a parsed URL")
		})
	}
}

func TestWindowsPathJoining_Success(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestWindowsPathJoinReference_Success(t *testing.T) {
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
			result, err := JoinReference(tt.base, tt.relative)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
