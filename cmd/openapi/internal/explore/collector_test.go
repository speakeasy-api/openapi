package explore

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectOperations_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a simple test OpenAPI document
	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}

	// Add a simple path with operations
	pathItem := openapi.NewPathItem()

	// Add GET operation
	getOp := &openapi.Operation{
		OperationID: strPtr("getUsers"),
		Summary:     strPtr("Get all users"),
		Description: strPtr("Returns a list of all users"),
		Tags:        []string{"users"},
	}
	pathItem.Set(openapi.HTTPMethodGet, getOp)

	// Add POST operation
	postOp := &openapi.Operation{
		OperationID: strPtr("createUser"),
		Summary:     strPtr("Create a user"),
		Description: strPtr("Creates a new user"),
		Tags:        []string{"users"},
	}
	pathItem.Set(openapi.HTTPMethodPost, postOp)

	refPathItem := &openapi.ReferencedPathItem{
		Object: pathItem,
	}
	doc.Paths.Set("/users", refPathItem)

	// Collect operations
	operations, err := CollectOperations(ctx, doc)
	require.NoError(t, err, "should collect operations without error")

	// Verify results
	assert.Len(t, operations, 2, "should collect 2 operations")

	// Check first operation (GET, should come before POST due to sorting)
	assert.Equal(t, "GET", operations[0].Method)
	assert.Equal(t, "/users", operations[0].Path)
	assert.Equal(t, "getUsers", operations[0].OperationID)
	assert.Equal(t, "Get all users", operations[0].Summary)
	assert.Equal(t, "Returns a list of all users", operations[0].Description)
	assert.Equal(t, []string{"users"}, operations[0].Tags)
	assert.True(t, operations[0].Folded, "should start folded")

	// Check second operation
	assert.Equal(t, "POST", operations[1].Method)
	assert.Equal(t, "/users", operations[1].Path)
	assert.Equal(t, "createUser", operations[1].OperationID)
	assert.Equal(t, "Create a user", operations[1].Summary)
}

func TestCollectOperations_MultiplePathsSorted(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}

	// Add paths in non-alphabetical order
	paths := []string{"/users", "/pets", "/admin"}
	for _, path := range paths {
		pathItem := openapi.NewPathItem()
		op := &openapi.Operation{
			Summary: strPtr("Operation for " + path),
		}
		pathItem.Set(openapi.HTTPMethodGet, op)
		refPathItem := &openapi.ReferencedPathItem{Object: pathItem}
		doc.Paths.Set(path, refPathItem)
	}

	operations, err := CollectOperations(ctx, doc)
	require.NoError(t, err)

	// Verify operations are sorted by path
	assert.Equal(t, "/admin", operations[0].Path)
	assert.Equal(t, "/pets", operations[1].Path)
	assert.Equal(t, "/users", operations[2].Path)
}

func TestCollectOperations_EmptyDocument(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}

	operations, err := CollectOperations(ctx, doc)
	require.NoError(t, err)
	assert.Empty(t, operations, "should return empty slice for document with no operations")
}

func TestCollectOperations_DeprecatedOperation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}

	pathItem := openapi.NewPathItem()
	op := &openapi.Operation{
		OperationID: strPtr("deprecatedOp"),
		Deprecated:  boolPtr(true),
	}
	pathItem.Set(openapi.HTTPMethodGet, op)
	refPathItem := &openapi.ReferencedPathItem{Object: pathItem}
	doc.Paths.Set("/deprecated", refPathItem)

	operations, err := CollectOperations(ctx, doc)
	require.NoError(t, err)
	require.Len(t, operations, 1)

	assert.True(t, operations[0].Deprecated, "should capture deprecated status")
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
