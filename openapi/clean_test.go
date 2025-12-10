package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClean_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/clean/clean_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Clean unused components
	err = openapi.Clean(ctx, inputDoc)
	require.NoError(t, err)

	// Marshal the cleaned document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/clean/clean_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Cleaned document should match expected output")
}

func TestClean_RemoveAllUnusedComponents_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with all unused components
	inputFile, err := os.Open("testdata/clean/clean_empty_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Clean unused components
	err = openapi.Clean(ctx, inputDoc)
	require.NoError(t, err)

	// Marshal the cleaned document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/clean/clean_empty_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Document with all unused components should have components removed entirely")
}

func TestClean_EmptyDocument_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with nil document
	err := openapi.Clean(ctx, nil)
	require.NoError(t, err)

	// Test with minimal document (no components)
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	err = openapi.Clean(ctx, doc)
	require.NoError(t, err)
}

func TestClean_NoComponents_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with document that has no components section
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "API without components",
			Version: "1.0.0",
		},
		Paths: &openapi.Paths{},
	}

	err := openapi.Clean(ctx, doc)
	require.NoError(t, err)
	assert.Nil(t, doc.Components, "Components should remain nil")
}
