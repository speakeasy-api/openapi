package openapi_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitize_ConfigReadAndFile_Success(t *testing.T) {
	t.Parallel()
	readErr := errors.New("config read failed")
	_, err := openapi.LoadSanitizeConfig(iotest.ErrReader(readErr))
	require.ErrorIs(t, err, readErr, "config reader failures should preserve their cause")
	path := filepath.Join(t.TempDir(), "sanitize.yaml")
	require.NoError(t, os.WriteFile(path, []byte("keepUnusedComponents: true\nextensionPatterns:\n  keep: [x-public-*]\n"), 0o600), "config fixture should be created")
	config, err := openapi.LoadSanitizeConfigFromFile(path)
	require.NoError(t, err, "config should load from disk")
	assert.Equal(t, &openapi.SanitizeOptions{KeepUnusedComponents: true, ExtensionPatterns: &openapi.ExtensionFilter{Keep: []string{"x-public-*"}}}, config, "file options should be parsed exactly")
}

func TestSanitize_InvalidAndUnmatchedPatterns_Success(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		filter   *openapi.ExtensionFilter
		keys     []string
		warnings []string
	}{
		{"whitelist", &openapi.ExtensionFilter{Keep: []string{"[", "x-absent", "x-keep"}}, []string{"x-keep"}, []string{"invalid keep pattern '[' was skipped", "keep pattern 'x-absent' did not match any extensions in the document"}},
		{"blacklist", &openapi.ExtensionFilter{Remove: []string{"[", "x-absent", "x-remove"}}, []string{"x-keep", "x-other"}, []string{"invalid remove pattern '[' was skipped", "remove pattern 'x-absent' did not match any extensions in the document"}},
		{"combined", &openapi.ExtensionFilter{Keep: []string{"[", "x-absent", "x-keep"}, Remove: []string{"[", "x-absent", "x-keep", "x-remove"}}, []string{"x-keep", "x-other"}, []string{"invalid keep pattern '[' was skipped", "keep pattern 'x-absent' did not match any extensions in the document", "invalid remove pattern '[' was skipped", "remove pattern 'x-absent' did not match any extensions in the document"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := unmarshalOpenAPI(t, t.Context(), "openapi: 3.1.0\ninfo: {title: Filters, version: '1'}\npaths: {}\nx-keep: true\nx-remove: true\nx-other: true\n")
			result, err := openapi.Sanitize(t.Context(), doc, &openapi.SanitizeOptions{ExtensionPatterns: tt.filter, KeepUnusedComponents: true, KeepUnknownProperties: true})
			require.NoError(t, err, "invalid patterns should produce warnings without failing sanitization")
			assert.ElementsMatch(t, tt.warnings, result.Warnings, "invalid and unmatched patterns should be reported separately")
			assert.ElementsMatch(t, tt.keys, slices.Collect(doc.Extensions.Keys()), "keep patterns should override matching remove patterns")
		})
	}
}

func TestSanitize_UnknownPropertiesInReusableObjects_Success(t *testing.T) {
	t.Parallel()
	const source = `openapi: 3.1.0
info: {title: Reusable objects, version: '1'}
paths: {}
components:
  schemas:
    Boolean: true
    String: {type: string, rogueField: rogue}
  responses:
    Object: {description: response, rogueField: rogue}
    Alias: {$ref: '#/components/responses/Object'}
  parameters:
    Object: {name: id, in: query, schema: {type: string}, rogueField: rogue}
    Alias: {$ref: '#/components/parameters/Object'}
  requestBodies:
    Object: {content: {application/json: {schema: {type: string}}}, rogueField: rogue}
    Alias: {$ref: '#/components/requestBodies/Object'}
  headers:
    Object: {schema: {type: string}, rogueField: rogue}
    Alias: {$ref: '#/components/headers/Object'}
  examples:
    Object: {value: example, rogueField: rogue}
    Alias: {$ref: '#/components/examples/Object'}
  links:
    Object: {operationId: getItem, rogueField: rogue}
    Alias: {$ref: '#/components/links/Object'}
  callbacks:
    Object: {}
    Alias: {$ref: '#/components/callbacks/Object'}
  pathItems:
    Object: {summary: Path}
    Alias: {$ref: '#/components/pathItems/Object'}
  securitySchemes:
    Object: {type: http, scheme: bearer, rogueField: rogue}
    Alias: {$ref: '#/components/securitySchemes/Object'}
`
	doc := unmarshalOpenAPI(t, t.Context(), source)
	_, err := openapi.Sanitize(t.Context(), doc, &openapi.SanitizeOptions{KeepUnusedComponents: true})
	require.NoError(t, err, "all reusable object kinds should support unknown-property cleanup")
	var output bytes.Buffer
	require.NoError(t, openapi.Marshal(t.Context(), doc, &output), "sanitized document should marshal")
	assert.NotContains(t, output.String(), "rogue", "unknown fields should be removed from every object")
	for _, section := range []string{"responses", "parameters", "requestBodies", "headers", "examples", "links", "callbacks", "pathItems", "securitySchemes"} {
		assert.Contains(t, output.String(), "#/components/"+section+"/Object", "aliases should remain untouched while their object definitions are sanitized")
	}
	assert.Contains(t, output.String(), "Boolean: true", "boolean schemas should survive cleanup")
}

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

	// Sanitize with pattern matching - only remove x-go-* extensions (blacklist mode)
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Remove: []string{"x-go-*"},
		},
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

	// Sanitize with multiple patterns (blacklist mode)
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Remove: []string{"x-go-*", "x-internal-*"},
		},
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
  remove:
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
	require.NotNil(t, opts.ExtensionPatterns)
	assert.Equal(t, []string{"x-go-*", "x-internal-*"}, opts.ExtensionPatterns.Remove)
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
  remove:
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
  remove:
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

	// Configure to keep ALL extensions using wildcard whitelist
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Keep: []string{"*"}, // Wildcard = keep ALL extensions
		},
		KeepUnknownProperties: false, // Remove unknown properties
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

func TestSanitize_WhitelistMode_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with various extensions
	inputFile, err := os.Open("testdata/sanitize/sanitize_pattern_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Sanitize with whitelist - only keep x-speakeasy-* extensions
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Keep: []string{"x-speakeasy-*"},
		},
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
	actualYAML := buf.String()

	// Verify that only x-speakeasy-* extensions remain
	assert.Contains(t, actualYAML, "x-speakeasy-", "Should contain x-speakeasy-* extensions")
	assert.NotContains(t, actualYAML, "x-go-", "Should not contain x-go-* extensions")
}

func TestSanitize_WhitelistOverridesBlacklist_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_multi_pattern_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Remove all x-speakeasy-* extensions EXCEPT x-speakeasy-schema-*
	// This demonstrates whitelist overriding blacklist for a narrower match
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Keep:   []string{"x-speakeasy-schema-*"}, // Keep only schema-related extensions
			Remove: []string{"x-speakeasy-*"},        // Remove all speakeasy extensions
		},
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.String()

	// Verify whitelist overrides blacklist
	// x-speakeasy-schema-* should be kept (matches whitelist)
	assert.Contains(t, actualYAML, "x-speakeasy-schema-version", "Should keep x-speakeasy-schema-version (whitelist)")
	assert.Contains(t, actualYAML, "x-speakeasy-schema-id", "Should keep x-speakeasy-schema-id (whitelist)")
	assert.Contains(t, actualYAML, "x-speakeasy-schema-name", "Should keep x-speakeasy-schema-name (whitelist)")

	// Other x-speakeasy-* should be removed (matches blacklist, not whitelist)
	assert.NotContains(t, actualYAML, "x-speakeasy-retries", "Should remove x-speakeasy-retries (blacklist, not in whitelist)")
	assert.NotContains(t, actualYAML, "x-speakeasy-pagination", "Should remove x-speakeasy-pagination (blacklist, not in whitelist)")
	assert.NotContains(t, actualYAML, "x-speakeasy-entity", "Should remove x-speakeasy-entity (blacklist, not in whitelist)")

	// Non-speakeasy extensions should remain (not affected by either pattern)
	assert.Contains(t, actualYAML, "x-go-", "Should keep x-go-* (not affected by patterns)")
	assert.Contains(t, actualYAML, "x-internal-", "Should keep x-internal-* (not affected by patterns)")
}

func TestSanitize_EmptyFilter_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Empty filter should remove all extensions
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns:    &openapi.ExtensionFilter{},
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.String()

	// Verify all extensions are removed
	assert.NotContains(t, actualYAML, "x-go-", "Should not contain any x-go-* extensions")
	assert.NotContains(t, actualYAML, "x-speakeasy-", "Should not contain any x-speakeasy-* extensions")
}

func TestSanitize_WildcardKeep_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Keep all extensions with wildcard
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Keep: []string{"*"},
		},
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Warnings, "Should not have warnings")

	// Marshal the sanitized document
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.String()

	// Verify all extensions are kept
	assert.Contains(t, actualYAML, "x-", "Should contain extensions")
}

func TestSanitize_InvalidKeepPattern_Warning(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Use invalid glob pattern in Keep
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Keep: []string{"x-[invalid-pattern"},
		},
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Warnings, "Should have warnings for invalid pattern")
	assert.Contains(t, result.Warnings[0], "invalid keep pattern", "Warning should mention invalid keep pattern")
}

func TestSanitize_InvalidRemovePattern_Warning(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/sanitize/sanitize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Use invalid glob pattern in Remove
	opts := &openapi.SanitizeOptions{
		ExtensionPatterns: &openapi.ExtensionFilter{
			Remove: []string{"x-[invalid-pattern"},
		},
		KeepUnusedComponents: true,
	}
	result, err := openapi.Sanitize(ctx, inputDoc, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Warnings, "Should have warnings for invalid pattern")
	assert.Contains(t, result.Warnings[0], "invalid remove pattern", "Warning should mention invalid remove pattern")
}
