package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoin_Counter_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the second document
	secondFile, err := os.Open("testdata/join/subdir/second.yaml")
	require.NoError(t, err)
	defer secondFile.Close()

	secondDoc, validationErrs, err := openapi.Unmarshal(ctx, secondFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Second document should be valid")

	// Load the third document
	thirdFile, err := os.Open("testdata/join/third.yaml")
	require.NoError(t, err)
	defer thirdFile.Close()

	thirdDoc, validationErrs, err := openapi.Unmarshal(ctx, thirdFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Third document should be valid")

	// Configure join options with counter strategy
	documents := []openapi.JoinDocumentInfo{
		{
			Document: secondDoc,
			FilePath: "subdir/second.yaml",
		},
		{
			Document: thirdDoc,
			FilePath: "third.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	// Join documents
	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Marshal the joined document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, mainDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/join/joined_counter_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Joined document with counter strategy should match expected output")
}

func TestJoin_FilePath_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the second document
	secondFile, err := os.Open("testdata/join/subdir/second.yaml")
	require.NoError(t, err)
	defer secondFile.Close()

	secondDoc, validationErrs, err := openapi.Unmarshal(ctx, secondFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Second document should be valid")

	// Load the third document
	thirdFile, err := os.Open("testdata/join/third.yaml")
	require.NoError(t, err)
	defer thirdFile.Close()

	thirdDoc, validationErrs, err := openapi.Unmarshal(ctx, thirdFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Third document should be valid")

	// Configure join options with filepath strategy
	documents := []openapi.JoinDocumentInfo{
		{
			Document: secondDoc,
			FilePath: "subdir/second.yaml",
		},
		{
			Document: thirdDoc,
			FilePath: "third.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictFilePath,
	}

	// Join documents
	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Marshal the joined document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, mainDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/join/joined_filepath_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Joined document with filepath strategy should match expected output")
}

func TestJoin_EmptyDocuments_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Store original values for comparison
	originalTitle := mainDoc.Info.Title
	originalVersion := mainDoc.Info.Version

	// Join with empty documents slice
	documents := []openapi.JoinDocumentInfo{}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Main document should remain unchanged
	assert.Equal(t, originalTitle, mainDoc.Info.Title)
	assert.Equal(t, originalVersion, mainDoc.Info.Version)
}

func TestJoin_NilMainDocument_Error(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	documents := []openapi.JoinDocumentInfo{}
	opts := openapi.JoinOptions{}

	err := openapi.Join(ctx, nil, documents, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "main document is nil")
}

func TestJoin_NilDocumentInSlice_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Include nil document in slice
	documents := []openapi.JoinDocumentInfo{
		{
			Document: nil,
			FilePath: "nil.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	// Should not error, just skip nil documents
	err = openapi.Join(ctx, mainDoc, documents, opts)
	assert.NoError(t, err)
}

func TestJoin_NoFilePath_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the second document
	secondFile, err := os.Open("testdata/join/subdir/second.yaml")
	require.NoError(t, err)
	defer secondFile.Close()

	secondDoc, validationErrs, err := openapi.Unmarshal(ctx, secondFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Second document should be valid")

	// Join documents without file path
	documents := []openapi.JoinDocumentInfo{
		{
			Document: secondDoc,
			FilePath: "", // Empty file path
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Verify original /users path exists
	assert.True(t, mainDoc.Paths.Has("/users"))

	// Verify conflicting path uses fallback name
	assert.True(t, mainDoc.Paths.Has("/users#document_0"))
}

func TestJoin_ServersSecurityConflicts_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the conflict servers document
	conflictServersFile, err := os.Open("testdata/join/conflict_servers.yaml")
	require.NoError(t, err)
	defer conflictServersFile.Close()

	conflictServersDoc, validationErrs, err := openapi.Unmarshal(ctx, conflictServersFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Conflict servers document should be valid")

	// Load the conflict security document
	conflictSecurityFile, err := os.Open("testdata/join/conflict_security.yaml")
	require.NoError(t, err)
	defer conflictSecurityFile.Close()

	conflictSecurityDoc, validationErrs, err := openapi.Unmarshal(ctx, conflictSecurityFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Conflict security document should be valid")

	// Configure join options with counter strategy
	documents := []openapi.JoinDocumentInfo{
		{
			Document: conflictServersDoc,
			FilePath: "conflict_servers.yaml",
		},
		{
			Document: conflictSecurityDoc,
			FilePath: "conflict_security.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Marshal the result to YAML for comparison
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, mainDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Read expected output
	expectedBytes, err := os.ReadFile("testdata/join/joined_conflicts_expected.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expectedBytes), string(actualYAML), "Joined document with server/security conflicts should match expected output")
}
