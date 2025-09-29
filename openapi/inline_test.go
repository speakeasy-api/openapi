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
		OpenAPI: openapi.Version,
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

func TestInline_SiblingDirectories_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document from test subdirectory
	inputFile, err := os.Open("testdata/inline/test/openapi.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure inlining options
	opts := openapi.InlineOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/test/openapi.yaml",
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
	expectedBytes, err := os.ReadFile("testdata/inline/inlined_sibling_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Inlined document should match expected output")
}

func TestInline_AdditionalOperations_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with additionalOperations
	inputFile, err := os.Open("testdata/inline/additionaloperations_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure inlining options
	opts := openapi.InlineOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/additionaloperations_input.yaml",
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
	actualYAML := buf.String()

	// Verify that additionalOperations are preserved
	assert.Contains(t, actualYAML, "additionalOperations:", "additionalOperations should be preserved")

	// Verify that external references in additionalOperations were inlined
	assert.NotContains(t, actualYAML, "$ref:", "No references should remain after inlining")
	assert.NotContains(t, actualYAML, "external_custom_operations.yaml", "No external file references should remain")

	// Verify that the COPY operation has inlined content
	assert.Contains(t, actualYAML, "COPY:", "COPY operation should be present")
	assert.Contains(t, actualYAML, "operationId: copyResource", "COPY operation content should be inlined")

	// Verify that external parameter was inlined in COPY operation
	copyOperationSection := extractAdditionalOperationSection(actualYAML, "COPY")
	assert.Contains(t, copyOperationSection, "name: destination", "DestinationParam should be inlined")
	assert.Contains(t, copyOperationSection, "in: header", "DestinationParam should be inlined")

	// Verify that external request body was inlined in COPY operation
	assert.Contains(t, copyOperationSection, "source_path:", "CopyRequest schema should be inlined")
	assert.Contains(t, copyOperationSection, "destination_path:", "CopyRequest schema should be inlined")

	// Verify that the PURGE operation has inlined content
	assert.Contains(t, actualYAML, "PURGE:", "PURGE operation should be present")
	assert.Contains(t, actualYAML, "operationId: purgeResource", "PURGE operation content should be inlined")

	// Verify that external parameter was inlined in PURGE operation
	purgeOperationSection := extractAdditionalOperationSection(actualYAML, "PURGE")
	assert.Contains(t, purgeOperationSection, "name: X-Confirm-Purge", "ConfirmationParam should be inlined")
	assert.Contains(t, purgeOperationSection, "pattern: ^CONFIRM-[A-Z0-9]{8}$", "ConfirmationParam schema should be inlined")

	// Verify that the SYNC operation has inlined content
	assert.Contains(t, actualYAML, "SYNC:", "SYNC operation should be present")
	assert.Contains(t, actualYAML, "operationId: syncResource", "SYNC operation content should be inlined")

	// Verify that external schemas were inlined in SYNC operation
	syncOperationSection := extractAdditionalOperationSection(actualYAML, "SYNC")
	assert.Contains(t, syncOperationSection, "source:", "SyncConfig schema should be inlined")
	assert.Contains(t, syncOperationSection, "destination:", "SyncConfig schema should be inlined")
	assert.Contains(t, syncOperationSection, "sync_id:", "SyncResult schema should be inlined")
	assert.Contains(t, syncOperationSection, "files_synced:", "SyncResult schema should be inlined")

	// Verify that the BATCH operation has inlined content
	assert.Contains(t, actualYAML, "BATCH:", "BATCH operation should be present")
	assert.Contains(t, actualYAML, "operationId: batchProcess", "BATCH operation content should be inlined")

	// Verify that nested external schemas were properly inlined
	batchOperationSection := extractAdditionalOperationSection(actualYAML, "BATCH")
	assert.Contains(t, batchOperationSection, "parallel_execution:", "BatchConfig schema should be inlined")
	assert.Contains(t, batchOperationSection, "batch_id:", "BatchResult schema should be inlined")
	assert.Contains(t, batchOperationSection, "max_attempts:", "RetryPolicy schema should be inlined")

	// Verify components section was removed (since RemoveUnusedComponents is true)
	// Note: Some components might remain if they're still referenced from the main document
	if !assert.NotContains(t, actualYAML, "components:", "Components section should be removed after inlining") {
		// If components section exists, ensure it doesn't contain the external schemas
		assert.NotContains(t, actualYAML, "ResourceMetadata:", "External ResourceMetadata should not be in components after inlining")
		assert.NotContains(t, actualYAML, "SyncConfig:", "External SyncConfig should not be in components after inlining")
	}
}

// Helper function to extract a specific additionalOperation section from YAML
func extractAdditionalOperationSection(yamlContent, operationName string) string {
	lines := strings.Split(yamlContent, "\n")
	var sectionLines []string
	inTargetOperation := false
	indentLevel := -1

	for _, line := range lines {
		if strings.Contains(line, operationName+":") && strings.Contains(line, "additionalOperations") == false {
			inTargetOperation = true
			indentLevel = len(line) - len(strings.TrimLeft(line, " "))
			sectionLines = append(sectionLines, line)
			continue
		}

		if inTargetOperation {
			currentIndent := len(line) - len(strings.TrimLeft(line, " "))
			// If we hit a line at the same or lower indent level, we've left the operation
			if strings.TrimSpace(line) != "" && currentIndent <= indentLevel {
				break
			}
			sectionLines = append(sectionLines, line)
		}
	}

	return strings.Join(sectionLines, "\n")
}
