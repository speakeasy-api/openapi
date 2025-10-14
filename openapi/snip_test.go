package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnip_RemoveOperation_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/snip/snip_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "input document should be valid")

	// Remove DELETE /users operation (also removes UnusedSchema via Clean)
	removed, err := openapi.Snip(ctx, inputDoc, []openapi.OperationIdentifier{
		{Path: "/users", Method: "DELETE"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, removed, "should remove 1 operation")

	// Marshal the snipped document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/snip/snip_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "snipped document should match expected output")
}

func TestSnip_RemoveByOperationID_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/snip/snip_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, _, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)

	// Remove by operation ID
	removed, err := openapi.Snip(ctx, inputDoc, []openapi.OperationIdentifier{
		{OperationID: "deleteAllUsers"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, removed, "should remove 1 operation by ID")

	// Verify the operation was removed
	usersPath, exists := inputDoc.Paths.Get("/users")
	require.True(t, exists, "should keep /users path")
	assert.Nil(t, usersPath.Object.GetOperation(openapi.HTTPMethodDelete), "DELETE should be removed")
	assert.NotNil(t, usersPath.Object.GetOperation(openapi.HTTPMethodGet), "GET should remain")
	assert.NotNil(t, usersPath.Object.GetOperation(openapi.HTTPMethodPost), "POST should remain")
}

func TestSnip_NonExistentOperation_NoError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	inputFile, err := os.Open("testdata/snip/snip_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, _, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)

	// Try to remove non-existent operations
	removed, err := openapi.Snip(ctx, inputDoc, []openapi.OperationIdentifier{
		{Path: "/nonexistent", Method: "GET"},
		{OperationID: "nonExistentID"},
	})

	require.NoError(t, err, "should not error on non-existent operations")
	assert.Equal(t, 0, removed, "should remove 0 operations")
}

func TestSnip_EmptyOperationList_NoError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	inputFile, err := os.Open("testdata/snip/snip_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, _, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)

	// Snip with empty list
	removed, err := openapi.Snip(ctx, inputDoc, []openapi.OperationIdentifier{})
	require.NoError(t, err)
	assert.Equal(t, 0, removed, "should remove 0 operations with empty list")
}

func TestSnip_NilDocument_Error(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	removed, err := openapi.Snip(ctx, nil, []openapi.OperationIdentifier{
		{Path: "/users", Method: "GET"},
	})

	require.Error(t, err, "should error on nil document")
	assert.Equal(t, 0, removed)
	assert.Contains(t, err.Error(), "document cannot be nil")
}
