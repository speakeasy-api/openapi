package openapi_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitize_RemoveAllExtensions_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with various extensions
	inputFile, err := os.Open("testdata/sanitize/sanitize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Sanitize with default options (remove all extensions and clean components)
	result, err := openapi.Sanitize(ctx, inputDoc, nil)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/sanitize/sanitize_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Sanitized document should match expected output")
}

func TestSanitize_PatternBased_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with various extensions
	inputFile, err := os.Open("testdata/sanitize/sanitize_pattern_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Sanitize with pattern matching - only remove x-go-* extensions
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns:     []string{"x-go-*"},
		KeepUnusedComponents:  true,
		KeepUnknownProperties: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/sanitize/sanitize_pattern_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Pattern-based sanitized document should match expected output")
}

func TestSanitize_MultiplePatterns_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with various extensions
	inputFile, err := os.Open("testdata/sanitize/sanitize_multi_pattern_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Sanitize with multiple patterns
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns:    []string{"x-go-*", "x-internal-*"},
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/sanitize/sanitize_multi_pattern_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Multi-pattern sanitized document should match expected output")
}

func TestSanitize_KeepComponents_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Sanitize but keep unused components
	opts := &openapi.SanitizeOptions{
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output (all extensions removed, components kept)
	expectedBytes, err := os.ReadFile("testdata/sanitize/sanitize_keep_components_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Sanitized document with kept components should match expected output")
}

func TestSanitize_EmptyDocument_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with nil document
	result, err := openapi.Sanitize(ctx, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Test with minimal document (no components, no extensions)
	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	result, err = openapi.Sanitize(ctx, doc, nil)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")
}

func TestSanitize_NoExtensions_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load document without extensions
	inputFile, err := os.Open("testdata/sanitize/sanitize_no_extensions_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Sanitize (should be a no-op for extensions)
	result, err := openapi.Sanitize(ctx, inputDoc, nil)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Document should still be valid
	assert.NotNil(t, inputDoc)
}

func TestLoadSanitizeConfig_Success(t *testing.T) {
	t.Parallel()

	configYAML := `extensionPatterns:
  - "x-go-*"
  - "x-internal-*"
keepUnusedComponents: true
keepUnknownProperties: false
`

	// Load config from reader
	opts, err := openapi.LoadSanitizeConfig(strings.NewReader(configYAML))
	require.NoError(t, err)
	require.NotNil(t, opts)

	// Verify config was loaded correctly
	assert.Equal(t, []string{"x-go-*", "x-internal-*"}, opts.ExtensionPatterns)
	assert.True(t, opts.KeepUnusedComponents)
	assert.False(t, opts.KeepUnknownProperties)
}

func TestLoadSanitizeConfig_FileNotFound_Error(t *testing.T) {
	t.Parallel()

	// Try to load non-existent config file
	_, err := openapi.LoadSanitizeConfigFromFile("testdata/sanitize/nonexistent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open config file")
}

func TestLoadSanitizeConfig_InvalidYAML_Error(t *testing.T) {
	t.Parallel()

	invalidYAML := `extensionPatterns:
  - "x-go-*"
  invalid yaml syntax here: [
keepUnusedComponents: true
`

	// Try to load invalid YAML
	_, err := openapi.LoadSanitizeConfig(strings.NewReader(invalidYAML))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestSanitize_ConfigFile_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_pattern_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	configYAML := `extensionPatterns:
  - "x-go-*"
keepUnusedComponents: true
keepUnknownProperties: true
`

	// Load config from reader
	opts, err := openapi.LoadSanitizeConfig(strings.NewReader(configYAML))
	require.NoError(t, err)

	// Sanitize using config
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/sanitize/sanitize_pattern_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Config-based sanitized document should match expected output")
}

func TestSanitize_KeepExtensionsRemoveUnknownProperties_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load input with both extensions and unknown properties
	inputFile, err := os.Open("testdata/sanitize/sanitize_keep_extensions_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure to keep ALL extensions but remove unknown properties
	// Empty array = keep all extensions, nil = remove all
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns:     []string{}, // Empty array = keep ALL extensions
		KeepUnknownProperties: false,      // Remove unknown properties
		KeepUnusedComponents:  true,
	}

	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/sanitize/sanitize_keep_extensions_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Should keep extensions but remove unknown properties")
}
