package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MultiFile_BasicReferences(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Read the root OpenAPI file
	data, err := os.ReadFile("testdata/multifile-basic/openapi.yaml")
	require.NoError(t, err, "failed to read openapi.yaml")

	// Unmarshal the document
	doc, validationErrs, err := openapi.Unmarshal(ctx, bytes.NewReader(data))
	require.NoError(t, err, "unmarshal should succeed")
	require.NotNil(t, doc, "document should not be nil")

	t.Logf("Unmarshal validation errors: %d", len(validationErrs))
	for _, verr := range validationErrs {
		t.Logf("  - %v", verr)
	}

	// Build index with proper resolve options
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "testdata/multifile-basic/openapi.yaml",
	}

	index := openapi.BuildIndex(ctx, doc, resolveOpts)
	require.NotNil(t, index, "index should not be nil")

	// Check for index errors
	indexErrors := index.GetAllErrors()
	t.Logf("Index errors: %d", len(indexErrors))
	for i, ierr := range indexErrors {
		t.Logf("  %d: %v", i+1, ierr)
	}

	// The key test: verify no "source is nil" errors
	sourceNilErrors := []error{}
	for _, ierr := range indexErrors {
		if assert.Contains(t, ierr.Error(), "source is nil") {
			sourceNilErrors = append(sourceNilErrors, ierr)
		}
	}

	if len(sourceNilErrors) > 0 {
		t.Errorf("Found %d 'source is nil' errors - this indicates reference resolution failure:", len(sourceNilErrors))
		for i, err := range sourceNilErrors {
			t.Logf("  %d: %v", i+1, err)
		}
		t.FailNow()
	}

	// Verify external components were indexed
	t.Logf("External schemas: %d", len(index.ExternalSchemas))
	t.Logf("External responses: %d", len(index.ExternalResponses))
	t.Logf("External headers: %d", len(index.ExternalHeaders))
	t.Logf("External links: %d", len(index.ExternalLinks))
	t.Logf("External callbacks: %d", len(index.ExternalCallbacks))
	t.Logf("External pathItems: %d", len(index.ExternalPathItems))

	// We should have external components from components.yaml
	assert.NotEmpty(t, index.ExternalSchemas, "should have external schemas")
	assert.NotEmpty(t, index.ExternalResponses, "should have external responses")
	assert.NotEmpty(t, index.ExternalHeaders, "should have external headers")
	assert.NotEmpty(t, index.ExternalLinks, "should have external links")
	assert.NotEmpty(t, index.ExternalCallbacks, "should have external callbacks")
	assert.NotEmpty(t, index.ExternalPathItems, "should have external pathItems")

	// Verify references can be resolved
	t.Run("references resolve successfully", func(t *testing.T) {
		t.Parallel()
		// Get the operation that has the external response reference
		require.NotNil(t, doc.Paths, "paths should not be nil")
		pathItemRef, found := doc.Paths.Get("/test")
		require.True(t, found, "should find /test path")
		require.NotNil(t, pathItemRef, "pathItem should not be nil")

		// Get the actual PathItem (may need to resolve if it's a reference)
		if pathItemRef.IsReference() {
			_, err := pathItemRef.Resolve(ctx, resolveOpts)
			require.NoError(t, err, "pathItem should resolve")
		}
		pathItem := pathItemRef.GetObject()
		require.NotNil(t, pathItem, "resolved pathItem should not be nil")
		getOp := pathItem.Get()
		require.NotNil(t, getOp, "should have GET operation")

		// Try to resolve the 200 response reference
		require.NotNil(t, getOp.Responses, "responses should not be nil")
		response200, found := getOp.Responses.Get("200")
		require.True(t, found, "should have 200 response")
		require.True(t, response200.IsReference(), "200 response should be a reference")

		// Resolve it
		validationErrs, err := response200.Resolve(ctx, resolveOpts)
		require.NoError(t, err, "resolving 200 response should not error")
		assert.Empty(t, validationErrs, "should have no validation errors")

		// Verify the resolved object
		resolvedResponse := response200.GetObject()
		require.NotNil(t, resolvedResponse, "resolved response should not be nil")
		assert.Equal(t, "Successful response", resolvedResponse.Description, "should have correct description")
	})
}

// Test the specific pattern: external file with internal component references
// This is failing because when resolving a component from an external file,
// internal references within that component (#/components/...) cannot be resolved
func Test_MultiFile_InternalReferencesInExternalFile(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	data, err := os.ReadFile("testdata/multifile-simple/openapi.yaml")
	require.NoError(t, err)

	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader(data))
	require.NoError(t, err)
	require.NotNil(t, doc)

	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "testdata/multifile-simple/openapi.yaml",
	}

	index := openapi.BuildIndex(ctx, doc, resolveOpts)
	require.NotNil(t, index)

	indexErrors := index.GetAllErrors()
	t.Logf("Index errors: %d", len(indexErrors))
	for i, ierr := range indexErrors {
		t.Logf("  %d: %v", i+1, ierr)
	}

	// This test demonstrates the bug:
	// When TestResponse (from components.yaml) contains an internal reference
	// to #/components/schemas/TestSchema, that reference cannot be resolved
	// because the external file is not unmarshalled as a full document
	for _, ierr := range indexErrors {
		if assert.Contains(t, ierr.Error(), "source is nil") {
			t.Logf("BUG CONFIRMED: %v", ierr)
		}
	}
}
