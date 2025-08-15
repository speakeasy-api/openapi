package utils

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyReference_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		reference          string
		expectedType       ReferenceType
		expectedIsURL      bool
		expectedIsFile     bool
		expectedIsFragment bool
	}{
		// URL cases
		{
			name:               "http URL",
			reference:          "http://example.com/api/schema.json",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		{
			name:               "https URL",
			reference:          "https://api.example.com/v1/openapi.yaml",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		{
			name:               "ftp URL",
			reference:          "ftp://files.example.com/schemas/user.json",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		{
			name:               "file URL",
			reference:          "file:///path/to/schema.json",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		{
			name:               "custom scheme URL",
			reference:          "custom://example.com/resource",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		// File path cases
		{
			name:               "absolute unix path",
			reference:          "/path/to/schema.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "absolute windows path",
			reference:          "C:\\path\\to\\schema.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "relative path with dot",
			reference:          "./schemas/user.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "relative path with double dot",
			reference:          "../common/schemas.yaml",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "relative path with forward slash",
			reference:          "schemas/user.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "relative path with backslash",
			reference:          "schemas\\user.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		// Fragment cases
		{
			name:               "simple fragment",
			reference:          "#/components/schemas/User",
			expectedType:       ReferenceTypeFragment,
			expectedIsURL:      false,
			expectedIsFile:     false,
			expectedIsFragment: true,
		},
		{
			name:               "complex fragment",
			reference:          "#/paths/~1users~1{id}/get/responses/200",
			expectedType:       ReferenceTypeFragment,
			expectedIsURL:      false,
			expectedIsFile:     false,
			expectedIsFragment: true,
		},
		{
			name:               "root fragment",
			reference:          "#/",
			expectedType:       ReferenceTypeFragment,
			expectedIsURL:      false,
			expectedIsFile:     false,
			expectedIsFragment: true,
		},
		{
			name:               "empty fragment",
			reference:          "#",
			expectedType:       ReferenceTypeFragment,
			expectedIsURL:      false,
			expectedIsFile:     false,
			expectedIsFragment: true,
		},
		// Ambiguous cases (default to file path)
		{
			name:               "simple filename",
			reference:          "schema.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "filename without extension",
			reference:          "schema",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ClassifyReference(tt.reference)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedIsURL, result.IsURL)
			assert.Equal(t, tt.expectedIsFile, result.IsFile)
			assert.Equal(t, tt.expectedIsFragment, result.IsFragment)
			assert.Equal(t, tt.reference, result.Original)
		})
	}
}

func TestClassifyReference_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		reference string
	}{
		{
			name:      "empty string",
			reference: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ClassifyReference(tt.reference)
			require.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestIsURL_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		reference string
		expected  bool
	}{
		{
			name:      "http URL",
			reference: "http://example.com/schema.json",
			expected:  true,
		},
		{
			name:      "https URL",
			reference: "https://api.example.com/openapi.yaml",
			expected:  true,
		},
		{
			name:      "file path",
			reference: "/path/to/schema.json",
			expected:  false,
		},
		{
			name:      "relative path",
			reference: "./schema.json",
			expected:  false,
		},
		{
			name:      "fragment",
			reference: "#/components/schemas/User",
			expected:  false,
		},
		{
			name:      "empty string",
			reference: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsURL(tt.reference)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFilePath_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		reference string
		expected  bool
	}{
		{
			name:      "absolute path",
			reference: "/path/to/schema.json",
			expected:  true,
		},
		{
			name:      "relative path",
			reference: "./schema.json",
			expected:  true,
		},
		{
			name:      "windows path",
			reference: "C:\\path\\to\\schema.json",
			expected:  true,
		},
		{
			name:      "simple filename",
			reference: "schema.json",
			expected:  true,
		},
		{
			name:      "http URL",
			reference: "http://example.com/schema.json",
			expected:  false,
		},
		{
			name:      "fragment",
			reference: "#/components/schemas/User",
			expected:  false,
		},
		{
			name:      "empty string",
			reference: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsFilePath(tt.reference)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReferenceClassification_JoinWith_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		base     string
		relative string
		expected string
	}{
		// URL joining tests
		{
			name:     "URL with relative path",
			base:     "https://api.example.com/v1/schemas/",
			relative: "user.json",
			expected: "https://api.example.com/v1/schemas/user.json",
		},
		{
			name:     "URL with relative path up directory",
			base:     "https://api.example.com/v1/schemas/user.json",
			relative: "../common/base.json",
			expected: "https://api.example.com/v1/common/base.json",
		},
		{
			name:     "URL with absolute path",
			base:     "https://api.example.com/v1/schemas/user.json",
			relative: "/v2/schemas/user.json",
			expected: "https://api.example.com/v2/schemas/user.json",
		},
		{
			name:     "URL with fragment",
			base:     "https://api.example.com/schema.json",
			relative: "#/definitions/User",
			expected: "https://api.example.com/schema.json#/definitions/User",
		},
		{
			name:     "URL with existing fragment replaced",
			base:     "https://api.example.com/schema.json#/old",
			relative: "#/definitions/User",
			expected: "https://api.example.com/schema.json#/definitions/User",
		},
		// File path joining tests
		{
			name:     "file path with relative path",
			base:     "/path/to/schemas/user.json",
			relative: "common.json",
			expected: "/path/to/schemas/common.json",
		},
		{
			name:     "file path with relative path up directory",
			base:     "/path/to/schemas/user.json",
			relative: "../base/common.json",
			expected: "/path/to/base/common.json",
		},
		{
			name:     "file path with dot relative path",
			base:     "/path/to/schemas/user.json",
			relative: "./common.json",
			expected: "/path/to/schemas/common.json",
		},
		{
			name:     "file path with absolute relative path",
			base:     "/path/to/schemas/user.json",
			relative: "/other/path/schema.json",
			expected: "/other/path/schema.json",
		},
		{
			name:     "file path with fragment",
			base:     "/path/to/schema.json",
			relative: "#/definitions/User",
			expected: "/path/to/schema.json#/definitions/User",
		},
		// Fragment base tests
		{
			name:     "fragment base with relative path",
			base:     "#/components/schemas/User",
			relative: "common.json",
			expected: "common.json",
		},
		// Empty relative tests
		{
			name:     "empty relative returns base",
			base:     "https://api.example.com/schema.json",
			relative: "",
			expected: "https://api.example.com/schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			classification, err := ClassifyReference(tt.base)
			require.NoError(t, err)
			require.NotNil(t, classification)

			result, err := classification.JoinWith(tt.relative)
			require.NoError(t, err)
			// Clean both paths to normalize separators for cross-platform compatibility
			assert.Equal(t, filepath.Clean(tt.expected), filepath.Clean(result))
		})
	}
}

func TestReferenceClassification_JoinWith_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		base     string
		relative string
	}{
		{
			name:     "invalid relative URL",
			base:     "https://api.example.com/schema.json",
			relative: "ht tp://invalid url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			classification, err := ClassifyReference(tt.base)
			require.NoError(t, err)
			require.NotNil(t, classification)

			result, err := classification.JoinWith(tt.relative)
			require.Error(t, err)
			assert.Empty(t, result)
		})
	}
}

func TestJoinReference_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		base     string
		relative string
		expected string
	}{
		{
			name:     "URL joining",
			base:     "https://api.example.com/v1/",
			relative: "schema.json",
			expected: "https://api.example.com/v1/schema.json",
		},
		{
			name:     "file path joining",
			base:     "/path/to/base.json",
			relative: "schema.json",
			expected: "/path/to/schema.json",
		},
		{
			name:     "empty base returns relative",
			base:     "",
			relative: "schema.json",
			expected: "schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := JoinReference(tt.base, tt.relative)
			require.NoError(t, err)
			// Clean both paths to normalize separators for cross-platform compatibility
			assert.Equal(t, filepath.Clean(tt.expected), filepath.Clean(result))
		})
	}
}

func TestJoinReference_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		base     string
		relative string
	}{
		{
			name:     "invalid base reference",
			base:     "ht tp://invalid url",
			relative: "schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := JoinReference(tt.base, tt.relative)
			require.Error(t, err)
			assert.Empty(t, result)
		})
	}
}

func TestReferenceClassification_CachedURL(t *testing.T) {
	t.Parallel()
	// Test that URL parsing is cached and reused
	classification, err := ClassifyReference("https://api.example.com/schema.json")
	require.NoError(t, err)
	require.NotNil(t, classification)
	require.True(t, classification.IsURL)
	require.NotNil(t, classification.ParsedURL)

	// Verify the cached URL is correct
	assert.Equal(t, "https", classification.ParsedURL.Scheme)
	assert.Equal(t, "api.example.com", classification.ParsedURL.Host)
	assert.Equal(t, "/schema.json", classification.ParsedURL.Path)

	// Test that JoinWith uses the cached URL
	result, err := classification.JoinWith("user.json")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/user.json", result)
}

func TestIsFragment_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		reference string
		expected  bool
	}{
		{
			name:      "simple fragment",
			reference: "#/components/schemas/User",
			expected:  true,
		},
		{
			name:      "complex fragment",
			reference: "#/paths/~1users~1{id}/get/responses/200",
			expected:  true,
		},
		{
			name:      "empty fragment",
			reference: "#",
			expected:  true,
		},
		{
			name:      "file path",
			reference: "/path/to/schema.json",
			expected:  false,
		},
		{
			name:      "http URL",
			reference: "http://example.com/schema.json",
			expected:  false,
		},
		{
			name:      "empty string",
			reference: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsFragment(tt.reference)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReferenceClassification_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		reference          string
		expectedType       ReferenceType
		expectedIsURL      bool
		expectedIsFile     bool
		expectedIsFragment bool
	}{
		{
			name:               "URL with fragment",
			reference:          "https://example.com/schema.json#/definitions/User",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		{
			name:               "URL with query parameters",
			reference:          "https://api.example.com/schema?version=v1",
			expectedType:       ReferenceTypeURL,
			expectedIsURL:      true,
			expectedIsFile:     false,
			expectedIsFragment: false,
		},
		{
			name:               "file path with spaces",
			reference:          "/path/to/my schema.json",
			expectedType:       ReferenceTypeFilePath,
			expectedIsURL:      false,
			expectedIsFile:     true,
			expectedIsFragment: false,
		},
		{
			name:               "fragment with special characters",
			reference:          "#/components/schemas/User%20Profile",
			expectedType:       ReferenceTypeFragment,
			expectedIsURL:      false,
			expectedIsFile:     false,
			expectedIsFragment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ClassifyReference(tt.reference)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedIsURL, result.IsURL)
			assert.Equal(t, tt.expectedIsFile, result.IsFile)
			assert.Equal(t, tt.expectedIsFragment, result.IsFragment)
		})
	}
}
