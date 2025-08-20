package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInline_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/inline/inline_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure inlining options
	opts := openapi.InlineOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/inline_input.yaml",
		},
		RemoveUnusedComponents: true,
	}

	// Inline all references
	err = openapi.Inline(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the inlined document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/inline_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Inlined document should match expected output")
}

func TestInline_EmptyDocument(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with nil document
	err := openapi.Inline(ctx, nil, openapi.InlineOptions{})
	require.NoError(t, err)

	// Test with minimal document
	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	opts := openapi.InlineOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   doc,
			TargetLocation: "test.yaml",
		},
	}

	err = openapi.Inline(ctx, doc, opts)
	assert.NoError(t, err)
}
