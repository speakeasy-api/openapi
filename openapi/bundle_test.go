package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundle_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/inline/inline_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/inline_input.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/bundled_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document should match expected output")
}

func TestBundle_CounterNaming_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/inline/inline_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options with counter naming
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/inline_input.yaml",
		},
		NamingStrategy: openapi.BundleNamingCounter,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/bundled_counter_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document with counter naming should match expected output")
}

func TestBundle_EmptyDocument(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with nil document
	err := openapi.Bundle(ctx, nil, openapi.BundleOptions{})
	require.NoError(t, err)

	// Test with minimal document
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   doc,
			TargetLocation: "test.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	err = openapi.Bundle(ctx, doc, opts)
	require.NoError(t, err)

	// Document should remain unchanged
	assert.Equal(t, openapi.Version, doc.OpenAPI)
	assert.Equal(t, "Empty API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)

	// No components should be added
	if doc.Components != nil && doc.Components.Schemas != nil {
		assert.Equal(t, 0, doc.Components.Schemas.Len(), "No schemas should be added for document without external references")
	}
}

func TestBundle_SiblingDirectories_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with sibling directory references
	inputFile, err := os.Open("testdata/inline/test/openapi.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/test/openapi.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/bundled_sibling_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document should match expected output")
}
